// Package catalog provides persistent storage for instance state.
package catalog

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors for catalog operations.
var (
	ErrNotFound      = errors.New("entry not found")
	ErrAlreadyExists = errors.New("entry already exists")
	ErrLockTimeout   = errors.New("failed to acquire catalog lock")
)

// Status represents the instance lifecycle state.
type Status string

const (
	StatusCreating Status = "creating"
	StatusRunning  Status = "running"
	StatusStopped  Status = "stopped"
	StatusError    Status = "error"
)

// Entry represents a persisted instance record.
type Entry struct {
	ID          string    `json:"id"`
	Repo        string    `json:"repo"`         // Absolute path to source repository
	RepoID      string    `json:"repo_id"`      // Unique repository identifier
	Branch      string    `json:"branch"`       // Branch name
	Worktree    string    `json:"worktree"`     // Absolute path to worktree
	ContainerID string    `json:"container_id"` // Container ID (may be empty)
	CreatedAt   time.Time `json:"created_at"`
	Status      Status    `json:"status"`
}

// ListFilter filters catalog queries.
type ListFilter struct {
	RepoID string // Filter by repository identifier (empty = all)
	Status Status // Filter by status (empty = all)
}

// Store provides persistent storage for instance entries.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/store.go . Store
type Store interface {
	// Add creates a new entry.
	// Returns ErrAlreadyExists if an entry with the same RepoID+Branch already exists.
	Add(ctx context.Context, entry Entry) error

	// Get retrieves an entry by ID.
	// Returns ErrNotFound if not found.
	Get(ctx context.Context, id string) (*Entry, error)

	// GetByRepoBranch retrieves an entry by repository ID and branch.
	// Returns ErrNotFound if not found.
	GetByRepoBranch(ctx context.Context, repoID, branch string) (*Entry, error)

	// Update modifies an existing entry.
	// Returns ErrNotFound if not found.
	Update(ctx context.Context, entry Entry) error

	// Remove deletes an entry by ID.
	// Returns ErrNotFound if not found.
	Remove(ctx context.Context, id string) error

	// List returns all entries matching the filter.
	List(ctx context.Context, filter ListFilter) ([]Entry, error)
}
