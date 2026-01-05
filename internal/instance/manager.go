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
	"github.com/jmgilman/headjack/internal/exec"
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
	Exec(ctx context.Context, id string, cfg *container.ExecConfig) error
	Stop(ctx context.Context, id string) error
	Start(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*container.Container, error)
	List(ctx context.Context, filter container.ListFilter) ([]container.Container, error)
	ExecCommand() []string
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

// RuntimeType identifies the container runtime being used.
type RuntimeType string

// Runtime type constants.
const (
	RuntimePodman RuntimeType = "podman"
	RuntimeDocker RuntimeType = "docker"
)

// ManagerConfig configures the Manager.
type ManagerConfig struct {
	WorktreesDir string        // Directory for storing worktrees (e.g., ~/.local/share/headjack/git)
	LogsDir      string        // Directory for storing logs (e.g., ~/.local/share/headjack/logs)
	RuntimeType  RuntimeType   // Container runtime type (docker or podman)
	ConfigFlags  []string      // Additional flags to pass to the container runtime
	Executor     exec.Executor // Command executor (for devcontainer runtime creation)
}

// Manager orchestrates instance lifecycle operations.
type Manager struct {
	catalog      catalogStore
	runtime      containerRuntime
	publicRT     container.Runtime // Public runtime interface (same as runtime, exposed for devcontainer)
	executor     exec.Executor
	git          gitOpener
	mux          sessionMultiplexer
	logPaths     *logging.PathManager
	worktreesDir string
	runtimeType  RuntimeType
	configFlags  []string
}

// NewManager creates a new instance manager.
func NewManager(store catalogStore, runtime containerRuntime, opener gitOpener, mux sessionMultiplexer, cfg *ManagerConfig) *Manager {
	runtimeType := cfg.RuntimeType
	if runtimeType == "" {
		runtimeType = RuntimeDocker
	}

	// Type assert to get public runtime interface (all container.Runtime implementations satisfy containerRuntime)
	var publicRT container.Runtime
	if rt, ok := runtime.(container.Runtime); ok {
		publicRT = rt
	}

	return &Manager{
		catalog:      store,
		runtime:      runtime,
		publicRT:     publicRT,
		executor:     cfg.Executor,
		git:          opener,
		mux:          mux,
		logPaths:     logging.NewPathManager(cfg.LogsDir),
		worktreesDir: cfg.WorktreesDir,
		runtimeType:  runtimeType,
		configFlags:  cfg.ConfigFlags,
	}
}

// Runtime returns the underlying container runtime.
// This is used by the devcontainer runtime to delegate lifecycle operations.
func (m *Manager) Runtime() container.Runtime {
	return m.publicRT
}

// Executor returns the command executor.
// This is used by the devcontainer runtime to execute CLI commands.
func (m *Manager) Executor() exec.Executor {
	return m.executor
}

// Create creates a new instance for the given repository and branch.
func (m *Manager) Create(ctx context.Context, repoPath string, cfg *CreateConfig) (*Instance, error) {
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

	// Select runtime: use provided override or default to manager's runtime
	runtime := m.selectRuntime(cfg.Runtime)

	// Build container run config based on mode (devcontainer vs vanilla)
	runCfg := m.buildRunConfig(cfg, containerName, worktreePath)

	// Create container
	c, err := runtime.Run(ctx, runCfg)
	if err != nil {
		// Cleanup worktree on container failure
		if wtErr := repo.RemoveWorktree(ctx, worktreePath); wtErr != nil {
			cleanup()
			return nil, fmt.Errorf("create container: %w (additionally, failed to remove worktree: %v)", err, wtErr)
		}
		cleanup()
		return nil, fmt.Errorf("create container: %w", err)
	}

	// Update catalog with container info (including devcontainer-specific fields if present)
	entry.ContainerID = c.ID
	entry.Status = catalog.StatusRunning
	entry.RemoteUser = c.RemoteUser
	entry.RemoteWorkdir = c.RemoteWorkspaceFolder
	if updateErr := m.catalog.Update(ctx, &entry); updateErr != nil {
		// Cleanup container - use retry logic in case of transient issues
		if stopErr := m.stopContainerWithRetry(ctx, c.ID); stopErr != nil && stopErr != container.ErrNotFound {
			// Container stop failed - return combined error so user knows cleanup failed
			cleanup()
			return nil, fmt.Errorf("update catalog entry: %w (additionally, failed to stop container: %v)", updateErr, stopErr)
		}
		if removeErr := runtime.Remove(ctx, c.ID); removeErr != nil && removeErr != container.ErrNotFound {
			cleanup()
			return nil, fmt.Errorf("update catalog entry: %w (additionally, failed to remove container: %v)", updateErr, removeErr)
		}
		// Cleanup worktree
		if wtErr := repo.RemoveWorktree(ctx, worktreePath); wtErr != nil {
			cleanup()
			return nil, fmt.Errorf("update catalog entry: %w (additionally, failed to remove worktree: %v)", updateErr, wtErr)
		}
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

// selectRuntime returns the provided runtime override if set, otherwise the manager's default.
func (m *Manager) selectRuntime(override containerRuntime) containerRuntime {
	if override != nil {
		return override
	}
	return m.runtime
}

// buildRunConfig creates a container.RunConfig based on the creation mode.
// For devcontainer mode (WorkspaceFolder set), it configures for devcontainer CLI.
// For vanilla mode, it applies config flags.
// CLI flags (from --) are appended after config flags for both modes.
func (m *Manager) buildRunConfig(cfg *CreateConfig, containerName, worktreePath string) *container.RunConfig {
	// Merge config flags with CLI flags (config first, CLI flags appended)
	flags := m.mergeFlags(cfg.RuntimeFlags)

	// Devcontainer mode: minimal config, devcontainer CLI handles the rest
	if cfg.WorkspaceFolder != "" {
		return &container.RunConfig{
			Name:            containerName,
			WorkspaceFolder: cfg.WorkspaceFolder,
			Mounts: []container.Mount{
				{Source: worktreePath, Target: "/workspace", ReadOnly: false},
			},
			Flags: flags,
		}
	}

	// Vanilla mode: use merged flags
	return &container.RunConfig{
		Name:  containerName,
		Image: cfg.Image,
		Mounts: []container.Mount{
			{Source: worktreePath, Target: "/workspace", ReadOnly: false},
		},
		Flags: flags,
	}
}

// mergeFlags combines config flags with CLI flags.
// Config flags come first, CLI flags are appended (allowing override via runtime behavior).
func (m *Manager) mergeFlags(cliFlags []string) []string {
	if len(m.configFlags) == 0 && len(cliFlags) == 0 {
		return nil
	}

	result := make([]string, 0, len(m.configFlags)+len(cliFlags))
	result = append(result, m.configFlags...)
	result = append(result, cliFlags...)
	return result
}

// Get retrieves an instance by ID, including live container status.
func (m *Manager) Get(ctx context.Context, id string) (*Instance, error) {
	entry, err := m.catalog.Get(ctx, id)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
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
		if errors.Is(err, catalog.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get catalog entry: %w", err)
	}

	return m.entryToInstance(ctx, entry)
}

// List returns all instances matching the filter.
// Instances with degraded containers (e.g., container not found) are silently
// skipped to ensure the list operation completes even if some instances have issues.
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
			// Skip degraded instances (e.g., container not found) to ensure
			// the list operation completes successfully
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
		if errors.Is(err, catalog.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("get catalog entry: %w", err)
	}

	if err := m.shutdownContainer(ctx, entry, shutdownContainerOpts{RemoveContainer: false}); err != nil {
		return err
	}

	entry.Status = catalog.StatusStopped
	if err := m.catalog.Update(ctx, entry); err != nil {
		return fmt.Errorf("update catalog entry: %w", err)
	}

	return nil
}

// Start starts a stopped instance's container.
func (m *Manager) Start(ctx context.Context, id string) error {
	entry, err := m.catalog.Get(ctx, id)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("get catalog entry: %w", err)
	}

	if entry.ContainerID == "" {
		return errors.New("instance has no container")
	}

	if err := m.runtime.Start(ctx, entry.ContainerID); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	entry.Status = catalog.StatusRunning
	if err := m.catalog.Update(ctx, entry); err != nil {
		return fmt.Errorf("update catalog entry: %w", err)
	}

	return nil
}

// waitForSessionsTerminated polls until all sessions are gone or timeout.
func (m *Manager) waitForSessionsTerminated(ctx context.Context, sessions []catalog.Session) error {
	// Build a set of session names to wait for
	waiting := make(map[string]bool)
	for _, s := range sessions {
		waiting[s.MuxSessionID] = true
	}

	// Poll for up to 5 seconds
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return errors.New("timeout waiting for sessions to terminate")
		case <-ticker.C:
			liveSessions, err := m.mux.ListSessions(ctx)
			if err != nil {
				// If we can't list, assume they're gone
				return nil
			}

			// Check if any of our sessions are still alive
			stillAlive := false
			for _, live := range liveSessions {
				if waiting[live.Name] {
					stillAlive = true
					break
				}
			}

			if !stillAlive {
				return nil
			}
		}
	}
}

// shutdownContainerOpts configures the shutdownContainer helper.
type shutdownContainerOpts struct {
	// RemoveContainer specifies whether to remove the container after stopping.
	RemoveContainer bool
}

// shutdownContainer kills all sessions and stops (and optionally removes) the container.
// This is the common shutdown sequence used by Stop, Remove, and Recreate.
// It modifies entry.Sessions to nil after killing sessions.
// The entry is NOT persisted to the catalog; the caller is responsible for that.
func (m *Manager) shutdownContainer(ctx context.Context, entry *catalog.Entry, opts shutdownContainerOpts) error {
	// Kill all sessions before stopping the container.
	// The container stop will fail with "Resource busy" if there are active
	// multiplexer sessions connected to processes inside the container.
	if m.mux != nil && len(entry.Sessions) > 0 {
		for _, sess := range entry.Sessions {
			// Best-effort kill - session may already be dead
			_ = m.mux.KillSession(ctx, sess.MuxSessionID) //nolint:errcheck

			// Remove session log (best-effort)
			_ = m.logPaths.RemoveSessionLog(entry.ID, sess.ID) //nolint:errcheck
		}

		// Wait for sessions to fully terminate. The kill is async and processes
		// inside the container may still be running briefly after the kill returns.
		// Best-effort wait - we'll try to stop the container anyway
		_ = m.waitForSessionsTerminated(ctx, entry.Sessions) //nolint:errcheck
	}

	// Clear sessions from entry (caller must persist this change)
	entry.Sessions = nil

	// Stop container
	if entry.ContainerID != "" {
		if err := m.stopContainerWithRetry(ctx, entry.ContainerID); err != nil {
			if err != container.ErrNotFound {
				return fmt.Errorf("stop container: %w", err)
			}
		}

		// Remove container if requested
		if opts.RemoveContainer {
			if err := m.runtime.Remove(ctx, entry.ContainerID); err != nil {
				if err != container.ErrNotFound {
					return fmt.Errorf("remove container: %w", err)
				}
			}
		}
	}

	return nil
}

// stopContainerWithRetry attempts to stop a container, retrying on "Resource busy" errors.
// This handles the case where container processes spawned by killed sessions are still
// cleaning up when we first try to stop.
func (m *Manager) stopContainerWithRetry(ctx context.Context, containerID string) error {
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			if lastErr != nil {
				return lastErr
			}
			return errors.New("timeout waiting to stop container")
		default:
			lastErr = m.runtime.Stop(ctx, containerID)
			if lastErr == nil {
				return nil
			}
			// If it's not a "busy" error, return immediately
			if !strings.Contains(lastErr.Error(), "busy") {
				return lastErr
			}
			// Wait before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				// continue to retry
			}
		}
	}
}

// Remove removes an instance completely (container, worktree, catalog entry).
func (m *Manager) Remove(ctx context.Context, id string) error {
	entry, err := m.catalog.Get(ctx, id)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("get catalog entry: %w", err)
	}

	if err := m.shutdownContainer(ctx, entry, shutdownContainerOpts{RemoveContainer: true}); err != nil {
		return err
	}

	// Remove worktree
	if entry.Worktree != "" {
		repo, repoErr := m.git.Open(ctx, entry.Repo)
		if repoErr != nil {
			// Repository might have been deleted - that's ok, can't remove worktree anyway
			if repoErr != git.ErrNotRepository {
				return fmt.Errorf("open repository for worktree removal: %w", repoErr)
			}
		} else {
			if wtErr := repo.RemoveWorktree(ctx, entry.Worktree); wtErr != nil {
				// Worktree might already be gone - that's ok
				if wtErr != git.ErrWorktreeNotFound {
					return fmt.Errorf("remove worktree: %w", wtErr)
				}
			}
		}
	}

	// Remove instance logs directory (best-effort)
	_ = m.logPaths.RemoveInstanceLogs(id) //nolint:errcheck // best-effort cleanup

	// Remove catalog entry
	if err := m.catalog.Remove(ctx, id); err != nil {
		return fmt.Errorf("remove catalog entry: %w", err)
	}

	return nil
}

// Attach executes a command in an instance's container.
// If the instance is stopped, it will be started first.
func (m *Manager) Attach(ctx context.Context, id string, cfg AttachConfig) error {
	entry, err := m.catalog.Get(ctx, id)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
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

	// Determine workdir: use explicit config, then devcontainer remote workdir, then default
	workdir := cfg.Workdir
	if workdir == "" && entry.RemoteWorkdir != "" {
		workdir = entry.RemoteWorkdir
	}

	// Execute command (use RemoteUser for devcontainer instances)
	return m.runtime.Exec(ctx, entry.ContainerID, &container.ExecConfig{
		Command:     cmd,
		Interactive: cfg.Interactive,
		Workdir:     workdir,
		Env:         cfg.Env,
		User:        entry.RemoteUser,
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

	// Run agent-specific setup before starting the session
	if setupErr := m.runAgentSetup(ctx, entry.ContainerID, sessionType, cfg.Env, cfg.RequiresAgentSetup, entry.RemoteUser); setupErr != nil {
		return nil, fmt.Errorf("agent setup: %w", setupErr)
	}

	// Determine workdir: use devcontainer remote workdir if set, otherwise default to /workspace
	workdir := "/workspace"
	if entry.RemoteWorkdir != "" {
		workdir = entry.RemoteWorkdir
	}

	// Build the command to execute inside the container
	// The multiplexer runs on the host, so we wrap the command with the runtime's exec command
	execCmd := append(m.runtime.ExecCommand(), "-it")
	if entry.RemoteUser != "" {
		execCmd = append(execCmd, "-u", entry.RemoteUser)
	}
	execCmd = append(execCmd, "-w", workdir)
	for _, e := range cfg.Env {
		execCmd = append(execCmd, "-e", e)
	}
	execCmd = append(execCmd, entry.ContainerID)
	if len(cfg.Command) > 0 {
		execCmd = append(execCmd, cfg.Command...)
	} else {
		// Default to shell if no command specified
		execCmd = append(execCmd, "/bin/bash")
	}

	// Create multiplexer session with logging
	// The multiplexer runs on the host, executing the runtime's exec command to run inside the container
	_, err = m.mux.CreateSession(ctx, &multiplexer.CreateSessionOpts{
		Name:    muxSessionName,
		Command: execCmd,
		Cwd:     entry.Worktree,
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
		// Cleanup the multiplexer session we just created
		if killErr := m.mux.KillSession(ctx, muxSessionName); killErr != nil {
			// Session kill failed - return combined error so user knows cleanup failed
			return nil, fmt.Errorf("update catalog entry: %w (additionally, failed to kill session %q: %v)", updateErr, muxSessionName, killErr)
		}
		// Cleanup the session log file
		_ = m.logPaths.RemoveSessionLog(instanceID, sessionID) //nolint:errcheck // log cleanup is truly best-effort
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

// runAgentSetup performs agent-specific setup before starting a session.
// For Claude, this creates the config file needed to skip onboarding.
// For Gemini/Codex with subscription auth, this writes OAuth credentials to file locations.
// API key auth skips file setup since credentials are passed via environment variables.
// The user parameter specifies which user to run setup as (for devcontainer instances).
func (m *Manager) runAgentSetup(ctx context.Context, containerID string, sessionType catalog.SessionType, env []string, requiresSetup bool, user string) error {
	switch sessionType {
	case catalog.SessionTypeClaude:
		// Always create ~/.claude.json with hasCompletedOnboarding to skip interactive setup.
		// This is required for both OAuth token and API key authentication in headless environments.
		// See: https://github.com/anthropics/claude-code/issues/8938
		setupCmd := `mkdir -p ~/.claude && echo '{"hasCompletedOnboarding":true}' > ~/.claude.json`
		return m.runtime.Exec(ctx, containerID, &container.ExecConfig{
			Command: []string{"sh", "-c", setupCmd},
			User:    user,
		})

	case catalog.SessionTypeGemini:
		// Skip file setup for API key auth - credentials are in GEMINI_API_KEY env var
		if !requiresSetup {
			return nil
		}
		// Write Gemini config files from env vars for subscription auth.
		// GEMINI_OAUTH_CREDS contains JSON with oauth_creds and google_accounts.
		// We also write a minimal settings.json to set the auth type.
		setupCmd := `mkdir -p ~/.gemini && \
echo "$GEMINI_OAUTH_CREDS" | jq -r '.oauth_creds' > ~/.gemini/oauth_creds.json && \
echo "$GEMINI_OAUTH_CREDS" | jq -r '.google_accounts' > ~/.gemini/google_accounts.json && \
echo '{"security":{"auth":{"selectedType":"oauth-personal"}}}' > ~/.gemini/settings.json`
		return m.runtime.Exec(ctx, containerID, &container.ExecConfig{
			Command: []string{"sh", "-c", setupCmd},
			Env:     env,
			User:    user,
		})

	case catalog.SessionTypeCodex:
		// Skip file setup for API key auth - credentials are in OPENAI_API_KEY env var
		if !requiresSetup {
			return nil
		}
		// Write Codex auth.json from env var for subscription auth.
		// CODEX_AUTH_JSON contains the contents of ~/.codex/auth.json.
		setupCmd := `mkdir -p ~/.codex && echo "$CODEX_AUTH_JSON" > ~/.codex/auth.json`
		return m.runtime.Exec(ctx, containerID, &container.ExecConfig{
			Command: []string{"sh", "-c", setupCmd},
			Env:     env,
			User:    user,
		})

	default:
		return nil
	}
}

// getRunningInstance retrieves an instance and verifies its container is running.
func (m *Manager) getRunningInstance(ctx context.Context, instanceID string) (*catalog.Entry, error) {
	entry, err := m.catalog.Get(ctx, instanceID)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
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
		return nil, &NotRunningError{
			InstanceID:  entry.ID,
			ContainerID: entry.ContainerID,
			Status:      c.Status,
		}
	}

	return entry, nil
}

// GetSession retrieves a session by name within an instance.
func (m *Manager) GetSession(ctx context.Context, instanceID, sessionName string) (*Session, error) {
	entry, err := m.catalog.Get(ctx, instanceID)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
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
		if errors.Is(err, catalog.ErrNotFound) {
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
		if errors.Is(err, catalog.ErrNotFound) {
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
		if errors.Is(err, catalog.ErrNotFound) {
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

	// Attach to the multiplexer session (blocks until user exits or detaches)
	attachErr := m.mux.AttachSession(ctx, session.MuxSessionID)

	// After attach returns, check if session still exists in multiplexer.
	// If not, the user exited (not detached) so we clean up the catalog.
	m.cleanupExitedSession(ctx, instanceID, sessionName, session.MuxSessionID)

	return attachErr
}

// cleanupExitedSession removes a session from the catalog if it no longer exists in the multiplexer.
// This handles the case where a user exits a session (vs detaching).
func (m *Manager) cleanupExitedSession(ctx context.Context, instanceID, sessionName, muxSessionID string) {
	sessions, err := m.mux.ListSessions(ctx)
	if err != nil {
		return // Best effort - don't fail if we can't list sessions
	}

	// Check if our session still exists
	for _, s := range sessions {
		if s.Name == muxSessionID {
			return // Session still exists (user detached), nothing to clean up
		}
	}

	// Session no longer exists in multiplexer - remove from catalog
	// Re-fetch entry since it may have changed while we were attached
	entry, err := m.catalog.Get(ctx, instanceID)
	if err != nil {
		return
	}

	newSessions := make([]catalog.Session, 0, len(entry.Sessions))
	for _, s := range entry.Sessions {
		if s.Name != sessionName {
			newSessions = append(newSessions, s)
		}
	}
	entry.Sessions = newSessions

	//nolint:errcheck // Best-effort cleanup - don't fail command if catalog update fails
	m.catalog.Update(ctx, entry)
}

// GetMRUSession returns the most recently used session for an instance.
func (m *Manager) GetMRUSession(ctx context.Context, instanceID string) (*Session, error) {
	entry, err := m.catalog.Get(ctx, instanceID)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
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

// GlobalMRUSession represents the most recently used session along with
// its containing instance information. This is returned by GetGlobalMRUSession
// to provide context about which instance owns the session.
type GlobalMRUSession struct {
	InstanceID string  // ID of the instance containing the session
	Session    Session // The most recently used session
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
