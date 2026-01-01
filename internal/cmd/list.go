package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/exec"
	"github.com/jmgilman/headjack/internal/git"
	"github.com/jmgilman/headjack/internal/instance"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List instances",
	Long: `List instances managed by Headjack.

By default, lists instances for the current repository only.
Use --all to list instances across all repositories.`,
	Example: `  # List instances for current repo
  headjack list

  # List all instances across all repos
  headjack list --all`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return fmt.Errorf("get all flag: %w", err)
		}

		filter := instance.ListFilter{}

		// If not showing all, filter by current repo
		if !all {
			repoPath, wdErr := os.Getwd()
			if wdErr != nil {
				return fmt.Errorf("get working directory: %w", wdErr)
			}

			opener := git.NewOpener(exec.New())
			repo, openErr := opener.Open(cmd.Context(), repoPath)
			if openErr != nil {
				return fmt.Errorf("open repository: %w", openErr)
			}

			filter.RepoID = repo.Identifier()
		}

		mgr := ManagerFromContext(cmd.Context())
		instances, err := mgr.List(cmd.Context(), filter)
		if err != nil {
			return fmt.Errorf("list instances: %w", err)
		}

		if len(instances) == 0 {
			if all {
				fmt.Println("No instances found")
			} else {
				fmt.Println("No instances found for this repository")
			}
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintln(w, "ID\tBRANCH\tSTATUS\tREPO"); err != nil {
			return fmt.Errorf("write header: %w", err)
		}
		for _, inst := range instances {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", inst.ID, inst.Branch, inst.Status, inst.RepoID); err != nil {
				return fmt.Errorf("write instance: %w", err)
			}
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("flush output: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolP("all", "a", false, "list instances across all repositories")
}
