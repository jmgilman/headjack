package container

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/jmgilman/headjack/internal/exec"
)

// baseRuntime provides shared functionality for container runtimes.
// Concrete implementations (Podman, Apple) embed this and provide runtime-specific behavior.
type baseRuntime struct {
	exec        exec.Executor
	binaryName  string
	execCommand []string
}

// cliError formats an error from a container CLI, including stderr if available.
func cliError(operation string, result *exec.Result, err error) error {
	if result != nil {
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr != "" {
			return fmt.Errorf("%s: %s", operation, stderr)
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// execInteractive runs a container exec command with TTY support.
// This is shared between runtime implementations.
func (r *baseRuntime) execInteractive(ctx context.Context, args []string) error {
	stdinFd := int(os.Stdin.Fd())

	// Check if stdin is a terminal
	if !term.IsTerminal(stdinFd) {
		// Fall back to non-interactive mode
		_, err := r.exec.Run(ctx, &exec.RunOptions{
			Name:   r.binaryName,
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
		Name:   r.binaryName,
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	return err
}

// ExecCommand returns the command prefix for executing commands in a container.
func (r *baseRuntime) ExecCommand() []string {
	return r.execCommand
}

// buildRunArgs constructs the common container run arguments.
func buildRunArgs(cfg *RunConfig) []string {
	args := []string{"run", "--detach", "--name", cfg.Name}

	// Add merged flags (image labels + config, merged by manager)
	args = append(args, cfg.Flags...)

	for _, m := range cfg.Mounts {
		mountSpec := fmt.Sprintf("%s:%s", m.Source, m.Target)
		if m.ReadOnly {
			mountSpec += ":ro"
		}
		args = append(args, "-v", mountSpec)
	}

	for _, e := range cfg.Env {
		args = append(args, "-e", e)
	}

	args = append(args, cfg.Image)

	// Add init command (default to "sleep infinity" if not specified)
	initCmd := cfg.Init
	if initCmd == "" {
		initCmd = "sleep infinity"
	}
	args = append(args, strings.Fields(initCmd)...)

	return args
}

// buildExecArgs constructs the common container exec arguments.
func buildExecArgs(id string, cfg ExecConfig) []string {
	args := []string{"exec"}

	if cfg.Interactive {
		args = append(args, "-it")
	}

	if cfg.Workdir != "" {
		args = append(args, "-w", cfg.Workdir)
	}

	for _, e := range cfg.Env {
		args = append(args, "-e", e)
	}

	args = append(args, id)
	args = append(args, cfg.Command...)

	return args
}

// buildBuildArgs constructs the common image build arguments.
func buildBuildArgs(cfg *BuildConfig) []string {
	args := []string{"build", "-t", cfg.Tag}

	if cfg.Dockerfile != "" {
		args = append(args, "-f", cfg.Dockerfile)
	}

	args = append(args, cfg.Context)
	return args
}

// parseContainerStatus converts CLI status strings to Status constants.
func parseContainerStatus(cliStatus string) Status {
	switch strings.ToLower(cliStatus) {
	case cliStatusRunning:
		return StatusRunning
	case cliStatusStopped, cliStatusExited, cliStatusCreated:
		return StatusStopped
	default:
		return StatusUnknown
	}
}

// isAlreadyExistsError checks if stderr indicates container already exists.
func isAlreadyExistsError(stderr string) bool {
	return strings.Contains(stderr, "already in use") || strings.Contains(stderr, "already exists")
}

// isNotFoundError checks if stderr indicates container not found.
func isNotFoundError(stderr string) bool {
	return strings.Contains(stderr, "no such") ||
		strings.Contains(stderr, "no container") ||
		strings.Contains(stderr, "not found")
}
