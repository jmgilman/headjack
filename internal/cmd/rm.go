package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <branch>",
	Short: "Remove an instance entirely",
	Long: `Remove an instance, including its container and git worktree.

This command:
- Stops the container if running
- Deletes the container
- Deletes the git worktree
- Removes the instance from the catalog

WARNING: This deletes uncommitted work in the worktree.`,
	Example: `  # Remove with confirmation prompt
  headjack rm feat/auth

  # Force remove without confirmation
  headjack rm feat/auth --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return fmt.Errorf("get force flag: %w", err)
		}

		mgr, err := requireManager(cmd.Context())
		if err != nil {
			return err
		}

		inst, err := getInstanceByBranch(cmd.Context(), mgr, branch)
		if err != nil {
			return err
		}

		// Confirm removal unless --force
		if !force {
			fmt.Printf("This will remove instance %s for branch %s.\n", inst.ID, inst.Branch)
			fmt.Printf("Worktree at %s will be deleted.\n", inst.Worktree)
			fmt.Print("Are you sure? [y/N] ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Canceled")
				return nil
			}
		}

		// Remove the instance
		if err := mgr.Remove(cmd.Context(), inst.ID); err != nil {
			return fmt.Errorf("remove instance: %w", err)
		}

		fmt.Printf("Removed instance %s for branch %s\n", inst.ID, inst.Branch)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)

	rmCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
}
