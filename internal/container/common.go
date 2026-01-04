package container

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/jmgilman/headjack/internal/exec"
)

// containerParser handles runtime-specific JSON parsing for container inspect and list operations.
// Each runtime implementation provides its own parser to handle the different JSON formats
// returned by each container CLI.
type containerParser interface {
	// parseInspect parses the JSON output of the inspect command.
	parseInspect(data []byte) (*Container, error)
	// parseList parses the JSON output of the list command.
	parseList(data []byte) ([]Container, error)
}

// baseRuntime provides shared functionality for container runtimes.
// Concrete implementations (Docker, Podman) configure this with runtime-specific settings
// and provide a containerParser for JSON parsing.
type baseRuntime struct {
	exec        exec.Executor
	binaryName  string
	execCommand []string
	listArgs    []string        // e.g., ["ps", "-a"] for Podman, ["list"] for Apple
	parser      containerParser // Runtime-specific JSON parser
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

// Run creates and starts a new container.
func (r *baseRuntime) Run(ctx context.Context, cfg *RunConfig) (*Container, error) {
	args := buildRunArgs(cfg)

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: r.binaryName,
		Args: args,
	})
	if err != nil {
		stderr := string(result.Stderr)
		if isAlreadyExistsError(stderr) {
			return nil, ErrAlreadyExists
		}
		return nil, cliError("run container", result, err)
	}

	// Container ID is returned on stdout
	containerID := strings.TrimSpace(string(result.Stdout))

	return &Container{
		ID:        containerID,
		Name:      cfg.Name,
		Image:     cfg.Image,
		Status:    StatusRunning,
		CreatedAt: time.Now(),
	}, nil
}

// Exec executes a command in a running container.
func (r *baseRuntime) Exec(ctx context.Context, id string, cfg *ExecConfig) error {
	// Verify container exists and is running
	container, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	if container.Status != StatusRunning {
		return ErrNotRunning
	}

	args := buildExecArgs(id, cfg)

	if cfg.Interactive {
		return r.execInteractive(ctx, args)
	}

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: r.binaryName,
		Args: args,
	})
	if err != nil {
		return cliError("exec in container", result, err)
	}

	return nil
}

// Stop stops a running container gracefully.
func (r *baseRuntime) Stop(ctx context.Context, id string) error {
	// Verify container exists
	c, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	// No-op if already stopped
	if c.Status == StatusStopped {
		return nil
	}

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: r.binaryName,
		Args: []string{"stop", id},
	})
	if err != nil {
		return cliError("stop container", result, err)
	}

	return nil
}

// Start starts a stopped container.
func (r *baseRuntime) Start(ctx context.Context, id string) error {
	// Verify container exists
	c, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	// No-op if already running
	if c.Status == StatusRunning {
		return nil
	}

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: r.binaryName,
		Args: []string{"start", id},
	})
	if err != nil {
		return cliError("start container", result, err)
	}

	return nil
}

// Remove deletes a container.
func (r *baseRuntime) Remove(ctx context.Context, id string) error {
	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: r.binaryName,
		Args: []string{"rm", id},
	})
	if err != nil {
		stderr := string(result.Stderr)
		if isNotFoundError(stderr) {
			return ErrNotFound
		}
		return cliError("remove container", result, err)
	}

	return nil
}

// Get retrieves container information by ID or name.
func (r *baseRuntime) Get(ctx context.Context, id string) (*Container, error) {
	if r.parser == nil {
		return nil, ErrNoParser
	}

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: r.binaryName,
		Args: []string{"inspect", id},
	})
	if err != nil {
		stderr := string(result.Stderr)
		if isNotFoundError(stderr) {
			return nil, ErrNotFound
		}
		return nil, cliError("inspect container", result, err)
	}

	return r.parser.parseInspect(result.Stdout)
}

// List returns all containers matching the filter.
func (r *baseRuntime) List(ctx context.Context, filter ListFilter) ([]Container, error) {
	if r.parser == nil {
		return nil, ErrNoParser
	}

	args := append([]string{}, r.listArgs...)
	args = append(args, "--format", "json")

	if filter.Name != "" {
		args = append(args, "--filter", "name="+filter.Name)
	}

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: r.binaryName,
		Args: args,
	})
	if err != nil {
		return nil, cliError("list containers", result, err)
	}

	// Handle empty list
	stdout := strings.TrimSpace(string(result.Stdout))
	if stdout == "" || stdout == "[]" {
		return []Container{}, nil
	}

	return r.parser.parseList(result.Stdout)
}

// Build builds an OCI image from a Dockerfile.
func (r *baseRuntime) Build(ctx context.Context, cfg *BuildConfig) error {
	args := buildBuildArgs(cfg)

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: r.binaryName,
		Args: args,
	})
	if err != nil {
		return fmt.Errorf("%w: %s", ErrBuildFailed, strings.TrimSpace(string(result.Stderr)))
	}

	return nil
}

// execInteractive runs a container exec command with TTY support.
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
func buildExecArgs(id string, cfg *ExecConfig) []string {
	args := []string{"exec"}

	if cfg.Interactive {
		args = append(args, "-it")
	}

	if cfg.User != "" {
		args = append(args, "-u", cfg.User)
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
	normalized := strings.ToLower(stderr)
	return strings.Contains(normalized, "no such") ||
		strings.Contains(normalized, "no container") ||
		strings.Contains(normalized, "not found")
}
