package git

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jmgilman/headjack/internal/exec"
)

// openerGitError formats an error from a git command in the opener, including stderr if available.
func openerGitError(operation string, result *exec.Result, err error) error {
	if result != nil {
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr != "" {
			return fmt.Errorf("%s: %s", operation, stderr)
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

type opener struct {
	exec exec.Executor
}

// NewOpener creates a new Opener that uses the provided Executor.
func NewOpener(e exec.Executor) Opener {
	return &opener{exec: e}
}

func (o *opener) Open(ctx context.Context, path string) (Repository, error) {
	// Get repository root
	root, err := o.getRepoRoot(ctx, path)
	if err != nil {
		return nil, err
	}

	// Generate repository identifier
	identifier, err := o.generateIdentifier(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("generate identifier: %w", err)
	}

	return &repository{
		root:       root,
		identifier: identifier,
		exec:       o.exec,
	}, nil
}

// getRepoRoot returns the repository root for the given path.
func (o *opener) getRepoRoot(ctx context.Context, path string) (string, error) {
	result, err := o.exec.Run(ctx, &exec.RunOptions{
		Name: "git",
		Args: []string{"rev-parse", "--show-toplevel"},
		Dir:  path,
	})
	if err != nil {
		// Check if the error indicates not a git repo
		if result != nil {
			stderr := string(result.Stderr)
			if strings.Contains(stderr, "not a git repository") {
				return "", ErrNotRepository
			}
		}
		return "", openerGitError("get repository root", result, err)
	}

	return strings.TrimSpace(string(result.Stdout)), nil
}

// generateIdentifier creates a unique identifier for the repository.
// Format: "<repo-name>-<short-initial-commit-hash>"
func (o *opener) generateIdentifier(ctx context.Context, root string) (string, error) {
	// Get the initial commit hash (first commit in history)
	result, err := o.exec.Run(ctx, &exec.RunOptions{
		Name: "git",
		Args: []string{"rev-list", "--max-parents=0", "HEAD"},
		Dir:  root,
	})
	if err != nil {
		return "", openerGitError("get initial commit", result, err)
	}

	// Take first line (in case of multiple roots) and first 7 chars
	lines := strings.Split(strings.TrimSpace(string(result.Stdout)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", errors.New("no commits found in repository")
	}

	hash := lines[0]
	if len(hash) > 7 {
		hash = hash[:7]
	}

	// Get repo name from directory
	name := filepath.Base(root)

	return fmt.Sprintf("%s-%s", name, hash), nil
}
