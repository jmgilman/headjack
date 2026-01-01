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

// PodmanConfig holds Podman-specific runtime configuration.
type PodmanConfig struct {
	Privileged bool     // Run containers in privileged mode
	Flags      []string // Custom flags passed to podman run
}

type podmanRuntime struct {
	exec   exec.Executor
	config PodmanConfig
}

// NewPodmanRuntime creates a Runtime using Podman CLI.
func NewPodmanRuntime(e exec.Executor, cfg PodmanConfig) Runtime {
	return &podmanRuntime{exec: e, config: cfg}
}

// podmanError formats an error from the podman CLI, including stderr if available.
func podmanError(operation string, result *exec.Result, err error) error {
	if result != nil {
		stderr := strings.TrimSpace(string(result.Stderr))
		if stderr != "" {
			return fmt.Errorf("%s: %s", operation, stderr)
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func (r *podmanRuntime) Run(ctx context.Context, cfg *RunConfig) (*Container, error) {
	args := []string{"run", "--detach", "--name", cfg.Name, "--systemd=always"}

	if r.config.Privileged {
		args = append(args, "--privileged")
	}

	// Add custom flags from config
	args = append(args, r.config.Flags...)

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

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "podman",
		Args: args,
	})
	if err != nil {
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "already in use") || strings.Contains(stderr, "already exists") {
			return nil, ErrAlreadyExists
		}
		return nil, podmanError("run container", result, err)
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

func (r *podmanRuntime) Exec(ctx context.Context, id string, cfg ExecConfig) error {
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
		Name: "podman",
		Args: args,
	})
	if err != nil {
		return podmanError("exec in container", result, err)
	}

	return nil
}

// execInteractive runs the podman exec command with TTY support.
func (r *podmanRuntime) execInteractive(ctx context.Context, args []string) error {
	stdinFd := int(os.Stdin.Fd())

	// Check if stdin is a terminal
	if !term.IsTerminal(stdinFd) {
		// Fall back to non-interactive mode
		_, err := r.exec.Run(ctx, &exec.RunOptions{
			Name:   "podman",
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
		Name:   "podman",
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	return err
}

func (r *podmanRuntime) Stop(ctx context.Context, id string) error {
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
		Name: "podman",
		Args: []string{"stop", id},
	})
	if err != nil {
		return podmanError("stop container", result, err)
	}

	return nil
}

func (r *podmanRuntime) Start(ctx context.Context, id string) error {
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
		Name: "podman",
		Args: []string{"start", id},
	})
	if err != nil {
		return podmanError("start container", result, err)
	}

	return nil
}

func (r *podmanRuntime) Remove(ctx context.Context, id string) error {
	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "podman",
		Args: []string{"rm", id},
	})
	if err != nil {
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "no such") || strings.Contains(stderr, "no container") {
			return ErrNotFound
		}
		return podmanError("remove container", result, err)
	}

	return nil
}

func (r *podmanRuntime) Get(ctx context.Context, id string) (*Container, error) {
	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "podman",
		Args: []string{"inspect", id},
	})
	if err != nil {
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "no such") || strings.Contains(stderr, "no container") {
			return nil, ErrNotFound
		}
		return nil, podmanError("inspect container", result, err)
	}

	var infos []podmanInspect
	if err := json.Unmarshal(result.Stdout, &infos); err != nil {
		return nil, fmt.Errorf("parse container info: %w", err)
	}

	if len(infos) == 0 {
		return nil, ErrNotFound
	}

	return infos[0].toContainer(), nil
}

func (r *podmanRuntime) List(ctx context.Context, filter ListFilter) ([]Container, error) {
	args := []string{"ps", "-a", "--format", "json"}

	if filter.Name != "" {
		args = append(args, "--filter", "name="+filter.Name)
	}

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "podman",
		Args: args,
	})
	if err != nil {
		return nil, podmanError("list containers", result, err)
	}

	// Handle empty list
	stdout := strings.TrimSpace(string(result.Stdout))
	if stdout == "" || stdout == "[]" {
		return []Container{}, nil
	}

	var items []podmanListItem
	if err := json.Unmarshal(result.Stdout, &items); err != nil {
		return nil, fmt.Errorf("parse container list: %w", err)
	}

	containers := make([]Container, len(items))
	for i, item := range items {
		containers[i] = item.toContainer()
	}

	return containers, nil
}

func (r *podmanRuntime) Build(ctx context.Context, cfg *BuildConfig) error {
	args := []string{"build", "-t", cfg.Tag}

	if cfg.Dockerfile != "" {
		args = append(args, "-f", cfg.Dockerfile)
	}

	args = append(args, cfg.Context)

	result, err := r.exec.Run(ctx, &exec.RunOptions{
		Name: "podman",
		Args: args,
	})
	if err != nil {
		return fmt.Errorf("%w: %s", ErrBuildFailed, strings.TrimSpace(string(result.Stderr)))
	}

	return nil
}

// podmanInspect represents the JSON output of `podman inspect`.
type podmanInspect struct {
	ID      string `json:"Id"`
	Name    string `json:"Name"`
	Created string `json:"Created"`
	State   struct {
		Status string `json:"Status"`
	} `json:"State"`
	Config struct {
		Image string `json:"Image"`
	} `json:"Config"`
	ImageName string `json:"ImageName"`
}

func (p *podmanInspect) toContainer() *Container {
	status := StatusUnknown
	switch strings.ToLower(p.State.Status) {
	case cliStatusRunning:
		status = StatusRunning
	case cliStatusStopped, cliStatusExited, cliStatusCreated:
		status = StatusStopped
	}

	// Remove leading "/" from name if present
	name := strings.TrimPrefix(p.Name, "/")

	// Use ImageName if available, otherwise fall back to Config.Image
	image := p.ImageName
	if image == "" {
		image = p.Config.Image
	}

	// Podman uses RFC3339Nano format (with fractional seconds), fall back to RFC3339
	createdAt, err := time.Parse(time.RFC3339Nano, p.Created)
	if err != nil {
		createdAt, err = time.Parse(time.RFC3339, p.Created)
		if err != nil {
			createdAt = time.Time{}
		}
	}

	return &Container{
		ID:        p.ID,
		Name:      name,
		Image:     image,
		Status:    status,
		CreatedAt: createdAt,
	}
}

// podmanListItem represents a single item in `podman ps` JSON output.
type podmanListItem struct {
	ID      string   `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	State   string   `json:"State"`
	Created int64    `json:"Created"`
}

func (p *podmanListItem) toContainer() Container {
	status := StatusUnknown
	switch strings.ToLower(p.State) {
	case cliStatusRunning:
		status = StatusRunning
	case cliStatusStopped, cliStatusExited, cliStatusCreated:
		status = StatusStopped
	}

	name := ""
	if len(p.Names) > 0 {
		name = p.Names[0]
	}

	return Container{
		ID:        p.ID,
		Name:      name,
		Image:     p.Image,
		Status:    status,
		CreatedAt: time.Unix(p.Created, 0),
	}
}

func (r *podmanRuntime) ExecCommand() []string {
	return []string{"podman", "exec"}
}
