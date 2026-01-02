package container

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmgilman/headjack/internal/exec"
)

// AppleConfig holds Apple Containerization-specific runtime configuration.
type AppleConfig struct {
	// Currently empty - all flags go through RunConfig.Flags after merging
	// at the manager level. Kept for future runtime-specific settings.
}

// appleRuntime implements Runtime using Apple Containerization CLI.
// All common functionality is provided by the embedded baseRuntime.
type appleRuntime struct {
	baseRuntime
	config AppleConfig
}

// appleParser implements containerParser for Apple Containerization JSON output.
type appleParser struct{}

// NewAppleRuntime creates a Runtime using Apple Containerization CLI.
func NewAppleRuntime(e exec.Executor, cfg AppleConfig) Runtime {
	parser := &appleParser{}
	return &appleRuntime{
		baseRuntime: baseRuntime{
			exec:        e,
			binaryName:  "container",
			execCommand: []string{"container", "exec"},
			listArgs:    []string{"list"},
			parser:      parser,
		},
		config: cfg,
	}
}

// appleInspect represents the JSON output of `container inspect`.
type appleInspect struct {
	Status        string `json:"status"`
	Created       string `json:"created"` // ISO 8601 format if available
	Configuration struct {
		ID    string `json:"id"`
		Image struct {
			Reference string `json:"reference"`
		} `json:"image"`
	} `json:"configuration"`
}

func (c *appleInspect) toContainer() *Container {
	// Parse created timestamp if available
	// Apple Containerization uses ISO 8601 format
	var createdAt time.Time
	if c.Created != "" {
		// Try RFC3339Nano first (most precise), then RFC3339
		if parsed, err := time.Parse(time.RFC3339Nano, c.Created); err == nil {
			createdAt = parsed
		} else if parsed, err := time.Parse(time.RFC3339, c.Created); err == nil {
			createdAt = parsed
		}
		// If parsing fails, createdAt remains zero value
	}

	return &Container{
		ID:        c.Configuration.ID,
		Name:      c.Configuration.ID,
		Image:     c.Configuration.Image.Reference,
		Status:    parseContainerStatus(c.Status),
		CreatedAt: createdAt,
	}
}

// appleListItem represents a single item in `container list` JSON output.
// Note: Apple container list has similar format to inspect.
type appleListItem struct {
	Status        string `json:"status"`
	Created       string `json:"created"` // ISO 8601 format if available
	Configuration struct {
		ID    string `json:"id"`
		Image struct {
			Reference string `json:"reference"`
		} `json:"image"`
	} `json:"configuration"`
}

func (c *appleListItem) toContainer() Container {
	// Parse created timestamp if available
	var createdAt time.Time
	if c.Created != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, c.Created); err == nil {
			createdAt = parsed
		} else if parsed, err := time.Parse(time.RFC3339, c.Created); err == nil {
			createdAt = parsed
		}
	}

	return Container{
		ID:        c.Configuration.ID,
		Name:      c.Configuration.ID,
		Image:     c.Configuration.Image.Reference,
		Status:    parseContainerStatus(c.Status),
		CreatedAt: createdAt,
	}
}

// parseInspect parses the JSON output of `container inspect`.
func (p *appleParser) parseInspect(data []byte) (*Container, error) {
	var infos []appleInspect
	if err := json.Unmarshal(data, &infos); err != nil {
		return nil, fmt.Errorf("parse container info: %w", err)
	}

	if len(infos) == 0 {
		return nil, ErrNotFound
	}

	return infos[0].toContainer(), nil
}

// parseList parses the JSON output of `container list`.
func (p *appleParser) parseList(data []byte) ([]Container, error) {
	var items []appleListItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse container list: %w", err)
	}

	containers := make([]Container, len(items))
	for i, item := range items {
		containers[i] = item.toContainer()
	}

	return containers, nil
}
