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
	"github.com/jmgilman/headjack/internal/logging"
	"github.com/jmgilman/headjack/internal/multiplexer"
	"github.com/jmgilman/headjack/internal/names"
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

// sessionMultiplexer is the internal interface for multiplexer operations.
type sessionMultiplexer interface {
	CreateSession(ctx context.Context, opts *multiplexer.CreateSessionOpts) (*multiplexer.Session, error)
	AttachSession(ctx context.Context, sessionName string) error
	ListSessions(ctx context.Context) ([]multiplexer.Session, error)
	KillSession(ctx context.Context, sessionName string) error
}

// ManagerConfig configures the Manager.
type ManagerConfig struct {
	WorktreesDir string // Directory for storing worktrees (e.g., ~/.local/share/headjack/git)
	LogsDir      string // Directory for storing logs (e.g., ~/.local/share/headjack/logs)
}

// Manager orchestrates instance lifecycle operations.
type Manager struct {
	catalog      catalogStore
	runtime      containerRuntime
	git          gitOpener
	mux          sessionMultiplexer
	logPaths     *logging.PathManager
	worktreesDir string
}

// NewManager creates a new instance manager.
func NewManager(store catalogStore, runtime containerRuntime, opener gitOpener, mux sessionMultiplexer, cfg ManagerConfig) *Manager {
	return &Manager{
		catalog:      store,
		runtime:      runtime,
		git:          opener,
		mux:          mux,
		logPaths:     logging.NewPathManager(cfg.LogsDir),
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

// sessionNameExists checks if a session name already exists in the entry's sessions.
func sessionNameExists(sessions []catalog.Session, name string) bool {
	for i := range sessions {
		if sessions[i].Name == name {
			return true
		}
	}
	return false
}

// resolveSessionName generates or validates a session name.
// If name is empty, a unique name is auto-generated.
// If name is provided, it checks for conflicts and returns ErrSessionExists if found.
func resolveSessionName(sessions []catalog.Session, name string) (string, error) {
	if name == "" {
		existsFn := func(n string) bool {
			return sessionNameExists(sessions, n)
		}
		return names.GenerateUnique(existsFn, 100)
	}
	if sessionNameExists(sessions, name) {
		return "", ErrSessionExists
	}
	return name, nil
}

// CreateSession creates a new session within an instance.
// The session is created in detached mode within the container's multiplexer.
// If cfg.Name is empty, a unique name is auto-generated.
func (m *Manager) CreateSession(ctx context.Context, instanceID string, cfg *CreateSessionConfig) (*Session, error) {
	entry, err := m.getRunningInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	sessionID, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate session ID: %w", err)
	}

	sessionName, err := resolveSessionName(entry.Sessions, cfg.Name)
	if err != nil {
		return nil, err
	}

	muxSessionName, err := multiplexer.FormatSessionName(instanceID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("format session name: %w", err)
	}

	// Get the log path for output capture
	logPath, logErr := m.logPaths.EnsureSessionLog(instanceID, sessionID)
	if logErr != nil {
		return nil, fmt.Errorf("ensure session log: %w", logErr)
	}

	sessionType := catalog.SessionType(cfg.Type)
	if sessionType == "" {
		sessionType = catalog.SessionTypeShell
	}

	// Create multiplexer session with logging
	// Use the worktree path as cwd (host path, since zellij runs on host)
	_, err = m.mux.CreateSession(ctx, &multiplexer.CreateSessionOpts{
		Name:    muxSessionName,
		Command: cfg.Command,
		Cwd:     entry.Worktree,
		Env:     cfg.Env,
		LogPath: logPath,
	})
	if err != nil {
		return nil, fmt.Errorf("create multiplexer session: %w", err)
	}

	now := time.Now()
	catSession := catalog.Session{
		ID:           sessionID,
		Name:         sessionName,
		Type:         sessionType,
		MuxSessionID: muxSessionName,
		CreatedAt:    now,
		LastAccessed: now,
	}

	entry.Sessions = append(entry.Sessions, catSession)
	if updateErr := m.catalog.Update(ctx, entry); updateErr != nil {
		_ = m.mux.KillSession(ctx, muxSessionName) //nolint:errcheck // best-effort cleanup
		return nil, fmt.Errorf("update catalog entry: %w", updateErr)
	}

	return &Session{
		ID:           sessionID,
		Name:         sessionName,
		Type:         string(sessionType),
		MuxSessionID: muxSessionName,
		CreatedAt:    now,
		LastAccessed: now,
	}, nil
}

// getRunningInstance retrieves an instance and verifies its container is running.
func (m *Manager) getRunningInstance(ctx context.Context, instanceID string) (*catalog.Entry, error) {
	entry, err := m.catalog.Get(ctx, instanceID)
	if err != nil {
		if err == catalog.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get catalog entry: %w", err)
	}

	if entry.ContainerID == "" {
		return nil, errors.New("instance has no container")
	}

	c, err := m.runtime.Get(ctx, entry.ContainerID)
	if err != nil {
		return nil, fmt.Errorf("get container: %w", err)
	}

	if c.Status != container.StatusRunning {
		return nil, ErrInstanceNotRunning
	}

	return entry, nil
}

// GetSession retrieves a session by name within an instance.
func (m *Manager) GetSession(ctx context.Context, instanceID, sessionName string) (*Session, error) {
	entry, err := m.catalog.Get(ctx, instanceID)
	if err != nil {
		if err == catalog.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get catalog entry: %w", err)
	}

	for _, s := range entry.Sessions {
		if s.Name == sessionName {
			return &Session{
				ID:           s.ID,
				Name:         s.Name,
				Type:         string(s.Type),
				MuxSessionID: s.MuxSessionID,
				CreatedAt:    s.CreatedAt,
				LastAccessed: s.LastAccessed,
			}, nil
		}
	}

	return nil, ErrSessionNotFound
}

// ListSessions returns all sessions for an instance.
func (m *Manager) ListSessions(ctx context.Context, instanceID string) ([]Session, error) {
	entry, err := m.catalog.Get(ctx, instanceID)
	if err != nil {
		if err == catalog.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get catalog entry: %w", err)
	}

	sessions := make([]Session, len(entry.Sessions))
	for i, s := range entry.Sessions {
		sessions[i] = Session{
			ID:           s.ID,
			Name:         s.Name,
			Type:         string(s.Type),
			MuxSessionID: s.MuxSessionID,
			CreatedAt:    s.CreatedAt,
			LastAccessed: s.LastAccessed,
		}
	}

	return sessions, nil
}

// KillSession terminates a session and removes it from the catalog.
func (m *Manager) KillSession(ctx context.Context, instanceID, sessionName string) error {
	entry, err := m.catalog.Get(ctx, instanceID)
	if err != nil {
		if err == catalog.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("get catalog entry: %w", err)
	}

	// Find the session
	var sessionIndex = -1
	var session catalog.Session
	for i, s := range entry.Sessions {
		if s.Name == sessionName {
			sessionIndex = i
			session = s
			break
		}
	}
	if sessionIndex == -1 {
		return ErrSessionNotFound
	}

	// Kill the multiplexer session (best-effort)
	if killErr := m.mux.KillSession(ctx, session.MuxSessionID); killErr != nil {
		// Only return error if it's not "session not found" (already dead)
		if !errors.Is(killErr, multiplexer.ErrSessionNotFound) {
			return fmt.Errorf("kill multiplexer session: %w", killErr)
		}
	}

	// Remove session log (best-effort)
	_ = m.logPaths.RemoveSessionLog(instanceID, session.ID) //nolint:errcheck // best-effort cleanup

	// Remove session from entry and persist
	entry.Sessions = append(entry.Sessions[:sessionIndex], entry.Sessions[sessionIndex+1:]...)
	if updateErr := m.catalog.Update(ctx, entry); updateErr != nil {
		return fmt.Errorf("update catalog entry: %w", updateErr)
	}

	return nil
}

// AttachSession attaches to an existing session, updating the last accessed timestamp.
// This is a blocking operation that takes over the terminal.
func (m *Manager) AttachSession(ctx context.Context, instanceID, sessionName string) error {
	entry, err := m.catalog.Get(ctx, instanceID)
	if err != nil {
		if err == catalog.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("get catalog entry: %w", err)
	}

	// Find the session
	var sessionIndex = -1
	var session catalog.Session
	for i, s := range entry.Sessions {
		if s.Name == sessionName {
			sessionIndex = i
			session = s
			break
		}
	}
	if sessionIndex == -1 {
		return ErrSessionNotFound
	}

	// Update last accessed timestamp
	entry.Sessions[sessionIndex].LastAccessed = time.Now()
	if updateErr := m.catalog.Update(ctx, entry); updateErr != nil {
		return fmt.Errorf("update catalog entry: %w", updateErr)
	}

	// Attach to the multiplexer session
	return m.mux.AttachSession(ctx, session.MuxSessionID)
}

// GetMRUSession returns the most recently used session for an instance.
func (m *Manager) GetMRUSession(ctx context.Context, instanceID string) (*Session, error) {
	entry, err := m.catalog.Get(ctx, instanceID)
	if err != nil {
		if err == catalog.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get catalog entry: %w", err)
	}

	if len(entry.Sessions) == 0 {
		return nil, ErrNoSessionsAvailable
	}

	// Find the session with the most recent LastAccessed timestamp
	mru := &entry.Sessions[0]
	for i := range entry.Sessions {
		if entry.Sessions[i].LastAccessed.After(mru.LastAccessed) {
			mru = &entry.Sessions[i]
		}
	}

	return &Session{
		ID:           mru.ID,
		Name:         mru.Name,
		Type:         string(mru.Type),
		MuxSessionID: mru.MuxSessionID,
		CreatedAt:    mru.CreatedAt,
		LastAccessed: mru.LastAccessed,
	}, nil
}

// GlobalMRUSession represents a session with its instance context.
type GlobalMRUSession struct {
	InstanceID string
	Session    Session
}

// GetGlobalMRUSession returns the most recently used session across all instances.
func (m *Manager) GetGlobalMRUSession(ctx context.Context) (*GlobalMRUSession, error) {
	entries, err := m.catalog.List(ctx, catalog.ListFilter{})
	if err != nil {
		return nil, fmt.Errorf("list catalog entries: %w", err)
	}

	var globalMRU *GlobalMRUSession
	var latestAccessed time.Time

	for i := range entries {
		entry := &entries[i]
		for j := range entry.Sessions {
			s := &entry.Sessions[j]
			if globalMRU == nil || s.LastAccessed.After(latestAccessed) {
				latestAccessed = s.LastAccessed
				globalMRU = &GlobalMRUSession{
					InstanceID: entry.ID,
					Session: Session{
						ID:           s.ID,
						Name:         s.Name,
						Type:         string(s.Type),
						MuxSessionID: s.MuxSessionID,
						CreatedAt:    s.CreatedAt,
						LastAccessed: s.LastAccessed,
					},
				}
			}
		}
	}

	if globalMRU == nil {
		return nil, ErrNoSessionsAvailable
	}

	return globalMRU, nil
}
