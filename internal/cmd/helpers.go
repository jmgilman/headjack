package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

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
