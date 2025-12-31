// Package exec provides an abstraction over executing external commands.
package exec

import (
	"context"
	"io"
)

// Result holds the output from a completed command.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// RunOptions configures command execution.
type RunOptions struct {
	Name   string    // Command name or path (required)
	Args   []string  // Command arguments
	Dir    string    // Working directory (empty = current)
	Env    []string  // Additional environment variables (KEY=VALUE format)
	Stdin  io.Reader // Stdin source (nil = no input)
	Stdout io.Writer // If set, streams stdout here instead of capturing
	Stderr io.Writer // If set, streams stderr here instead of capturing
}

// Executor runs external commands.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/executor.go . Executor
type Executor interface {
	// Run executes a command and returns its output.
	// If Stdout/Stderr writers are set in opts, output streams there and
	// Result.Stdout/Stderr will be nil.
	// Returns os/exec.ExitError on non-zero exit (use errors.As to extract).
	Run(ctx context.Context, opts RunOptions) (*Result, error)

	// LookPath searches for an executable in PATH.
	// Returns the full path if found, or an error if not.
	LookPath(name string) (string, error)
}
