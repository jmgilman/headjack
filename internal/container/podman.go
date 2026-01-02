package container

import (
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

// podmanRuntime implements Runtime using Podman CLI.
// All common functionality is provided by the embedded baseRuntime.
type podmanRuntime struct {
	baseRuntime
	config PodmanConfig
}

// podmanParser implements containerParser for Podman JSON output.
type podmanParser struct{}

// NewPodmanRuntime creates a Runtime using Podman CLI.
func NewPodmanRuntime(e exec.Executor, cfg PodmanConfig) Runtime {
	parser := &podmanParser{}
	return &podmanRuntime{
		baseRuntime: baseRuntime{
			exec:        e,
			binaryName:  "podman",
			execCommand: []string{"podman", "exec"},
			listArgs:    []string{"ps", "-a"},
			parser:      parser,
		},
		config: cfg,
	}
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

// parseInspect parses the JSON output of `podman inspect`.
func (p *podmanParser) parseInspect(data []byte) (*Container, error) {
	var infos []podmanInspect
	if err := json.Unmarshal(data, &infos); err != nil {
		return nil, fmt.Errorf("parse container info: %w", err)
	}

	if len(infos) == 0 {
		return nil, ErrNotFound
	}

	return infos[0].toContainer(), nil
}

// parseList parses the JSON output of `podman ps`.
func (p *podmanParser) parseList(data []byte) ([]Container, error) {
	var items []podmanListItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse container list: %w", err)
	}

	containers := make([]Container, len(items))
	for i, item := range items {
		containers[i] = item.toContainer()
	}

	return containers, nil
}
