package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	if cfg := ConfigFromContext(ctx); cfg != nil && cfg.Default.BaseImage != "" {
		return cfg.Default.BaseImage
	}
	return config.DefaultBaseImage
}

func getInstanceByBranch(ctx context.Context, mgr *instance.Manager, branch, notFoundMsg string) (*instance.Instance, error) {
	repoPath, err := repoPath()
	if err != nil {
		return nil, err
	}

	inst, err := mgr.GetByBranch(ctx, repoPath, branch)
	if err != nil {
		if errors.Is(err, instance.ErrNotFound) && notFoundMsg != "" {
			return nil, fmt.Errorf(notFoundMsg, branch)
		}
		return nil, fmt.Errorf("get instance: %w", err)
	}

	return inst, nil
}
