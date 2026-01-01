// Package container provides an abstraction over container runtime operations.
package container

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors for container operations.
var (
	ErrNotFound      = errors.New("container not found")
	ErrNotRunning    = errors.New("container not running")
	ErrAlreadyExists = errors.New("container already exists")
	ErrBuildFailed   = errors.New("image build failed")
)

// Status represents the container state.
type Status string

// Status constants represent possible container states.
const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusUnknown Status = "unknown"
)

// Container holds container metadata.
type Container struct {
	ID        string
	Name      string
	Image     string
	Status    Status
	CreatedAt time.Time
}

// Mount defines a host-to-container volume mount.
type Mount struct {
	Source   string // Host path
	Target   string // Container path
	ReadOnly bool
}

// RunConfig configures container creation.
type RunConfig struct {
	Name   string   // Container name (required)
	Image  string   // OCI image reference (required)
	Mounts []Mount  // Volume mounts
	Env    []string // Environment variables (KEY=VALUE format)
}

// ExecConfig configures command execution in a container.
type ExecConfig struct {
	Command     []string // Command and arguments (required)
	Env         []string // Additional environment variables
	Interactive bool     // If true, sets up TTY with raw mode and signal forwarding
	Workdir     string   // Working directory (empty = container default)
}

// BuildConfig configures image builds.
type BuildConfig struct {
	Context    string // Build context directory
	Dockerfile string // Path to Dockerfile (relative to context)
	Tag        string // Image tag to apply (required)
}

// ListFilter filters container listings.
type ListFilter struct {
	Name string // Filter by name prefix (empty = all)
}

// Runtime provides container lifecycle operations.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/runtime.go . Runtime
type Runtime interface {
	// Run creates and starts a new container.
	// The container runs with systemd as init and stays running until stopped.
	// Returns ErrAlreadyExists if a container with the same name exists.
	Run(ctx context.Context, cfg *RunConfig) (*Container, error)

	// Exec executes a command in a running container.
	// If Interactive is true, sets up TTY with raw mode and forwards signals.
	// Blocks until the command exits.
	// Returns ErrNotFound if container doesn't exist.
	// Returns ErrNotRunning if container is stopped.
	Exec(ctx context.Context, id string, cfg ExecConfig) error

	// Stop stops a running container gracefully.
	// No-op if already stopped.
	// Returns ErrNotFound if container doesn't exist.
	Stop(ctx context.Context, id string) error

	// Remove deletes a container.
	// Container must be stopped first.
	// Returns ErrNotFound if container doesn't exist.
	Remove(ctx context.Context, id string) error

	// Get retrieves container information by ID or name.
	// Returns ErrNotFound if container doesn't exist.
	Get(ctx context.Context, id string) (*Container, error)

	// List returns all containers matching the filter.
	List(ctx context.Context, filter ListFilter) ([]Container, error)

	// Build builds an OCI image from a Dockerfile.
	// Returns ErrBuildFailed if the build fails.
	Build(ctx context.Context, cfg *BuildConfig) error
}
