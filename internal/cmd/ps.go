package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/exec"
	"github.com/jmgilman/headjack/internal/git"
	"github.com/jmgilman/headjack/internal/instance"
	"github.com/jmgilman/headjack/internal/slogger"
)

var psCmd = &cobra.Command{
	Use:   "ps [branch]",
	Short: "List instances or sessions",
	Long: `List instances or sessions managed by Headjack.

By default, lists instances for the current repository.

If a branch is specified, lists sessions for that instance instead.

Use --all to list instances across all repositories (only applies when
listing instances, not sessions).`,
	Example: `  # List instances for current repo
  headjack ps

  # List all instances across all repos
  headjack ps --all

  # List sessions for a specific instance
  headjack ps feat/auth`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPsCmd,
}

func runPsCmd(cmd *cobra.Command, args []string) error {
	// If a branch is specified, list sessions for that instance
	if len(args) == 1 {
		return listSessions(cmd, args[0])
	}

	// Otherwise, list instances
	return listInstances(cmd)
}

func listInstances(cmd *cobra.Command) error {
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return fmt.Errorf("get all flag: %w", err)
	}

	mgr, err := requireManager(cmd.Context())
	if err != nil {
		return err
	}

	filter := instance.ListFilter{}

	// If not showing all, filter by current repo
	if !all {
		repoPathValue, pathErr := repoPath()
		if pathErr != nil {
			return pathErr
		}

		opener := git.NewOpener(exec.New())
		repo, openErr := opener.Open(cmd.Context(), repoPathValue)
		if openErr != nil {
			return fmt.Errorf("open repository: %w", openErr)
		}

		filter.RepoID = repo.Identifier()
	}

	instances, err := mgr.List(cmd.Context(), filter)
	if err != nil {
		return fmt.Errorf("list instances: %w", err)
	}

	if len(instances) == 0 {
		if all {
			slogger.L(cmd.Context()).Info("no instances found")
		} else {
			slogger.L(cmd.Context()).Info("no instances found for this repository")
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "BRANCH\tSTATUS\tSESSIONS\tCREATED"); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for i := range instances {
		inst := &instances[i]
		sessionCount, countErr := getSessionCount(cmd, mgr, inst.ID)
		if countErr != nil {
			// Best effort - show 0 if we can't get the count
			sessionCount = 0
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			inst.Branch,
			inst.Status,
			sessionCount,
			formatTimeAgo(inst.CreatedAt),
		); err != nil {
			return fmt.Errorf("write instance: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	return nil
}

func listSessions(cmd *cobra.Command, branch string) error {
	mgr, err := requireManager(cmd.Context())
	if err != nil {
		return err
	}

	inst, err := getInstanceByBranch(cmd.Context(), mgr, branch)
	if err != nil {
		return err
	}

	sessions, err := mgr.ListSessions(cmd.Context(), inst.ID)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	if len(sessions) == 0 {
		slogger.L(cmd.Context()).Info("no sessions found", "branch", branch)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "SESSION\tTYPE\tSTATUS\tCREATED\tACCESSED"); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for _, sess := range sessions {
		// Sessions in headjack are always detached when not actively attached
		status := "detached"
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			sess.Name,
			sess.Type,
			status,
			formatTimeAgo(sess.CreatedAt),
			formatTimeAgo(sess.LastAccessed),
		); err != nil {
			return fmt.Errorf("write session: %w", err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	return nil
}

func getSessionCount(cmd *cobra.Command, mgr *instance.Manager, instanceID string) (int, error) {
	sessions, err := mgr.ListSessions(cmd.Context(), instanceID)
	if err != nil {
		return 0, err
	}
	return len(sessions), nil
}

// formatTimeAgo formats a time as a human-readable relative time.
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	case d < 30*24*time.Hour:
		weeks := int(d.Hours() / 24 / 7)
		if weeks == 1 {
			return "1w ago"
		}
		return fmt.Sprintf("%dw ago", weeks)
	default:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1mo ago"
		}
		return fmt.Sprintf("%dmo ago", months)
	}
}

func init() {
	rootCmd.AddCommand(psCmd)

	psCmd.Flags().BoolP("all", "a", false, "list instances across all repositories")
}
