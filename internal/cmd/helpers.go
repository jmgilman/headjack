package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/config"
	"github.com/jmgilman/headjack/internal/instance"
)

func requireManager(ctx context.Context) (*instance.Manager, error) {
	mgr := ManagerFromContext(ctx)
	if mgr == nil {
		return nil, errors.New("instance manager not initialized")
	}
	return mgr, nil
}

func repoPath() (string, error) {
	path, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return path, nil
}

func defaultDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, config.DefaultDataDir), nil
}

func resolveBaseImage(ctx context.Context, override string) string {
	if override != "" {
		return override
	}
	if cfg := ConfigFromContext(ctx); cfg != nil {
		return cfg.Default.BaseImage
	}
	return ""
}

// runtimeLogsCommand returns the command to view container logs for the given runtime.
func runtimeLogsCommand(runtimeName, containerID string) string {
	switch runtimeName {
	case runtimeNameDocker:
		return "docker logs " + containerID
	default:
		return "podman logs " + containerID
	}
}

// formatNotRunningHint formats a hint for when a container is not running.
func formatNotRunningHint(cmd *cobra.Command, err *instance.NotRunningError) string {
	if err == nil || err.ContainerID == "" {
		return ""
	}
	runtimeName := runtimeNameDocker
	if cfg := ConfigFromContext(cmd.Context()); cfg != nil && cfg.Runtime.Name != "" {
		runtimeName = cfg.Runtime.Name
	}
	logsCmd := runtimeLogsCommand(runtimeName, err.ContainerID)
	if logsCmd == "" {
		return fmt.Sprintf("container %s is %s", err.ContainerID, err.Status)
	}
	return fmt.Sprintf("container %s is %s; check logs with `%s`", err.ContainerID, err.Status, logsCmd)
}

// parsePassthroughArgs extracts arguments after the -- separator from a cobra command.
// Returns nil if no -- separator was used.
func parsePassthroughArgs(cmd *cobra.Command, args []string) []string {
	dashIdx := cmd.ArgsLenAtDash()
	if dashIdx < 0 || dashIdx >= len(args) {
		return nil
	}
	return args[dashIdx:]
}

// mergeFlags combines base flags with additional flags.
// Base flags come first, additional flags are appended.
// Returns nil if both inputs are empty.
func mergeFlags(base, additional []string) []string {
	if len(base) == 0 && len(additional) == 0 {
		return nil
	}
	result := make([]string, 0, len(base)+len(additional))
	result = append(result, base...)
	result = append(result, additional...)
	return result
}

// getInstanceByBranch gets an existing instance by branch, returning an error with hint if not found.
// If the instance is stopped, it will be automatically restarted.
func getInstanceByBranch(ctx context.Context, mgr *instance.Manager, branch string) (*instance.Instance, error) {
	repoPath, err := repoPath()
	if err != nil {
		return nil, err
	}

	inst, err := mgr.GetByBranch(ctx, repoPath, branch)
	if err != nil {
		if errors.Is(err, instance.ErrNotFound) {
			return nil, fmt.Errorf("no instance found for branch %q\nhint: run 'hjk run %s' to create one", branch, branch)
		}
		return nil, fmt.Errorf("get instance: %w", err)
	}

	// Auto-restart if stopped
	if inst.Status == instance.StatusStopped {
		if startErr := mgr.Start(ctx, inst.ID); startErr != nil {
			return nil, fmt.Errorf("start stopped instance: %w", startErr)
		}
		fmt.Printf("Restarted instance %s for branch %s\n", inst.ID, inst.Branch)
		// Refresh the instance to get updated status
		inst, err = mgr.GetByBranch(ctx, repoPath, branch)
		if err != nil {
			return nil, fmt.Errorf("get restarted instance: %w", err)
		}
	}

	return inst, nil
}
