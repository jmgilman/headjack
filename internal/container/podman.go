package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmgilman/headjack/internal/exec"
)

// PodmanConfig holds Podman-specific runtime configuration.
type PodmanConfig struct {
	// Currently empty - all flags go through RunConfig.Flags after merging
	// at the manager level. Kept for future runtime-specific settings.
}

type podmanRuntime struct {
	baseRuntime
	config PodmanConfig
}

// NewPodmanRuntime creates a Runtime using Podman CLI.
func NewPodmanRuntime(e exec.Executor, cfg PodmanConfig) Runtime {
	return &podmanRuntime{
		baseRuntime: baseRuntime{
			exec:        e,
			binaryName:  "podman",
			execCommand: []string{"podman", "exec"},
		},
		config: cfg,
	}
}

func (r *podmanRuntime) Run(ctx context.Context, cfg *RunConfig) (*Container, error) {
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

func (r *podmanRuntime) Exec(ctx context.Context, id string, cfg ExecConfig) error {
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
		Name: r.binaryName,
		Args: []string{"stop", id},
	})
	if err != nil {
		return cliError("stop container", result, err)
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
		Name: r.binaryName,
		Args: []string{"start", id},
	})
	if err != nil {
		return cliError("start container", result, err)
	}

	return nil
}

func (r *podmanRuntime) Remove(ctx context.Context, id string) error {
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

func (r *podmanRuntime) Get(ctx context.Context, id string) (*Container, error) {
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
	status := parseContainerStatus(p.State.Status)

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
	status := parseContainerStatus(p.State)

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
