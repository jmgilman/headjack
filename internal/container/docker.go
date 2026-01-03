package container

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmgilman/headjack/internal/exec"
)

// DockerConfig holds Docker-specific runtime configuration.
type DockerConfig struct {
	// Currently empty - all flags go through RunConfig.Flags after merging
	// at the manager level. Kept for future runtime-specific settings.
}

// dockerRuntime implements Runtime using Docker CLI.
// All common functionality is provided by the embedded baseRuntime.
type dockerRuntime struct {
	baseRuntime
	config DockerConfig
}

// dockerParser implements containerParser for Docker JSON output.
type dockerParser struct{}

// NewDockerRuntime creates a Runtime using Docker CLI.
func NewDockerRuntime(e exec.Executor, cfg DockerConfig) Runtime {
	parser := &dockerParser{}
	return &dockerRuntime{
		baseRuntime: baseRuntime{
			exec:        e,
			binaryName:  "docker",
			execCommand: []string{"docker", "exec"},
			listArgs:    []string{"ps", "-a"},
			parser:      parser,
		},
		config: cfg,
	}
}

// dockerInspect represents the JSON output of `docker inspect`.
type dockerInspect struct {
	ID      string `json:"Id"`
	Name    string `json:"Name"`
	Created string `json:"Created"`
	State   struct {
		Status string `json:"Status"`
	} `json:"State"`
	Config struct {
		Image string `json:"Image"`
	} `json:"Config"`
}

func (d *dockerInspect) toContainer() *Container {
	status := parseContainerStatus(d.State.Status)

	// Remove leading "/" from name if present (Docker uses /container-name)
	name := strings.TrimPrefix(d.Name, "/")

	// Docker inspect returns Config.Image as the image name
	image := d.Config.Image

	// Docker uses RFC3339Nano format, fall back to RFC3339
	createdAt, err := time.Parse(time.RFC3339Nano, d.Created)
	if err != nil {
		createdAt, err = time.Parse(time.RFC3339, d.Created)
		if err != nil {
			createdAt = time.Time{}
		}
	}

	return &Container{
		ID:        d.ID,
		Name:      name,
		Image:     image,
		Status:    status,
		CreatedAt: createdAt,
	}
}

// dockerListItem represents a single item in `docker ps --format json` output.
// Note: Docker outputs one JSON object per line (NDJSON), not an array.
type dockerListItem struct {
	ID    string `json:"ID"`    // Note: uppercase ID, not Id like Podman
	Names string `json:"Names"` // String, not array like Podman
	Image string `json:"Image"`
	State string `json:"State"` // "running", "exited", etc.
}

func (d *dockerListItem) toContainer() Container {
	status := parseContainerStatus(d.State)

	return Container{
		ID:        d.ID,
		Name:      d.Names,
		Image:     d.Image,
		Status:    status,
		CreatedAt: time.Time{}, // Docker ps doesn't provide parseable timestamp
	}
}

// parseInspect parses the JSON output of `docker inspect`.
func (p *dockerParser) parseInspect(data []byte) (*Container, error) {
	var infos []dockerInspect
	if err := json.Unmarshal(data, &infos); err != nil {
		return nil, fmt.Errorf("parse container info: %w", err)
	}

	if len(infos) == 0 {
		return nil, ErrNotFound
	}

	return infos[0].toContainer(), nil
}

// parseList parses the JSON output of `docker ps --format json`.
// Docker outputs NDJSON (newline-delimited JSON), one object per line.
func (p *dockerParser) parseList(data []byte) ([]Container, error) {
	// Handle empty output
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "[]" {
		return []Container{}, nil
	}

	// Docker outputs NDJSON - one JSON object per line
	lines := strings.Split(trimmed, "\n")
	containers := make([]Container, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var item dockerListItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, fmt.Errorf("parse container list item: %w", err)
		}
		containers = append(containers, item.toContainer())
	}

	return containers, nil
}
