// Package git provides an abstraction over git operations.
package git

import (
	"context"
	"errors"
)

// Sentinel errors for git operations.
var (
	ErrNotRepository    = errors.New("not a git repository")
	ErrBranchExists     = errors.New("branch already exists")
	ErrBranchNotFound   = errors.New("branch not found")
	ErrWorktreeExists   = errors.New("worktree already exists")
	ErrWorktreeNotFound = errors.New("worktree not found")
)

// Worktree represents a git worktree.
type Worktree struct {
	Path   string // Filesystem path to the worktree
	Branch string // Branch checked out in the worktree
	Bare   bool   // True if this is the main worktree of a bare repo
}

// Repository provides git operations for a repository.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/repository.go . Repository
type Repository interface {
	// Root returns the absolute path to the repository root.
	Root() string

	// Identifier returns a unique identifier for the repository.
	// Format: "<repo-name>-<short-commit-hash>" (e.g., "myproject-a1b2c3").
	Identifier() string

	// BranchExists checks if a branch exists locally or in any remote.
	BranchExists(ctx context.Context, branch string) (bool, error)

	// CreateWorktree creates a new worktree at the specified path.
	// If the branch exists, it checks out that branch.
	// If the branch does not exist, it creates a new branch from HEAD.
	CreateWorktree(ctx context.Context, path, branch string) error

	// RemoveWorktree removes a worktree at the specified path.
	// Returns ErrWorktreeNotFound if the worktree does not exist.
	RemoveWorktree(ctx context.Context, path string) error

	// ListWorktrees returns all worktrees for the repository.
	ListWorktrees(ctx context.Context) ([]Worktree, error)

	// WorktreeForBranch returns the worktree path for a branch, if one exists.
	// Returns empty string if no worktree exists for the branch.
	WorktreeForBranch(ctx context.Context, branch string) (string, error)
}

// Opener opens git repositories.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/opener.go . Opener
type Opener interface {
	// Open opens the git repository containing the given path.
	// Returns ErrNotRepository if the path is not inside a git repository.
	Open(ctx context.Context, path string) (Repository, error)
}
