package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/instance"
)

var stopCmd = &cobra.Command{
	Use:   "stop <branch>",
	Short: "Stop a running instance's container",
	Long: `Stop the container associated with the specified instance.

The worktree is preserved and the instance can be resumed later with 'hjk run'.`,
	Example: `  hjk stop feat/auth`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]

		// Get current working directory as repo path
		repoPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		// Get instance by branch
		mgr := ManagerFromContext(cmd.Context())
		inst, err := mgr.GetByBranch(cmd.Context(), repoPath, branch)
		if err != nil {
			if errors.Is(err, instance.ErrNotFound) {
				return fmt.Errorf("no instance found for branch %q", branch)
			}
			return fmt.Errorf("get instance: %w", err)
		}

		// Stop the instance
		if err := mgr.Stop(cmd.Context(), inst.ID); err != nil {
			return fmt.Errorf("stop instance: %w", err)
		}

		fmt.Printf("Stopped instance %s for branch %s\n", inst.ID, inst.Branch)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
