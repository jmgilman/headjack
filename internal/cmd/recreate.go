package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var recreateCmd = &cobra.Command{
	Use:   "recreate <branch>",
	Short: "Recreate an instance's container without losing worktree state",
	Long: `Recreate the container for an instance while preserving the worktree.

This command:
- Stops and deletes the existing container
- Creates a new container with the same worktree

Useful when the container environment is corrupted or needs a fresh state.
The worktree (and all git-tracked and untracked files) is preserved.`,
	Example: `  # Recreate with same image
  headjack recreate feat/auth

  # Recreate with new image
  headjack recreate feat/auth --base my-registry.io/new-image:v2`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]

		mgr, err := requireManager(cmd.Context())
		if err != nil {
			return err
		}

		inst, err := getInstanceByBranch(cmd.Context(), mgr, branch, "no instance found for branch %q")
		if err != nil {
			return err
		}

		// Determine image to use (precedence: flag > config)
		// Config already has defaults set via Viper, so just use it
		image, err := cmd.Flags().GetString("base")
		if err != nil {
			return fmt.Errorf("get base flag: %w", err)
		}
		if image == "" {
			if cfg := ConfigFromContext(cmd.Context()); cfg != nil {
				image = cfg.Default.BaseImage
			}
		}

		// Recreate the instance
		newInst, err := mgr.Recreate(cmd.Context(), inst.ID, image)
		if err != nil {
			return fmt.Errorf("recreate instance: %w", err)
		}

		fmt.Printf("Recreated instance %s for branch %s with image %s\n", newInst.ID, newInst.Branch, image)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(recreateCmd)

	recreateCmd.Flags().String("base", "", "use a different base image")
}
