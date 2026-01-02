package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmgilman/headjack/internal/exec"
)

// AppleConfig holds Apple Containerization-specific runtime configuration.
type AppleConfig struct {
	// Currently empty - all flags go through RunConfig.Flags after merging
	// at the manager level. Kept for future runtime-specific settings.
}

type appleRuntime struct {
	baseRuntime
	config AppleConfig
}

// NewAppleRuntime creates a Runtime using Apple Containerization CLI.
func NewAppleRuntime(e exec.Executor, cfg AppleConfig) Runtime {
	return &appleRuntime{
		baseRuntime: baseRuntime{
			exec:        e,
			binaryName:  "container",
			execCommand: []string{"container", "exec"},
		},
		config: cfg,
	}
}

func (r *appleRuntime) Run(ctx context.Context, cfg *RunConfig) (*Container, error) {
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

func (r *appleRuntime) Exec(ctx context.Context, id string, cfg ExecConfig) error {
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
		Name: r.binaryName,
		Args: []string{"stop", id},
	})
	if err != nil {
		return cliError("stop container", result, err)
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
		Name: r.binaryName,
		Args: []string{"start", id},
	})
	if err != nil {
		return cliError("start container", result, err)
	}

	return nil
}

func (r *appleRuntime) Remove(ctx context.Context, id string) error {
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

func (r *appleRuntime) Get(ctx context.Context, id string) (*Container, error) {
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

	var infos []appleInspect
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

	var items []appleListItem
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

// appleInspect represents the JSON output of `container inspect`.
type appleInspect struct {
	Status        string `json:"status"`
	Configuration struct {
		ID    string `json:"id"`
		Image struct {
			Reference string `json:"reference"`
		} `json:"image"`
	} `json:"configuration"`
}

func (c *appleInspect) toContainer() *Container {
	return &Container{
		ID:     c.Configuration.ID,
		Name:   c.Configuration.ID,
		Image:  c.Configuration.Image.Reference,
		Status: parseContainerStatus(c.Status),
	}
}

// appleListItem represents a single item in `container list` JSON output.
// Note: Apple container list has same format as inspect.
type appleListItem struct {
	Status        string `json:"status"`
	Configuration struct {
		ID    string `json:"id"`
		Image struct {
			Reference string `json:"reference"`
		} `json:"image"`
	} `json:"configuration"`
}

func (c *appleListItem) toContainer() Container {
	return Container{
		ID:     c.Configuration.ID,
		Name:   c.Configuration.ID,
		Image:  c.Configuration.Image.Reference,
		Status: parseContainerStatus(c.Status),
	}
}
