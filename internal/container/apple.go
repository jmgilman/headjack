package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/jmgilman/headjack/internal/exec"
)

// AppleConfig holds Apple Containerization-specific runtime configuration.
type AppleConfig struct {
	Privileged bool     // Run containers in privileged mode
	Flags      []string // Custom flags passed to container run
}

type appleRuntime struct {
	exec   exec.Executor
	config AppleConfig
}

// NewAppleRuntime creates a Runtime using Apple Containerization CLI.
func NewAppleRuntime(e exec.Executor, cfg AppleConfig) Runtime {
	return &appleRuntime{exec: e, config: cfg}
}

// containerError formats an error from the container CLI, including stderr if available.
func containerError(operation string, result *exec.Result, err error) error {
	if result != nil {
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr != "" {
			return fmt.Errorf("%s: %s", operation, stderr)
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func (r *appleRuntime) Run(ctx context.Context, cfg *RunConfig) (*Container, error) {
	args := []string{"run", "--detach", "--name", cfg.Name}

	if r.config.Privileged {
		args = append(args, "--privileged")
	}

	// Add custom flags from config
	args = append(args, r.config.Flags...)

	// Add image-specific flags (e.g., from image labels)
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

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "container",
		Args: args,
	})
	if err != nil {
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "already exists") {
			return nil, ErrAlreadyExists
		}
		return nil, containerError("run container", result, err)
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

func (r *appleRuntime) Exec(ctx context.Context, id string, cfg ExecConfig) error {
	// Verify container exists and is running
	container, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	if container.Status != StatusRunning {
		return ErrNotRunning
	}

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

	if cfg.Interactive {
		return r.execInteractive(ctx, args)
	}

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "container",
		Args: args,
	})
	if err != nil {
		return containerError("exec in container", result, err)
	}

	return nil
}

// execInteractive runs the container exec command with TTY support.
func (r *appleRuntime) execInteractive(ctx context.Context, args []string) error {
	stdinFd := int(os.Stdin.Fd())

	// Check if stdin is a terminal
	if !term.IsTerminal(stdinFd) {
		// Fall back to non-interactive mode
		_, err := r.exec.Run(ctx, &exec.RunOptions{
			Name:   "container",
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
		Name:   "container",
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	return err
}

func (r *appleRuntime) Stop(ctx context.Context, id string) error {
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
		Name: "container",
		Args: []string{"stop", id},
	})
	if err != nil {
		return containerError("stop container", result, err)
	}

	return nil
}

func (r *appleRuntime) Start(ctx context.Context, id string) error {
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
		Name: "container",
		Args: []string{"start", id},
	})
	if err != nil {
		return containerError("start container", result, err)
	}

	return nil
}

func (r *appleRuntime) Remove(ctx context.Context, id string) error {
	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "container",
		Args: []string{"rm", id},
	})
	if err != nil {
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "not found") || strings.Contains(stderr, "no such") {
			return ErrNotFound
		}
		return containerError("remove container", result, err)
	}

	return nil
}

func (r *appleRuntime) Get(ctx context.Context, id string) (*Container, error) {
	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "container",
		Args: []string{"inspect", id},
	})
	if err != nil {
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "not found") || strings.Contains(stderr, "no such") {
			return nil, ErrNotFound
		}
		return nil, containerError("inspect container", result, err)
	}

	var infos []containerInspect
	if err := json.Unmarshal(result.Stdout, &infos); err != nil {
		return nil, fmt.Errorf("parse container info: %w", err)
	}

	if len(infos) == 0 {
		return nil, ErrNotFound
	}

	return infos[0].toContainer(), nil
}

func (r *appleRuntime) List(ctx context.Context, filter ListFilter) ([]Container, error) {
	args := []string{"list", "--format", "json"}

	if filter.Name != "" {
		args = append(args, "--filter", "name="+filter.Name)
	}

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "container",
		Args: args,
	})
	if err != nil {
		return nil, containerError("list containers", result, err)
	}

	// Handle empty list
	stdout := strings.TrimSpace(string(result.Stdout))
	if stdout == "" || stdout == "[]" {
		return []Container{}, nil
	}

	var items []containerListItem
	if err := json.Unmarshal(result.Stdout, &items); err != nil {
		return nil, fmt.Errorf("parse container list: %w", err)
	}

	containers := make([]Container, len(items))
	for i, item := range items {
		containers[i] = item.toContainer()
	}

	return containers, nil
}

func (r *appleRuntime) Build(ctx context.Context, cfg *BuildConfig) error {
	args := []string{"build", "-t", cfg.Tag}

	if cfg.Dockerfile != "" {
		args = append(args, "-f", cfg.Dockerfile)
	}

	args = append(args, cfg.Context)

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "container",
		Args: args,
	})
	if err != nil {
		return fmt.Errorf("%w: %s", ErrBuildFailed, strings.TrimSpace(string(result.Stderr)))
	}

	return nil
}

// containerInspect represents the JSON output of `container inspect`.
type containerInspect struct {
	Status        string `json:"status"`
	Configuration struct {
		ID    string `json:"id"`
		Image struct {
			Reference string `json:"reference"`
		} `json:"image"`
	} `json:"configuration"`
}

func (c *containerInspect) toContainer() *Container {
	status := StatusUnknown
	switch strings.ToLower(c.Status) {
	case cliStatusRunning:
		status = StatusRunning
	case cliStatusStopped, cliStatusExited:
		status = StatusStopped
	}

	return &Container{
		ID:     c.Configuration.ID,
		Name:   c.Configuration.ID,
		Image:  c.Configuration.Image.Reference,
		Status: status,
	}
}

// containerListItem represents a single item in `container list` JSON output.
// Note: Apple container list has same format as inspect.
type containerListItem struct {
	Status        string `json:"status"`
	Configuration struct {
		ID    string `json:"id"`
		Image struct {
			Reference string `json:"reference"`
		} `json:"image"`
	} `json:"configuration"`
}

func (c *containerListItem) toContainer() Container {
	status := StatusUnknown
	switch strings.ToLower(c.Status) {
	case cliStatusRunning:
		status = StatusRunning
	case cliStatusStopped, cliStatusExited:
		status = StatusStopped
	}

	return Container{
		ID:     c.Configuration.ID,
		Name:   c.Configuration.ID,
		Image:  c.Configuration.Image.Reference,
		Status: status,
	}
}

func (r *appleRuntime) ExecCommand() []string {
	return []string{"container", "exec"}
}
