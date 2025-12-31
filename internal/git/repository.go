package git

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/jmgilman/headjack/internal/exec"
)

type repository struct {
	root       string
	identifier string
	exec       exec.Executor
}

func (r *repository) Root() string {
	return r.root
}

func (r *repository) Identifier() string {
	return r.identifier
}

func (r *repository) BranchExists(ctx context.Context, branch string) (bool, error) {
	// Check local branches
	exists, err := r.localBranchExists(ctx, branch)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// Check remote-tracking branches
	return r.remoteBranchExists(ctx, branch)
}

// localBranchExists checks if a branch exists locally.
func (r *repository) localBranchExists(ctx context.Context, branch string) (bool, error) {
	result, err := r.exec.Run(ctx, exec.RunOptions{
		Name: "git",
		Args: []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branch},
		Dir:  r.root,
	})
	if err != nil {
		// Exit code 1 means branch doesn't exist, which is not an error
		if result != nil && result.ExitCode == 1 {
			return false, nil
		}
		return false, fmt.Errorf("check local branch: %w", err)
	}
	return true, nil
}

// remoteBranchExists checks if a branch exists in any remote.
func (r *repository) remoteBranchExists(ctx context.Context, branch string) (bool, error) {
	result, err := r.exec.Run(ctx, exec.RunOptions{
		Name: "git",
		Args: []string{"branch", "-r", "--list", "*/" + branch},
		Dir:  r.root,
	})
	if err != nil {
		return false, fmt.Errorf("check remote branch: %w", err)
	}

	// If output is non-empty, a matching remote branch exists
	return strings.TrimSpace(string(result.Stdout)) != "", nil
}

func (r *repository) CreateWorktree(ctx context.Context, path, branch string) error {
	// Check if branch already exists
	exists, err := r.BranchExists(ctx, branch)
	if err != nil {
		return fmt.Errorf("check branch existence: %w", err)
	}

	var args []string
	if exists {
		// Use existing branch
		args = []string{"worktree", "add", path, branch}
	} else {
		// Create new branch from HEAD
		args = []string{"worktree", "add", "-b", branch, path}
	}

	result, err := r.exec.Run(ctx, exec.RunOptions{
		Name: "git",
		Args: args,
		Dir:  r.root,
	})
	if err != nil {
		// Check for common error conditions
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "already exists") {
			return ErrWorktreeExists
		}
		if strings.Contains(stderr, "already checked out") {
			return fmt.Errorf("branch '%s' is already checked out: %w", branch, ErrWorktreeExists)
		}
		return fmt.Errorf("create worktree: %w", err)
	}

	return nil
}

func (r *repository) RemoveWorktree(ctx context.Context, path string) error {
	result, err := r.exec.Run(ctx, exec.RunOptions{
		Name: "git",
		Args: []string{"worktree", "remove", path},
		Dir:  r.root,
	})
	if err != nil {
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "is not a working tree") {
			return ErrWorktreeNotFound
		}
		return fmt.Errorf("remove worktree: %w", err)
	}

	return nil
}

func (r *repository) ListWorktrees(ctx context.Context) ([]Worktree, error) {
	result, err := r.exec.Run(ctx, exec.RunOptions{
		Name: "git",
		Args: []string{"worktree", "list", "--porcelain"},
		Dir:  r.root,
	})
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	return parseWorktreeList(string(result.Stdout))
}

// parseWorktreeList parses the porcelain output of `git worktree list --porcelain`.
// Format:
//
//	worktree /path/to/worktree
//	HEAD <sha>
//	branch refs/heads/<branch>
//	<blank line>
//
// For bare repos or detached HEAD:
//
//	worktree /path/to/worktree
//	HEAD <sha>
//	bare
//	<blank line>
func parseWorktreeList(output string) ([]Worktree, error) {
	var worktrees []Worktree
	var current Worktree

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "worktree "):
			current.Path = strings.TrimPrefix(line, "worktree ")

		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			// Extract branch name from refs/heads/<branch>
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")

		case line == "bare":
			current.Bare = true

		case line == "":
			// End of entry
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = Worktree{}
		}
	}

	// Handle last entry if no trailing newline
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse worktree list: %w", err)
	}

	return worktrees, nil
}

func (r *repository) WorktreeForBranch(ctx context.Context, branch string) (string, error) {
	worktrees, err := r.ListWorktrees(ctx)
	if err != nil {
		return "", err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return wt.Path, nil
		}
	}

	return "", nil
}
