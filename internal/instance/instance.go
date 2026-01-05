// Package instance provides high-level instance lifecycle management.
package instance

import (
	"errors"
	"fmt"
	"time"

	"github.com/jmgilman/headjack/internal/container"
)

// Sentinel errors for instance operations.
var (
	ErrNotFound            = errors.New("instance not found")
	ErrAlreadyExists       = errors.New("instance already exists for this branch")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionExists       = errors.New("session already exists")
	ErrInstanceNotRunning  = errors.New("instance is not running")
	ErrNoSessionsAvailable = errors.New("no sessions available")
)

// NotRunningError describes an instance whose container is not running.
type NotRunningError struct {
	InstanceID  string
	ContainerID string
	Status      container.Status
}

func (e *NotRunningError) Error() string {
	return fmt.Sprintf("instance is not running (container %s status: %s)", e.ContainerID, e.Status)
}

func (e *NotRunningError) Unwrap() error {
	return ErrInstanceNotRunning
}

// Status represents the instance lifecycle state.
type Status string

// Instance status constants.
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
	Branch          string            // Branch to create or checkout
	Image           string            // OCI image to use for container (vanilla mode)
	WorkspaceFolder string            // Path to folder with devcontainer.json (devcontainer mode)
	Runtime         container.Runtime // Optional runtime override (for devcontainer)
	RuntimeFlags    []string          // Additional flags to pass to the container runtime
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

// Session represents a session within an instance (returned by Manager methods).
// This mirrors catalog.Session but is part of the instance package's public API.
type Session struct {
	ID           string    // Unique session identifier
	Name         string    // Human-readable name (e.g., "happy-panda")
	Type         string    // Session type (shell, claude, gemini, codex)
	MuxSessionID string    // Multiplexer session identifier
	CreatedAt    time.Time // Creation timestamp
	LastAccessed time.Time // Last access timestamp (for MRU tracking)
}

// CreateSessionConfig configures session creation.
type CreateSessionConfig struct {
	Type               string   // Session type (shell, claude, gemini, codex)
	Name               string   // Optional session name (auto-generated if empty)
	Command            []string // Initial command to run (optional, defaults to shell)
	Env                []string // Additional environment variables
	CredentialType     string   // Credential type: "subscription" or "apikey" (empty for shell)
	RequiresAgentSetup bool     // Whether agent needs file setup in container
}
