package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var killCmd = &cobra.Command{
	Use:   "kill <branch>/<session>",
	Short: "Kill a specific session",
	Long: `Kill a specific session.

Terminates the multiplexer session and removes it from the catalog.
The instance and other sessions are unaffected.

The argument must be in the format <branch>/<session>, where branch is
the instance branch name and session is the session name.`,
	Example: `  # Kill a session
  headjack kill feat/auth/debug-shell
  headjack kill main/claude-experiment`,
	Args: cobra.ExactArgs(1),
	RunE: runKillCmd,
}

func runKillCmd(cmd *cobra.Command, args []string) error {
	branch, sessionName, err := parseBranchSession(args[0])
	if err != nil {
		return err
	}

	mgr, err := requireManager(cmd.Context())
	if err != nil {
		return err
	}

	inst, err := getInstanceByBranch(cmd.Context(), mgr, branch)
	if err != nil {
		return err
	}

	// Kill the session
	if err := mgr.KillSession(cmd.Context(), inst.ID, sessionName); err != nil {
		return fmt.Errorf("kill session: %w", err)
	}

	fmt.Printf("Killed session %s in instance %s\n", sessionName, branch)
	return nil
}

// parseBranchSession parses a "branch/session" argument.
// The branch can contain slashes (e.g., "feat/auth"), so we split on the last slash.
func parseBranchSession(arg string) (branch, session string, err error) {
	lastSlash := strings.LastIndex(arg, "/")
	if lastSlash == -1 {
		return "", "", fmt.Errorf("invalid format: expected <branch>/<session>, got %q", arg)
	}

	branch = arg[:lastSlash]
	session = arg[lastSlash+1:]

	if branch == "" {
		return "", "", errors.New("invalid format: branch cannot be empty")
	}
	if session == "" {
		return "", "", errors.New("invalid format: session cannot be empty")
	}

	return branch, session, nil
}

func init() {
	rootCmd.AddCommand(killCmd)
}
