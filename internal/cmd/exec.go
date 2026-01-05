package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/instance"
)

var execCmd = &cobra.Command{
	Use:   "exec <branch> [command...]",
	Short: "Execute a command in an instance's container",
	Long: `Execute a command within an existing instance's container.

By default, a new tmux session is created and attached. Use --no-mux to bypass
the multiplexer and execute the command directly (like 'docker exec').

If no command is specified, the default shell is started.

All session output is captured to a log file (when using tmux).`,
	Example: `  # Start a shell session in tmux
  hjk exec feat/auth

  # Run a command in tmux
  hjk exec feat/auth npm test

  # Run a command directly (bypass tmux)
  hjk exec feat/auth --no-mux ls -la

  # Start shell directly (bypass tmux)
  hjk exec feat/auth --no-mux

  # Run with custom session name
  hjk exec feat/auth --name build-session npm run build`,
	Args:               cobra.MinimumNArgs(1),
	RunE:               runExecCmd,
	DisableFlagParsing: false,
}

// execFlags holds parsed flags for the exec command.
type execFlags struct {
	noMux       bool
	sessionName string
	detached    bool
}

// parseExecFlags extracts and validates flags from the command.
func parseExecFlags(cmd *cobra.Command) (*execFlags, error) {
	noMux, err := cmd.Flags().GetBool("no-mux")
	if err != nil {
		return nil, fmt.Errorf("get no-mux flag: %w", err)
	}
	sessionName, err := cmd.Flags().GetString("name")
	if err != nil {
		return nil, fmt.Errorf("get name flag: %w", err)
	}
	detached, err := cmd.Flags().GetBool("detached")
	if err != nil {
		return nil, fmt.Errorf("get detached flag: %w", err)
	}

	return &execFlags{
		noMux:       noMux,
		sessionName: sessionName,
		detached:    detached,
	}, nil
}

func runExecCmd(cmd *cobra.Command, args []string) error {
	branch := args[0]
	cmdArgs := args[1:] // May be empty (means shell)

	mgr, err := requireManager(cmd.Context())
	if err != nil {
		return err
	}

	flags, err := parseExecFlags(cmd)
	if err != nil {
		return err
	}

	// Get existing instance (do NOT create)
	inst, err := getInstanceByBranch(cmd.Context(), mgr, branch)
	if err != nil {
		return err
	}

	if flags.noMux {
		// Direct execution (bypasses multiplexer entirely)
		return mgr.Attach(cmd.Context(), inst.ID, instance.AttachConfig{
			Command:     cmdArgs, // Empty = shell
			Interactive: true,
		})
	}

	// Create tmux session for the command
	sessionCfg := &instance.CreateSessionConfig{
		Type:    "shell",
		Name:    flags.sessionName,
		Command: cmdArgs, // Empty = shell
	}

	session, err := mgr.CreateSession(cmd.Context(), inst.ID, sessionCfg)
	if err != nil {
		if errors.Is(err, instance.ErrSessionExists) {
			return fmt.Errorf("session %q already exists in instance %s", flags.sessionName, inst.ID)
		}
		var notRunningErr *instance.NotRunningError
		if errors.As(err, &notRunningErr) {
			hint := formatExecNotRunningHint(cmd, notRunningErr)
			if hint != "" {
				return fmt.Errorf("create session: %w\nhint: %s", err, hint)
			}
		}
		return fmt.Errorf("create session: %w", err)
	}

	if flags.detached {
		fmt.Printf("Created session %s in instance %s (detached)\n", session.Name, inst.ID)
		return nil
	}

	return mgr.AttachSession(cmd.Context(), inst.ID, session.Name)
}

func formatExecNotRunningHint(cmd *cobra.Command, err *instance.NotRunningError) string {
	return formatNotRunningHint(cmd, err)
}

func init() {
	rootCmd.AddCommand(execCmd)

	execCmd.Flags().Bool("no-mux", false, "bypass tmux and execute directly")
	execCmd.Flags().StringP("name", "n", "", "override auto-generated session name (ignored with --no-mux)")
	execCmd.Flags().BoolP("detached", "d", false, "create session but don't attach (ignored with --no-mux)")
}
