// Package devcontainer provides integration with the Dev Container CLI.
package devcontainer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/jmgilman/headjack/internal/container"
	"github.com/jmgilman/headjack/internal/exec"
)

// Runtime wraps an underlying container runtime with devcontainer CLI support.
// It implements the container.Runtime interface by delegating creation and exec
// to the devcontainer CLI, while lifecycle operations (stop, start, remove, etc.)
// are delegated to the underlying runtime.
type Runtime struct {
	underlying container.Runtime // Docker or Podman runtime
	exec       exec.Executor     // Command executor
	cliPath    string            // Path to devcontainer CLI binary
	dockerPath string            // Path to docker/podman binary (for --docker-path)
}

// NewRuntime creates a DevcontainerRuntime wrapping the given underlying runtime.
func NewRuntime(underlying container.Runtime, executor exec.Executor, cliPath, dockerPath string) *Runtime {
	return &Runtime{
		underlying: underlying,
		exec:       executor,
		cliPath:    cliPath,
		dockerPath: dockerPath,
	}
}

// upResult represents the JSON output from devcontainer up.
type upResult struct {
	Outcome               string `json:"outcome"`
	ContainerID           string `json:"containerId"`
	RemoteUser            string `json:"remoteUser"`
	RemoteWorkspaceFolder string `json:"remoteWorkspaceFolder"`
}

// Run creates a container using devcontainer up.
func (r *Runtime) Run(ctx context.Context, cfg *container.RunConfig) (*container.Container, error) {
	if cfg.WorkspaceFolder == "" {
		return nil, errors.New("WorkspaceFolder is required for devcontainer runtime")
	}

	args := []string{
		"up",
		"--workspace-folder", cfg.WorkspaceFolder,
		"--docker-path", r.dockerPath,
	}

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: r.cliPath,
		Args: args,
	})
	if err != nil {
		stderr := ""
		if result != nil {
			stderr = strings.TrimSpace(string(result.Stderr))
		}
		if stderr != "" {
			return nil, fmt.Errorf("devcontainer up: %s", stderr)
		}
		return nil, fmt.Errorf("devcontainer up: %w", err)
	}

	// Parse JSON output
	var upRes upResult
	if err := json.Unmarshal(result.Stdout, &upRes); err != nil {
		return nil, fmt.Errorf("parse devcontainer output: %w", err)
	}

	if upRes.Outcome != "success" {
		return nil, fmt.Errorf("devcontainer up failed: %s", upRes.Outcome)
	}

	// Return container info with devcontainer-specific fields
	return &container.Container{
		ID:                    upRes.ContainerID,
		Name:                  cfg.Name,
		Status:                container.StatusRunning,
		RemoteUser:            upRes.RemoteUser,
		RemoteWorkspaceFolder: upRes.RemoteWorkspaceFolder,
	}, nil
}

// Exec executes a command using devcontainer exec.
func (r *Runtime) Exec(ctx context.Context, id string, cfg *container.ExecConfig) error {
	// Verify container exists and is running via underlying runtime
	c, err := r.underlying.Get(ctx, id)
	if err != nil {
		return err
	}
	if c.Status != container.StatusRunning {
		return container.ErrNotRunning
	}

	args := []string{
		"exec",
		"--container-id", id,
		"--docker-path", r.dockerPath,
	}

	// Add user if specified
	if cfg.User != "" {
		args = append(args, "--remote-user", cfg.User)
	}

	// Add environment variables
	for _, env := range cfg.Env {
		args = append(args, "--remote-env", env)
	}

	// Handle workdir by wrapping command with cd if needed
	// The devcontainer CLI doesn't have a direct --remote-cwd flag
	cmd := cfg.Command
	if cfg.Workdir != "" && len(cmd) > 0 {
		// Wrap command to change directory first, with proper shell quoting
		shellCmd := fmt.Sprintf("cd %s && exec \"$@\"", shellQuote(cfg.Workdir))
		cmd = append([]string{"sh", "-c", shellCmd, "--"}, cmd...)
	}

	// Add command
	args = append(args, cmd...)

	if cfg.Interactive {
		return r.execInteractive(ctx, args)
	}

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: r.cliPath,
		Args: args,
	})
	if err != nil {
		stderr := ""
		if result != nil {
			stderr = strings.TrimSpace(string(result.Stderr))
		}
		if stderr != "" {
			return fmt.Errorf("devcontainer exec: %s", stderr)
		}
		return fmt.Errorf("devcontainer exec: %w", err)
	}

	return nil
}

// execInteractive runs a devcontainer exec command with TTY support.
func (r *Runtime) execInteractive(ctx context.Context, args []string) error {
	stdinFd := int(os.Stdin.Fd())

	// Check if stdin is a terminal
	if !term.IsTerminal(stdinFd) {
		// Fall back to non-interactive mode
		_, err := r.exec.Run(ctx, &exec.RunOptions{
			Name:   r.cliPath,
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		})
		return err
	}

	// Put terminal in raw mode
	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return fmt.Errorf("set terminal raw mode: %w", err)
	}
	defer func() { _ = term.Restore(stdinFd, oldState) }()

	// Handle window resize signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	// Run the command with stdio attached
	_, err = r.exec.Run(ctx, &exec.RunOptions{
		Name:   r.cliPath,
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	return err
}

// Stop delegates to the underlying runtime.
func (r *Runtime) Stop(ctx context.Context, id string) error {
	return r.underlying.Stop(ctx, id)
}

// Start delegates to the underlying runtime.
func (r *Runtime) Start(ctx context.Context, id string) error {
	return r.underlying.Start(ctx, id)
}

// Remove delegates to the underlying runtime.
func (r *Runtime) Remove(ctx context.Context, id string) error {
	return r.underlying.Remove(ctx, id)
}

// Get delegates to the underlying runtime.
func (r *Runtime) Get(ctx context.Context, id string) (*container.Container, error) {
	return r.underlying.Get(ctx, id)
}

// List delegates to the underlying runtime.
func (r *Runtime) List(ctx context.Context, filter container.ListFilter) ([]container.Container, error) {
	return r.underlying.List(ctx, filter)
}

// Build delegates to the underlying runtime.
func (r *Runtime) Build(ctx context.Context, cfg *container.BuildConfig) error {
	return r.underlying.Build(ctx, cfg)
}

// ExecCommand returns the underlying runtime's exec command.
func (r *Runtime) ExecCommand() []string {
	return r.underlying.ExecCommand()
}

// shellQuote quotes a string for safe use in a shell command.
// It wraps the string in single quotes and escapes any embedded single quotes.
func shellQuote(s string) string {
	// Replace each ' with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}
