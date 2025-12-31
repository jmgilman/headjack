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

// SessionType represents the type of session running within an instance.
type SessionType string

const (
	SessionTypeShell  SessionType = "shell"
	SessionTypeClaude SessionType = "claude"
	SessionTypeGemini SessionType = "gemini"
	SessionTypeCodex  SessionType = "codex"
)

// Session represents a persistent, attachable process running within an instance.
type Session struct {
	ID            string      `json:"id"`             // Unique session identifier
	Name          string      `json:"name"`           // Human-readable name (e.g., "happy-panda")
	Type          SessionType `json:"type"`           // Session type (shell, claude, gemini, codex)
	ZellijSession string      `json:"zellij_session"` // Zellij session identifier
	CreatedAt     time.Time   `json:"created_at"`     // Creation timestamp
	LastAccessed  time.Time   `json:"last_accessed"`  // Last access timestamp (for MRU tracking)
}

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
	Sessions    []Session `json:"sessions"` // Sessions running within this instance
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
