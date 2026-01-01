package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/instance"
)

var attachCmd = &cobra.Command{
	Use:   "attach [branch] [session]",
	Short: "Attach to an existing session",
	Long: `Attach to an existing session using most-recently-used (MRU) selection.

The command uses an MRU strategy:
  - No arguments: attach to the most recently accessed session across all instances
  - Branch only: attach to the most recently accessed session for that instance
  - Branch and session: attach to the specified session

If no sessions exist for the resolved scope, the command errors with a message
suggesting 'hjk run' to create one.

To detach from a session without terminating it, use the Zellij keybinding
(default: Ctrl+O, d). This returns you to your host terminal while the
session continues running.`,
	Example: `  # Attach to whatever you were last working on
  hjk attach

  # Attach to most recent session in feat/auth
  hjk attach feat/auth

  # Attach to specific session
  hjk attach feat/auth claude-main`,
	Args: cobra.MaximumNArgs(2),
	RunE: runAttachCmd,
}

func runAttachCmd(cmd *cobra.Command, args []string) error {
	mgr := ManagerFromContext(cmd.Context())

	switch len(args) {
	case 0:
		return attachGlobalMRU(cmd, mgr)
	case 1:
		return attachInstanceMRU(cmd, mgr, args[0])
	default:
		return attachExplicitSession(cmd, mgr, args[0], args[1])
	}
}

// attachGlobalMRU attaches to the most recently accessed session across all instances.
func attachGlobalMRU(cmd *cobra.Command, mgr *instance.Manager) error {
	globalMRU, err := mgr.GetGlobalMRUSession(cmd.Context())
	if err != nil {
		if errors.Is(err, instance.ErrNoSessionsAvailable) {
			return errors.New("no sessions exist (use 'hjk run' to create one)")
		}
		return fmt.Errorf("get global MRU session: %w", err)
	}

	return mgr.AttachSession(cmd.Context(), globalMRU.InstanceID, globalMRU.Session.Name)
}

// attachInstanceMRU attaches to the most recently accessed session for a specific instance.
func attachInstanceMRU(cmd *cobra.Command, mgr *instance.Manager, branch string) error {
	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	inst, err := mgr.GetByBranch(cmd.Context(), repoPath, branch)
	if err != nil {
		if errors.Is(err, instance.ErrNotFound) {
			return fmt.Errorf("no instance found for branch %q (use 'hjk run' to create one)", branch)
		}
		return fmt.Errorf("get instance: %w", err)
	}

	session, err := mgr.GetMRUSession(cmd.Context(), inst.ID)
	if err != nil {
		if errors.Is(err, instance.ErrNoSessionsAvailable) {
			return fmt.Errorf("no sessions exist for branch %q (use 'hjk run' to create one)", branch)
		}
		return fmt.Errorf("get MRU session: %w", err)
	}

	return mgr.AttachSession(cmd.Context(), inst.ID, session.Name)
}

// attachExplicitSession attaches to a specific session by name.
func attachExplicitSession(cmd *cobra.Command, mgr *instance.Manager, branch, sessionName string) error {
	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	inst, err := mgr.GetByBranch(cmd.Context(), repoPath, branch)
	if err != nil {
		if errors.Is(err, instance.ErrNotFound) {
			return fmt.Errorf("no instance found for branch %q (use 'hjk run' to create one)", branch)
		}
		return fmt.Errorf("get instance: %w", err)
	}

	// Verify session exists
	_, err = mgr.GetSession(cmd.Context(), inst.ID, sessionName)
	if err != nil {
		if errors.Is(err, instance.ErrSessionNotFound) {
			return fmt.Errorf("session %q not found in instance for branch %q", sessionName, branch)
		}
		return fmt.Errorf("get session: %w", err)
	}

	return mgr.AttachSession(cmd.Context(), inst.ID, sessionName)
}

func init() {
	rootCmd.AddCommand(attachCmd)
}
