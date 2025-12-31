// Package instance provides high-level instance lifecycle management.
package instance

import (
	"errors"
	"time"

	"github.com/jmgilman/headjack/internal/container"
)

// Sentinel errors for instance operations.
var (
	ErrNotFound      = errors.New("instance not found")
	ErrAlreadyExists = errors.New("instance already exists for this branch")
)

// Status represents the instance lifecycle state.
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusError   Status = "error"
)

// Instance represents a managed development environment.
type Instance struct {
	ID          string               // Unique instance identifier
	Repo        string               // Absolute path to source repository
	RepoID      string               // Unique repository identifier
	Branch      string               // Branch name
	Worktree    string               // Absolute path to worktree
	ContainerID string               // Container ID (may be empty if not created)
	Container   *container.Container // Live container state (nil if not running)
	CreatedAt   time.Time
	Status      Status
}

// CreateConfig configures instance creation.
type CreateConfig struct {
	Branch string // Branch to create or checkout
	Image  string // OCI image to use for container
}

// AttachConfig configures instance attachment.
type AttachConfig struct {
	Command     []string // Command to execute (default: shell)
	Interactive bool     // If true, sets up TTY
	Workdir     string   // Working directory in container
	Env         []string // Additional environment variables
}

// ListFilter filters instance listings.
type ListFilter struct {
	RepoID string // Filter by repository identifier (empty = all)
	Status Status // Filter by status (empty = all)
}
