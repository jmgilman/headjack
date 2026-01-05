package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/slogger"
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

		mgr, err := requireManager(cmd.Context())
		if err != nil {
			return err
		}

		inst, err := getInstanceByBranch(cmd.Context(), mgr, branch)
		if err != nil {
			return err
		}

		// Stop the instance
		if err := mgr.Stop(cmd.Context(), inst.ID); err != nil {
			return fmt.Errorf("stop instance: %w", err)
		}

		slogger.L(cmd.Context()).Info("stopped instance", "id", inst.ID, "branch", inst.Branch)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
