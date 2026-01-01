package instance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jmgilman/headjack/internal/catalog"
	"github.com/jmgilman/headjack/internal/container"
	"github.com/jmgilman/headjack/internal/git"
)

// containerNamePrefix is the prefix for all managed containers.
const containerNamePrefix = "hjk"

// catalogStore is the internal interface for catalog operations.
type catalogStore interface {
	Add(ctx context.Context, entry *catalog.Entry) error
	Get(ctx context.Context, id string) (*catalog.Entry, error)
	GetByRepoBranch(ctx context.Context, repoID, branch string) (*catalog.Entry, error)
	Update(ctx context.Context, entry *catalog.Entry) error
	Remove(ctx context.Context, id string) error
	List(ctx context.Context, filter catalog.ListFilter) ([]catalog.Entry, error)
}

// containerRuntime is the internal interface for container operations.
type containerRuntime interface {
	Run(ctx context.Context, cfg *container.RunConfig) (*container.Container, error)
	Exec(ctx context.Context, id string, cfg container.ExecConfig) error
	Stop(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*container.Container, error)
	List(ctx context.Context, filter container.ListFilter) ([]container.Container, error)
}

// gitOpener is the internal interface for opening git repositories.
type gitOpener interface {
	Open(ctx context.Context, path string) (git.Repository, error)
}

// ManagerConfig configures the Manager.
type ManagerConfig struct {
	WorktreesDir string // Directory for storing worktrees (e.g., ~/.local/share/headjack/git)
}

// Manager orchestrates instance lifecycle operations.
type Manager struct {
	catalog      catalogStore
	runtime      containerRuntime
	git          gitOpener
	worktreesDir string
}

// NewManager creates a new instance manager.
func NewManager(store catalogStore, runtime containerRuntime, opener gitOpener, cfg ManagerConfig) *Manager {
	return &Manager{
		catalog:      store,
		runtime:      runtime,
		git:          opener,
		worktreesDir: cfg.WorktreesDir,
	}
}

// Create creates a new instance for the given repository and branch.
func (m *Manager) Create(ctx context.Context, repoPath string, cfg CreateConfig) (*Instance, error) {
	// Open the repository
	repo, err := m.git.Open(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("open repository: %w", err)
	}

	repoID := repo.Identifier()

	// Check if instance already exists for this branch
	_, err = m.catalog.GetByRepoBranch(ctx, repoID, cfg.Branch)
	if err == nil {
		return nil, ErrAlreadyExists
	}
	if err != catalog.ErrNotFound {
		return nil, fmt.Errorf("check existing instance: %w", err)
	}

	// Generate instance ID
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate instance ID: %w", err)
	}

	// Determine paths
	worktreePath := m.worktreePath(repoID, cfg.Branch)
	containerName := m.containerName(repoID, cfg.Branch)

	// Create catalog entry first (for tracking partial state)
	entry := catalog.Entry{
		ID:        id,
		Repo:      repo.Root(),
		RepoID:    repoID,
		Branch:    cfg.Branch,
		Worktree:  worktreePath,
		CreatedAt: time.Now(),
		Status:    catalog.StatusCreating,
	}
	if addErr := m.catalog.Add(ctx, &entry); addErr != nil {
		return nil, fmt.Errorf("add catalog entry: %w", addErr)
	}

	// Cleanup on failure
	cleanup := func() {
		_ = m.catalog.Remove(ctx, id) //nolint:errcheck // best-effort cleanup
	}

	// Create worktree
	if wtErr := repo.CreateWorktree(ctx, worktreePath, cfg.Branch); wtErr != nil {
		cleanup()
		return nil, fmt.Errorf("create worktree: %w", wtErr)
	}

	// Create container
	c, err := m.runtime.Run(ctx, &container.RunConfig{
		Name:  containerName,
		Image: cfg.Image,
		Mounts: []container.Mount{
			{Source: worktreePath, Target: "/workspace", ReadOnly: false},
		},
	})
	if err != nil {
		// Cleanup worktree on container failure
		_ = repo.RemoveWorktree(ctx, worktreePath) //nolint:errcheck // best-effort cleanup
		cleanup()
		return nil, fmt.Errorf("create container: %w", err)
	}

	// Update catalog with container info
	entry.ContainerID = c.ID
	entry.Status = catalog.StatusRunning
	if updateErr := m.catalog.Update(ctx, &entry); updateErr != nil {
		// Best-effort cleanup
		_ = m.runtime.Stop(ctx, c.ID)              //nolint:errcheck // best-effort cleanup
		_ = m.runtime.Remove(ctx, c.ID)            //nolint:errcheck // best-effort cleanup
		_ = repo.RemoveWorktree(ctx, worktreePath) //nolint:errcheck // best-effort cleanup
		cleanup()
		return nil, fmt.Errorf("update catalog entry: %w", updateErr)
	}

	return &Instance{
		ID:          id,
		Repo:        repo.Root(),
		RepoID:      repoID,
		Branch:      cfg.Branch,
		Worktree:    worktreePath,
		ContainerID: c.ID,
		Container:   c,
		CreatedAt:   entry.CreatedAt,
		Status:      StatusRunning,
	}, nil
}

// Get retrieves an instance by ID, including live container status.
func (m *Manager) Get(ctx context.Context, id string) (*Instance, error) {
	entry, err := m.catalog.Get(ctx, id)
	if err != nil {
		if err == catalog.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get catalog entry: %w", err)
	}

	return m.entryToInstance(ctx, entry)
}

// GetByBranch retrieves an instance by repository path and branch.
func (m *Manager) GetByBranch(ctx context.Context, repoPath, branch string) (*Instance, error) {
	repo, err := m.git.Open(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("open repository: %w", err)
	}

	entry, err := m.catalog.GetByRepoBranch(ctx, repo.Identifier(), branch)
	if err != nil {
		if err == catalog.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get catalog entry: %w", err)
	}

	return m.entryToInstance(ctx, entry)
}

// List returns all instances matching the filter.
func (m *Manager) List(ctx context.Context, filter ListFilter) ([]Instance, error) {
	entries, err := m.catalog.List(ctx, catalog.ListFilter{
		RepoID: filter.RepoID,
		Status: catalog.Status(filter.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("list catalog entries: %w", err)
	}

	instances := make([]Instance, 0, len(entries))
	for i := range entries {
		inst, err := m.entryToInstance(ctx, &entries[i])
		if err != nil {
			// Log and continue on individual failures
			continue
		}
		instances = append(instances, *inst)
	}

	return instances, nil
}

// Stop stops an instance's container.
func (m *Manager) Stop(ctx context.Context, id string) error {
	entry, err := m.catalog.Get(ctx, id)
	if err != nil {
		if err == catalog.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("get catalog entry: %w", err)
	}

	if entry.ContainerID != "" {
		if err := m.runtime.Stop(ctx, entry.ContainerID); err != nil {
			if err != container.ErrNotFound {
				return fmt.Errorf("stop container: %w", err)
			}
		}
	}

	entry.Status = catalog.StatusStopped
	if err := m.catalog.Update(ctx, entry); err != nil {
		return fmt.Errorf("update catalog entry: %w", err)
	}

	return nil
}

// Remove removes an instance completely (container, worktree, catalog entry).
func (m *Manager) Remove(ctx context.Context, id string) error {
	entry, err := m.catalog.Get(ctx, id)
	if err != nil {
		if err == catalog.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("get catalog entry: %w", err)
	}

	// Stop and remove container (best-effort)
	if entry.ContainerID != "" {
		_ = m.runtime.Stop(ctx, entry.ContainerID)   //nolint:errcheck // best-effort cleanup
		_ = m.runtime.Remove(ctx, entry.ContainerID) //nolint:errcheck // best-effort cleanup
	}

	// Remove worktree (best-effort)
	if entry.Worktree != "" {
		repo, repoErr := m.git.Open(ctx, entry.Repo)
		if repoErr == nil {
			_ = repo.RemoveWorktree(ctx, entry.Worktree) //nolint:errcheck // best-effort cleanup
		}
	}

	// Remove catalog entry
	if err := m.catalog.Remove(ctx, id); err != nil {
		return fmt.Errorf("remove catalog entry: %w", err)
	}

	return nil
}

// Recreate removes the container and creates a new one with the specified image.
func (m *Manager) Recreate(ctx context.Context, id, image string) (*Instance, error) {
	entry, err := m.catalog.Get(ctx, id)
	if err != nil {
		if err == catalog.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get catalog entry: %w", err)
	}

	// Stop and remove old container
	if entry.ContainerID != "" {
		_ = m.runtime.Stop(ctx, entry.ContainerID) //nolint:errcheck // best-effort cleanup
		if removeErr := m.runtime.Remove(ctx, entry.ContainerID); removeErr != nil {
			if removeErr != container.ErrNotFound {
				return nil, fmt.Errorf("remove old container: %w", removeErr)
			}
		}
	}

	// Create new container
	containerName := m.containerName(entry.RepoID, entry.Branch)
	c, err := m.runtime.Run(ctx, &container.RunConfig{
		Name:  containerName,
		Image: image,
		Mounts: []container.Mount{
			{Source: entry.Worktree, Target: "/workspace", ReadOnly: false},
		},
	})
	if err != nil {
		entry.Status = catalog.StatusError
		_ = m.catalog.Update(ctx, entry) //nolint:errcheck // best-effort status update
		return nil, fmt.Errorf("create container: %w", err)
	}

	// Update catalog
	entry.ContainerID = c.ID
	entry.Status = catalog.StatusRunning
	if err := m.catalog.Update(ctx, entry); err != nil {
		return nil, fmt.Errorf("update catalog entry: %w", err)
	}

	return &Instance{
		ID:          entry.ID,
		Repo:        entry.Repo,
		RepoID:      entry.RepoID,
		Branch:      entry.Branch,
		Worktree:    entry.Worktree,
		ContainerID: c.ID,
		Container:   c,
		CreatedAt:   entry.CreatedAt,
		Status:      StatusRunning,
	}, nil
}

// Attach executes a command in an instance's container.
// If the instance is stopped, it will be started first.
func (m *Manager) Attach(ctx context.Context, id string, cfg AttachConfig) error {
	entry, err := m.catalog.Get(ctx, id)
	if err != nil {
		if err == catalog.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("get catalog entry: %w", err)
	}

	if entry.ContainerID == "" {
		return errors.New("instance has no container")
	}

	// Check container status
	c, err := m.runtime.Get(ctx, entry.ContainerID)
	if err != nil {
		return fmt.Errorf("get container: %w", err)
	}

	if c.Status != container.StatusRunning {
		return errors.New("container is not running")
	}

	// Build command (default to shell if empty)
	cmd := cfg.Command
	if len(cmd) == 0 {
		cmd = []string{"/bin/bash"}
	}

	// Execute command
	return m.runtime.Exec(ctx, entry.ContainerID, container.ExecConfig{
		Command:     cmd,
		Interactive: cfg.Interactive,
		Workdir:     cfg.Workdir,
		Env:         cfg.Env,
	})
}

// entryToInstance converts a catalog entry to an instance, fetching live container status.
func (m *Manager) entryToInstance(ctx context.Context, entry *catalog.Entry) (*Instance, error) {
	inst := &Instance{
		ID:          entry.ID,
		Repo:        entry.Repo,
		RepoID:      entry.RepoID,
		Branch:      entry.Branch,
		Worktree:    entry.Worktree,
		ContainerID: entry.ContainerID,
		CreatedAt:   entry.CreatedAt,
		Status:      catalogStatusToInstanceStatus(entry.Status),
	}

	// Fetch live container status if we have a container ID
	if entry.ContainerID != "" {
		c, err := m.runtime.Get(ctx, entry.ContainerID)
		if err == nil {
			inst.Container = c
			// Update status based on live container state
			switch c.Status {
			case container.StatusRunning:
				inst.Status = StatusRunning
			case container.StatusStopped:
				inst.Status = StatusStopped
			default:
				inst.Status = StatusError
			}
		}
	}

	return inst, nil
}

// worktreePath returns the path for a worktree.
func (m *Manager) worktreePath(repoID, branch string) string {
	return filepath.Join(m.worktreesDir, repoID, sanitizeBranch(branch))
}

// containerName returns the container name for an instance.
func (m *Manager) containerName(repoID, branch string) string {
	return fmt.Sprintf("%s-%s-%s", containerNamePrefix, repoID, sanitizeBranch(branch))
}

// sanitizeBranch converts a branch name to a valid container/path name.
func sanitizeBranch(branch string) string {
	// Replace slashes with dashes
	s := strings.ReplaceAll(branch, "/", "-")
	// Remove any invalid characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9-_]`)
	s = reg.ReplaceAllString(s, "")
	// Trim leading/trailing dashes
	s = strings.Trim(s, "-")
	return s
}

// generateID generates a random 8-character hex ID.
func generateID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// catalogStatusToInstanceStatus converts catalog status to instance status.
func catalogStatusToInstanceStatus(s catalog.Status) Status {
	switch s {
	case catalog.StatusRunning:
		return StatusRunning
	case catalog.StatusStopped:
		return StatusStopped
	default:
		return StatusError
	}
}
