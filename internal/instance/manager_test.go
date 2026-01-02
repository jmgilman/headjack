package instance

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmgilman/headjack/internal/catalog"
	catalogmocks "github.com/jmgilman/headjack/internal/catalog/mocks"
	"github.com/jmgilman/headjack/internal/container"
	containermocks "github.com/jmgilman/headjack/internal/container/mocks"
	"github.com/jmgilman/headjack/internal/git"
	gitmocks "github.com/jmgilman/headjack/internal/git/mocks"
	"github.com/jmgilman/headjack/internal/multiplexer"
	muxmocks "github.com/jmgilman/headjack/internal/multiplexer/mocks"
	"github.com/jmgilman/headjack/internal/registry"
	registrymocks "github.com/jmgilman/headjack/internal/registry/mocks"
)

// Test constants for repeated values.
const (
	testRepoID   = "myrepo-abc123"
	testRepoPath = "/path/to/repo"
)

func TestNewManager(t *testing.T) {
	t.Run("sets worktrees directory", func(t *testing.T) {
		mgr := NewManager(nil, nil, nil, nil, nil, ManagerConfig{WorktreesDir: "/data/worktrees", LogsDir: "/data/logs"})

		require.NotNil(t, mgr)
		assert.Equal(t, "/data/worktrees", mgr.worktreesDir)
	})

	t.Run("defaults RuntimeType to Podman when not specified", func(t *testing.T) {
		mgr := NewManager(nil, nil, nil, nil, nil, ManagerConfig{})

		assert.Equal(t, RuntimePodman, mgr.runtimeType)
	})

	t.Run("respects explicit RuntimeType", func(t *testing.T) {
		mgr := NewManager(nil, nil, nil, nil, nil, ManagerConfig{RuntimeType: RuntimeApple})

		assert.Equal(t, RuntimeApple, mgr.runtimeType)
	})
}

func TestManager_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates instance successfully", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return testRepoID },
			RootFunc:       func() string { return testRepoPath },
			CreateWorktreeFunc: func(ctx context.Context, path, branch string) error {
				return nil
			},
		}
		opener := &gitmocks.OpenerMock{
			OpenFunc: func(ctx context.Context, path string) (git.Repository, error) {
				return repo, nil
			},
		}
		store := &catalogmocks.StoreMock{
			GetByRepoBranchFunc: func(ctx context.Context, repoID, branch string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
			AddFunc: func(ctx context.Context, entry *catalog.Entry) error {
				return nil
			},
			UpdateFunc: func(ctx context.Context, entry *catalog.Entry) error {
				return nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			RunFunc: func(ctx context.Context, cfg *container.RunConfig) (*container.Container, error) {
				return &container.Container{
					ID:     "container-123",
					Name:   cfg.Name,
					Image:  cfg.Image,
					Status: container.StatusRunning,
				}, nil
			},
		}

		mgr := NewManager(store, runtime, opener, nil, nil, ManagerConfig{WorktreesDir: "/data/worktrees", LogsDir: "/data/logs"})

		inst, err := mgr.Create(ctx, "/path/to/repo", CreateConfig{
			Branch: "feature/auth",
			Image:  "myimage:latest",
		})

		require.NoError(t, err)
		require.NotNil(t, inst)
		assert.Equal(t, testRepoID, inst.RepoID)
		assert.Equal(t, "feature/auth", inst.Branch)
		assert.Equal(t, "container-123", inst.ContainerID)
		assert.Equal(t, StatusRunning, inst.Status)
		assert.Contains(t, inst.Worktree, "/data/worktrees/myrepo-abc123/feature-auth")

		// Verify container was created with correct config
		require.Len(t, runtime.RunCalls(), 1)
		runCfg := runtime.RunCalls()[0].Cfg
		assert.Equal(t, "hjk-myrepo-abc123-feature-auth", runCfg.Name)
		assert.Equal(t, "myimage:latest", runCfg.Image)
		require.Len(t, runCfg.Mounts, 1)
		assert.Equal(t, "/workspace", runCfg.Mounts[0].Target)
	})

	t.Run("returns ErrAlreadyExists for duplicate branch", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return testRepoID },
		}
		opener := &gitmocks.OpenerMock{
			OpenFunc: func(ctx context.Context, path string) (git.Repository, error) {
				return repo, nil
			},
		}
		store := &catalogmocks.StoreMock{
			GetByRepoBranchFunc: func(ctx context.Context, repoID, branch string) (*catalog.Entry, error) {
				return &catalog.Entry{ID: "existing"}, nil
			},
		}

		mgr := NewManager(store, nil, opener, nil, nil, ManagerConfig{WorktreesDir: "/data/worktrees", LogsDir: "/data/logs"})

		_, err := mgr.Create(ctx, testRepoPath, CreateConfig{Branch: "main"})

		assert.ErrorIs(t, err, ErrAlreadyExists)
	})

	t.Run("cleans up on worktree failure", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return testRepoID },
			RootFunc:       func() string { return testRepoPath },
			CreateWorktreeFunc: func(ctx context.Context, path, branch string) error {
				return errors.New("worktree error")
			},
		}
		opener := &gitmocks.OpenerMock{
			OpenFunc: func(ctx context.Context, path string) (git.Repository, error) {
				return repo, nil
			},
		}
		store := &catalogmocks.StoreMock{
			GetByRepoBranchFunc: func(ctx context.Context, repoID, branch string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
			AddFunc: func(ctx context.Context, entry *catalog.Entry) error {
				return nil
			},
			RemoveFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}

		mgr := NewManager(store, nil, opener, nil, nil, ManagerConfig{WorktreesDir: "/data/worktrees", LogsDir: "/data/logs"})

		_, err := mgr.Create(ctx, testRepoPath, CreateConfig{Branch: "main"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "worktree error")
		// Verify cleanup was called
		require.Len(t, store.RemoveCalls(), 1)
	})

	t.Run("cleans up on container failure", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return testRepoID },
			RootFunc:       func() string { return testRepoPath },
			CreateWorktreeFunc: func(ctx context.Context, path, branch string) error {
				return nil
			},
			RemoveWorktreeFunc: func(ctx context.Context, path string) error {
				return nil
			},
		}
		opener := &gitmocks.OpenerMock{
			OpenFunc: func(ctx context.Context, path string) (git.Repository, error) {
				return repo, nil
			},
		}
		store := &catalogmocks.StoreMock{
			GetByRepoBranchFunc: func(ctx context.Context, repoID, branch string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
			AddFunc: func(ctx context.Context, entry *catalog.Entry) error {
				return nil
			},
			RemoveFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			RunFunc: func(ctx context.Context, cfg *container.RunConfig) (*container.Container, error) {
				return nil, errors.New("container error")
			},
		}

		mgr := NewManager(store, runtime, opener, nil, nil, ManagerConfig{WorktreesDir: "/data/worktrees", LogsDir: "/data/logs"})

		_, err := mgr.Create(ctx, testRepoPath, CreateConfig{Branch: "main"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "container error")
		// Verify cleanup was called
		require.Len(t, repo.RemoveWorktreeCalls(), 1)
		require.Len(t, store.RemoveCalls(), 1)
	})
}

func TestManager_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("returns instance by ID", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc123",
					Repo:        testRepoPath,
					RepoID:      testRepoID,
					Branch:      "main",
					ContainerID: "container-123",
					Status:      catalog.StatusRunning,
				}, nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				return &container.Container{
					ID:     "container-123",
					Status: container.StatusRunning,
				}, nil
			},
		}

		mgr := NewManager(store, runtime, nil, nil, nil, ManagerConfig{})

		inst, err := mgr.Get(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, "abc123", inst.ID)
		assert.Equal(t, StatusRunning, inst.Status)
		require.NotNil(t, inst.Container)
	})

	t.Run("returns ErrNotFound for missing ID", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		_, err := mgr.Get(ctx, "nonexistent")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestManager_GetByBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("returns instance by repo and branch", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return testRepoID },
		}
		opener := &gitmocks.OpenerMock{
			OpenFunc: func(ctx context.Context, path string) (git.Repository, error) {
				return repo, nil
			},
		}
		store := &catalogmocks.StoreMock{
			GetByRepoBranchFunc: func(ctx context.Context, repoID, branch string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc123",
					RepoID:      repoID,
					Branch:      branch,
					ContainerID: "container-123",
					Status:      catalog.StatusRunning,
				}, nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				return &container.Container{
					ID:     "container-123",
					Status: container.StatusRunning,
				}, nil
			},
		}

		mgr := NewManager(store, runtime, opener, nil, nil, ManagerConfig{})

		inst, err := mgr.GetByBranch(ctx, "/path/to/repo", "main")

		require.NoError(t, err)
		assert.Equal(t, "abc123", inst.ID)
		assert.Equal(t, "main", inst.Branch)
	})

	t.Run("returns ErrNotFound for missing branch", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return testRepoID },
		}
		opener := &gitmocks.OpenerMock{
			OpenFunc: func(ctx context.Context, path string) (git.Repository, error) {
				return repo, nil
			},
		}
		store := &catalogmocks.StoreMock{
			GetByRepoBranchFunc: func(ctx context.Context, repoID, branch string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
		}

		mgr := NewManager(store, nil, opener, nil, nil, ManagerConfig{})

		_, err := mgr.GetByBranch(ctx, "/path/to/repo", "nonexistent")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestManager_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all instances", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			ListFunc: func(ctx context.Context, filter catalog.ListFilter) ([]catalog.Entry, error) {
				return []catalog.Entry{
					{ID: "a", ContainerID: "c1", Status: catalog.StatusRunning},
					{ID: "b", ContainerID: "c2", Status: catalog.StatusStopped},
				}, nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				if id == "c1" {
					return &container.Container{ID: "c1", Status: container.StatusRunning}, nil
				}
				return &container.Container{ID: "c2", Status: container.StatusStopped}, nil
			},
		}

		mgr := NewManager(store, runtime, nil, nil, nil, ManagerConfig{})

		instances, err := mgr.List(ctx, ListFilter{})

		require.NoError(t, err)
		assert.Len(t, instances, 2)
	})

	t.Run("filters by status", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			ListFunc: func(ctx context.Context, filter catalog.ListFilter) ([]catalog.Entry, error) {
				assert.Equal(t, catalog.Status(StatusRunning), filter.Status)
				return []catalog.Entry{
					{ID: "a", ContainerID: "c1", Status: catalog.StatusRunning},
				}, nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				return &container.Container{ID: "c1", Status: container.StatusRunning}, nil
			},
		}

		mgr := NewManager(store, runtime, nil, nil, nil, ManagerConfig{})

		instances, err := mgr.List(ctx, ListFilter{Status: StatusRunning})

		require.NoError(t, err)
		assert.Len(t, instances, 1)
	})
}

func TestManager_Stop(t *testing.T) {
	ctx := context.Background()

	t.Run("stops container and updates catalog", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc123",
					ContainerID: "container-123",
					Status:      catalog.StatusRunning,
				}, nil
			},
			UpdateFunc: func(ctx context.Context, entry *catalog.Entry) error {
				assert.Equal(t, catalog.StatusStopped, entry.Status)
				return nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			StopFunc: func(ctx context.Context, id string) error {
				assert.Equal(t, "container-123", id)
				return nil
			},
		}

		mgr := NewManager(store, runtime, nil, nil, nil, ManagerConfig{})

		err := mgr.Stop(ctx, "abc123")

		require.NoError(t, err)
		require.Len(t, runtime.StopCalls(), 1)
		require.Len(t, store.UpdateCalls(), 1)
	})

	t.Run("returns ErrNotFound for missing instance", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		err := mgr.Stop(ctx, "nonexistent")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestManager_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("removes container, worktree, and catalog entry", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			RemoveWorktreeFunc: func(ctx context.Context, path string) error {
				return nil
			},
		}
		opener := &gitmocks.OpenerMock{
			OpenFunc: func(ctx context.Context, path string) (git.Repository, error) {
				return repo, nil
			},
		}
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc123",
					Repo:        "/path/to/repo",
					ContainerID: "container-123",
					Worktree:    "/data/git/myrepo/main",
				}, nil
			},
			RemoveFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			StopFunc: func(ctx context.Context, id string) error {
				return nil
			},
			RemoveFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}

		mgr := NewManager(store, runtime, opener, nil, nil, ManagerConfig{})

		err := mgr.Remove(ctx, "abc123")

		require.NoError(t, err)
		require.Len(t, runtime.StopCalls(), 1)
		require.Len(t, runtime.RemoveCalls(), 1)
		require.Len(t, repo.RemoveWorktreeCalls(), 1)
		require.Len(t, store.RemoveCalls(), 1)
	})

	t.Run("returns ErrNotFound for missing instance", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		err := mgr.Remove(ctx, "nonexistent")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestManager_Recreate(t *testing.T) {
	ctx := context.Background()

	t.Run("recreates container with new image", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc123",
					RepoID:      testRepoID,
					Branch:      "main",
					Worktree:    "/data/git/myrepo/main",
					ContainerID: "old-container",
					CreatedAt:   time.Now(),
					Status:      catalog.StatusRunning,
				}, nil
			},
			UpdateFunc: func(ctx context.Context, entry *catalog.Entry) error {
				return nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			StopFunc: func(ctx context.Context, id string) error {
				return nil
			},
			RemoveFunc: func(ctx context.Context, id string) error {
				return nil
			},
			RunFunc: func(ctx context.Context, cfg *container.RunConfig) (*container.Container, error) {
				return &container.Container{
					ID:     "new-container",
					Name:   cfg.Name,
					Image:  cfg.Image,
					Status: container.StatusRunning,
				}, nil
			},
		}

		mgr := NewManager(store, runtime, nil, nil, nil, ManagerConfig{})

		inst, err := mgr.Recreate(ctx, "abc123", "newimage:v2")

		require.NoError(t, err)
		assert.Equal(t, "new-container", inst.ContainerID)
		assert.Equal(t, StatusRunning, inst.Status)

		// Verify old container was removed
		require.Len(t, runtime.StopCalls(), 1)
		require.Len(t, runtime.RemoveCalls(), 1)

		// Verify new container was created with new image
		require.Len(t, runtime.RunCalls(), 1)
		assert.Equal(t, "newimage:v2", runtime.RunCalls()[0].Cfg.Image)
	})

	t.Run("returns ErrNotFound for missing instance", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		_, err := mgr.Recreate(ctx, "nonexistent", "image")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestManager_Attach(t *testing.T) {
	ctx := context.Background()

	t.Run("executes command in running container", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc123",
					ContainerID: "container-123",
				}, nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				return &container.Container{
					ID:     "container-123",
					Status: container.StatusRunning,
				}, nil
			},
			ExecFunc: func(ctx context.Context, id string, cfg container.ExecConfig) error {
				assert.Equal(t, "container-123", id)
				assert.Equal(t, []string{"bash", "-c", "echo hello"}, cfg.Command)
				return nil
			},
		}

		mgr := NewManager(store, runtime, nil, nil, nil, ManagerConfig{})

		err := mgr.Attach(ctx, "abc123", AttachConfig{
			Command: []string{"bash", "-c", "echo hello"},
		})

		require.NoError(t, err)
		require.Len(t, runtime.ExecCalls(), 1)
	})

	t.Run("uses default shell when no command specified", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc123",
					ContainerID: "container-123",
				}, nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				return &container.Container{
					ID:     "container-123",
					Status: container.StatusRunning,
				}, nil
			},
			ExecFunc: func(ctx context.Context, id string, cfg container.ExecConfig) error {
				assert.Equal(t, []string{"/bin/bash"}, cfg.Command)
				return nil
			},
		}

		mgr := NewManager(store, runtime, nil, nil, nil, ManagerConfig{})

		err := mgr.Attach(ctx, "abc123", AttachConfig{})

		require.NoError(t, err)
	})

	t.Run("returns error for stopped container", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc123",
					ContainerID: "container-123",
				}, nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				return &container.Container{
					ID:     "container-123",
					Status: container.StatusStopped,
				}, nil
			},
		}

		mgr := NewManager(store, runtime, nil, nil, nil, ManagerConfig{})

		err := mgr.Attach(ctx, "abc123", AttachConfig{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not running")
	})
}

func TestSanitizeBranch(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"main", "main"},
		{"feature/auth", "feature-auth"},
		{"feature/user-login", "feature-user-login"},
		{"bugfix/fix-123", "bugfix-fix-123"},
		{"release/v1.0.0", "release-v100"},
		{"test@branch", "testbranch"},
		{"-leading-dash", "leading-dash"},
		{"trailing-dash-", "trailing-dash"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeBranch(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestManager_CreateSession(t *testing.T) {
	ctx := context.Background()

	t.Run("creates session successfully", func(t *testing.T) {
		logsDir := t.TempDir()
		worktreeDir := t.TempDir()

		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc12345", // 8 chars, no hyphens
					ContainerID: "container-123",
					Worktree:    worktreeDir,
					Sessions:    []catalog.Session{},
				}, nil
			},
			UpdateFunc: func(ctx context.Context, entry *catalog.Entry) error {
				require.Len(t, entry.Sessions, 1)
				assert.Equal(t, catalog.SessionTypeShell, entry.Sessions[0].Type)
				assert.NotEmpty(t, entry.Sessions[0].ID)
				assert.NotEmpty(t, entry.Sessions[0].Name)
				return nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				return &container.Container{
					ID:     "container-123",
					Status: container.StatusRunning,
				}, nil
			},
			ExecCommandFunc: func() []string {
				return []string{"container", "exec"}
			},
		}
		mux := &muxmocks.MultiplexerMock{
			CreateSessionFunc: func(ctx context.Context, opts *multiplexer.CreateSessionOpts) (*multiplexer.Session, error) {
				assert.Contains(t, opts.Name, "hjk-abc12345-")
				assert.Equal(t, worktreeDir, opts.Cwd)
				assert.NotEmpty(t, opts.LogPath, "LogPath should be set for output capture")
				assert.Contains(t, opts.LogPath, logsDir)
				return &multiplexer.Session{Name: opts.Name}, nil
			},
		}

		mgr := NewManager(store, runtime, nil, mux, nil, ManagerConfig{LogsDir: logsDir})

		session, err := mgr.CreateSession(ctx, "abc12345", &CreateSessionConfig{})

		require.NoError(t, err)
		require.NotNil(t, session)
		assert.NotEmpty(t, session.ID)
		assert.NotEmpty(t, session.Name)
		assert.Equal(t, "shell", session.Type)
		require.Len(t, mux.CreateSessionCalls(), 1)
		require.Len(t, store.UpdateCalls(), 1)
	})

	t.Run("creates session with custom name", func(t *testing.T) {
		logsDir := t.TempDir()
		worktreeDir := t.TempDir()

		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc12345",
					ContainerID: "container-123",
					Worktree:    worktreeDir,
					Sessions:    []catalog.Session{},
				}, nil
			},
			UpdateFunc: func(ctx context.Context, entry *catalog.Entry) error {
				require.Len(t, entry.Sessions, 1)
				assert.Equal(t, "my-session", entry.Sessions[0].Name)
				return nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				return &container.Container{ID: "container-123", Status: container.StatusRunning}, nil
			},
			ExecCommandFunc: func() []string {
				return []string{"container", "exec"}
			},
		}
		mux := &muxmocks.MultiplexerMock{
			CreateSessionFunc: func(ctx context.Context, opts *multiplexer.CreateSessionOpts) (*multiplexer.Session, error) {
				return &multiplexer.Session{Name: opts.Name}, nil
			},
		}

		mgr := NewManager(store, runtime, nil, mux, nil, ManagerConfig{LogsDir: logsDir})

		session, err := mgr.CreateSession(ctx, "abc12345", &CreateSessionConfig{Name: "my-session"})

		require.NoError(t, err)
		assert.Equal(t, "my-session", session.Name)
	})

	t.Run("returns ErrSessionExists for duplicate name", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc12345",
					ContainerID: "container-123",
					Sessions: []catalog.Session{
						{ID: "sess1", Name: "existing-session"},
					},
				}, nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				return &container.Container{ID: "container-123", Status: container.StatusRunning}, nil
			},
		}

		mgr := NewManager(store, runtime, nil, nil, nil, ManagerConfig{})

		_, err := mgr.CreateSession(ctx, "abc12345", &CreateSessionConfig{Name: "existing-session"})

		assert.ErrorIs(t, err, ErrSessionExists)
	})

	t.Run("returns ErrInstanceNotRunning for stopped container", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:          "abc12345",
					ContainerID: "container-123",
					Sessions:    []catalog.Session{},
				}, nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			GetFunc: func(ctx context.Context, id string) (*container.Container, error) {
				return &container.Container{ID: "container-123", Status: container.StatusStopped}, nil
			},
		}

		mgr := NewManager(store, runtime, nil, nil, nil, ManagerConfig{})

		_, err := mgr.CreateSession(ctx, "abc12345", &CreateSessionConfig{})

		assert.ErrorIs(t, err, ErrInstanceNotRunning)
	})

	t.Run("returns ErrNotFound for missing instance", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		_, err := mgr.CreateSession(ctx, "nonexistent", &CreateSessionConfig{})

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestManager_GetSession(t *testing.T) {
	ctx := context.Background()

	t.Run("returns session by name", func(t *testing.T) {
		now := time.Now()
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID: "abc12345",
					Sessions: []catalog.Session{
						{ID: "sess1", Name: "first-session", Type: catalog.SessionTypeShell, CreatedAt: now, LastAccessed: now},
						{ID: "sess2", Name: "second-session", Type: catalog.SessionTypeClaude, CreatedAt: now, LastAccessed: now},
					},
				}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		session, err := mgr.GetSession(ctx, "abc12345", "second-session")

		require.NoError(t, err)
		assert.Equal(t, "sess2", session.ID)
		assert.Equal(t, "second-session", session.Name)
		assert.Equal(t, "claude", session.Type)
	})

	t.Run("returns ErrSessionNotFound for missing session", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:       "abc12345",
					Sessions: []catalog.Session{},
				}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		_, err := mgr.GetSession(ctx, "abc12345", "nonexistent")

		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("returns ErrNotFound for missing instance", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		_, err := mgr.GetSession(ctx, "nonexistent", "any")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestManager_ListSessions(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all sessions for instance", func(t *testing.T) {
		now := time.Now()
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID: "abc12345",
					Sessions: []catalog.Session{
						{ID: "sess1", Name: "first", Type: catalog.SessionTypeShell, CreatedAt: now, LastAccessed: now},
						{ID: "sess2", Name: "second", Type: catalog.SessionTypeClaude, CreatedAt: now, LastAccessed: now},
					},
				}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		sessions, err := mgr.ListSessions(ctx, "abc12345")

		require.NoError(t, err)
		require.Len(t, sessions, 2)
		assert.Equal(t, "first", sessions[0].Name)
		assert.Equal(t, "second", sessions[1].Name)
	})

	t.Run("returns empty slice for instance with no sessions", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:       "abc12345",
					Sessions: []catalog.Session{},
				}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		sessions, err := mgr.ListSessions(ctx, "abc12345")

		require.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("returns ErrNotFound for missing instance", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		_, err := mgr.ListSessions(ctx, "nonexistent")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestManager_KillSession(t *testing.T) {
	ctx := context.Background()

	t.Run("kills session and removes from catalog", func(t *testing.T) {
		logsDir := t.TempDir()
		// Create a mock log file
		sessLogPath := logsDir + "/abc12345/sess1.log"
		require.NoError(t, os.MkdirAll(logsDir+"/abc12345", 0o750))
		require.NoError(t, os.WriteFile(sessLogPath, []byte("log content"), 0o600))

		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID: "abc12345",
					Sessions: []catalog.Session{
						{ID: "sess1", Name: "my-session", MuxSessionID: "hjk-abc12345-sess1"},
					},
				}, nil
			},
			UpdateFunc: func(ctx context.Context, entry *catalog.Entry) error {
				assert.Empty(t, entry.Sessions)
				return nil
			},
		}
		mux := &muxmocks.MultiplexerMock{
			KillSessionFunc: func(ctx context.Context, sessionName string) error {
				assert.Equal(t, "hjk-abc12345-sess1", sessionName)
				return nil
			},
		}

		mgr := NewManager(store, nil, nil, mux, nil, ManagerConfig{LogsDir: logsDir})

		err := mgr.KillSession(ctx, "abc12345", "my-session")

		require.NoError(t, err)
		require.Len(t, mux.KillSessionCalls(), 1)
		require.Len(t, store.UpdateCalls(), 1)
		// Verify log file was removed
		_, statErr := os.Stat(sessLogPath)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("succeeds even if multiplexer session already dead", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID: "abc12345",
					Sessions: []catalog.Session{
						{ID: "sess1", Name: "my-session", MuxSessionID: "hjk-abc12345-sess1"},
					},
				}, nil
			},
			UpdateFunc: func(ctx context.Context, entry *catalog.Entry) error {
				return nil
			},
		}
		mux := &muxmocks.MultiplexerMock{
			KillSessionFunc: func(ctx context.Context, sessionName string) error {
				return multiplexer.ErrSessionNotFound
			},
		}

		mgr := NewManager(store, nil, nil, mux, nil, ManagerConfig{})

		err := mgr.KillSession(ctx, "abc12345", "my-session")

		require.NoError(t, err)
	})

	t.Run("returns ErrSessionNotFound for missing session", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:       "abc12345",
					Sessions: []catalog.Session{},
				}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		err := mgr.KillSession(ctx, "abc12345", "nonexistent")

		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("returns ErrNotFound for missing instance", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		err := mgr.KillSession(ctx, "nonexistent", "any")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestManager_AttachSession(t *testing.T) {
	ctx := context.Background()

	t.Run("attaches to session and updates last accessed", func(t *testing.T) {
		oldTime := time.Now().Add(-1 * time.Hour)
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID: "abc12345",
					Sessions: []catalog.Session{
						{ID: "sess1", Name: "my-session", MuxSessionID: "hjk-abc12345-sess1", LastAccessed: oldTime},
					},
				}, nil
			},
			UpdateFunc: func(ctx context.Context, entry *catalog.Entry) error {
				require.Len(t, entry.Sessions, 1)
				// LastAccessed should be updated to a recent time
				assert.True(t, entry.Sessions[0].LastAccessed.After(oldTime))
				return nil
			},
		}
		mux := &muxmocks.MultiplexerMock{
			AttachSessionFunc: func(ctx context.Context, sessionName string) error {
				assert.Equal(t, "hjk-abc12345-sess1", sessionName)
				return nil
			},
			ListSessionsFunc: func(ctx context.Context) ([]multiplexer.Session, error) {
				// Session still exists (user detached, not exited)
				return []multiplexer.Session{{Name: "hjk-abc12345-sess1"}}, nil
			},
		}

		mgr := NewManager(store, nil, nil, mux, nil, ManagerConfig{})

		err := mgr.AttachSession(ctx, "abc12345", "my-session")

		require.NoError(t, err)
		require.Len(t, mux.AttachSessionCalls(), 1)
		require.Len(t, store.UpdateCalls(), 1)
	})

	t.Run("returns ErrSessionNotFound for missing session", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:       "abc12345",
					Sessions: []catalog.Session{},
				}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		err := mgr.AttachSession(ctx, "abc12345", "nonexistent")

		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("cleans up session from catalog when user exits", func(t *testing.T) {
		oldTime := time.Now().Add(-1 * time.Hour)
		getCalls := 0
		updateCalls := 0
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				getCalls++
				return &catalog.Entry{
					ID: "abc12345",
					Sessions: []catalog.Session{
						{ID: "sess1", Name: "my-session", MuxSessionID: "hjk-abc12345-sess1", LastAccessed: oldTime},
					},
				}, nil
			},
			UpdateFunc: func(ctx context.Context, entry *catalog.Entry) error {
				updateCalls++
				if updateCalls == 2 {
					// Second update should have removed the session
					assert.Empty(t, entry.Sessions, "session should be removed from catalog")
				}
				return nil
			},
		}
		mux := &muxmocks.MultiplexerMock{
			AttachSessionFunc: func(ctx context.Context, sessionName string) error {
				return nil
			},
			ListSessionsFunc: func(ctx context.Context) ([]multiplexer.Session, error) {
				// Session no longer exists (user exited)
				return []multiplexer.Session{}, nil
			},
		}

		mgr := NewManager(store, nil, nil, mux, nil, ManagerConfig{})

		err := mgr.AttachSession(ctx, "abc12345", "my-session")

		require.NoError(t, err)
		assert.Equal(t, 2, getCalls, "should get entry twice (initial + cleanup)")
		assert.Equal(t, 2, updateCalls, "should update twice (timestamp + cleanup)")
	})
}

func TestManager_GetMRUSession(t *testing.T) {
	ctx := context.Background()

	t.Run("returns most recently accessed session", func(t *testing.T) {
		oldTime := time.Now().Add(-2 * time.Hour)
		recentTime := time.Now().Add(-1 * time.Minute)

		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID: "abc12345",
					Sessions: []catalog.Session{
						{ID: "sess1", Name: "old-session", LastAccessed: oldTime},
						{ID: "sess2", Name: "recent-session", LastAccessed: recentTime},
						{ID: "sess3", Name: "older-session", LastAccessed: oldTime.Add(-time.Hour)},
					},
				}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		session, err := mgr.GetMRUSession(ctx, "abc12345")

		require.NoError(t, err)
		assert.Equal(t, "recent-session", session.Name)
	})

	t.Run("returns ErrNoSessionsAvailable for instance with no sessions", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return &catalog.Entry{
					ID:       "abc12345",
					Sessions: []catalog.Session{},
				}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		_, err := mgr.GetMRUSession(ctx, "abc12345")

		assert.ErrorIs(t, err, ErrNoSessionsAvailable)
	})

	t.Run("returns ErrNotFound for missing instance", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			GetFunc: func(ctx context.Context, id string) (*catalog.Entry, error) {
				return nil, catalog.ErrNotFound
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		_, err := mgr.GetMRUSession(ctx, "nonexistent")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestGetImageRuntimeConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("returns defaults when registry is nil", func(t *testing.T) {
		mgr := NewManager(nil, nil, nil, nil, nil, ManagerConfig{
			RuntimeType: RuntimePodman,
		})

		cfg := mgr.getImageRuntimeConfig(ctx, "myimage:latest")

		assert.Empty(t, cfg.Init)
		assert.Nil(t, cfg.Flags)
	})

	t.Run("returns defaults when registry returns error", func(t *testing.T) {
		reg := &registrymocks.ClientMock{
			GetMetadataFunc: func(ctx context.Context, ref string) (*registry.ImageMetadata, error) {
				return nil, errors.New("registry unavailable")
			},
		}

		mgr := NewManager(nil, nil, nil, nil, reg, ManagerConfig{
			RuntimeType: RuntimePodman,
		})

		cfg := mgr.getImageRuntimeConfig(ctx, "myimage:latest")

		assert.Empty(t, cfg.Init)
		assert.Nil(t, cfg.Flags)
		require.Len(t, reg.GetMetadataCalls(), 1)
	})

	t.Run("extracts init label", func(t *testing.T) {
		reg := &registrymocks.ClientMock{
			GetMetadataFunc: func(ctx context.Context, ref string) (*registry.ImageMetadata, error) {
				return &registry.ImageMetadata{
					Labels: map[string]string{
						"io.headjack.init": "/lib/systemd/systemd",
					},
				}, nil
			},
		}

		mgr := NewManager(nil, nil, nil, nil, reg, ManagerConfig{
			RuntimeType: RuntimePodman,
		})

		cfg := mgr.getImageRuntimeConfig(ctx, "myimage:systemd")

		assert.Equal(t, "/lib/systemd/systemd", cfg.Init)
	})

	t.Run("extracts podman flags when using podman runtime", func(t *testing.T) {
		reg := &registrymocks.ClientMock{
			GetMetadataFunc: func(ctx context.Context, ref string) (*registry.ImageMetadata, error) {
				return &registry.ImageMetadata{
					Labels: map[string]string{
						"io.headjack.init":         "/lib/systemd/systemd",
						"io.headjack.podman.flags": "systemd=always privileged=true",
					},
				}, nil
			},
		}

		mgr := NewManager(nil, nil, nil, nil, reg, ManagerConfig{
			RuntimeType: RuntimePodman,
		})

		cfg := mgr.getImageRuntimeConfig(ctx, "myimage:systemd")

		assert.Equal(t, "/lib/systemd/systemd", cfg.Init)
		assert.Equal(t, "always", cfg.Flags["systemd"])
		assert.Equal(t, true, cfg.Flags["privileged"])
	})

	t.Run("ignores podman flags when using apple runtime", func(t *testing.T) {
		reg := &registrymocks.ClientMock{
			GetMetadataFunc: func(ctx context.Context, ref string) (*registry.ImageMetadata, error) {
				return &registry.ImageMetadata{
					Labels: map[string]string{
						"io.headjack.init":         "/lib/systemd/systemd",
						"io.headjack.podman.flags": "systemd=always privileged=true",
					},
				}, nil
			},
		}

		mgr := NewManager(nil, nil, nil, nil, reg, ManagerConfig{
			RuntimeType: RuntimeApple,
		})

		cfg := mgr.getImageRuntimeConfig(ctx, "myimage:systemd")

		assert.Equal(t, "/lib/systemd/systemd", cfg.Init)
		assert.Nil(t, cfg.Flags, "podman flags should be ignored for apple runtime")
	})

	t.Run("extracts apple flags when using apple runtime", func(t *testing.T) {
		reg := &registrymocks.ClientMock{
			GetMetadataFunc: func(ctx context.Context, ref string) (*registry.ImageMetadata, error) {
				return &registry.ImageMetadata{
					Labels: map[string]string{
						"io.headjack.init":        "/lib/systemd/systemd",
						"io.headjack.apple.flags": "network=bridge memory=2g",
					},
				}, nil
			},
		}

		mgr := NewManager(nil, nil, nil, nil, reg, ManagerConfig{
			RuntimeType: RuntimeApple,
		})

		cfg := mgr.getImageRuntimeConfig(ctx, "myimage:systemd")

		assert.Equal(t, "/lib/systemd/systemd", cfg.Init)
		assert.Equal(t, "bridge", cfg.Flags["network"])
		assert.Equal(t, "2g", cfg.Flags["memory"])
	})

	t.Run("ignores apple flags when using podman runtime", func(t *testing.T) {
		reg := &registrymocks.ClientMock{
			GetMetadataFunc: func(ctx context.Context, ref string) (*registry.ImageMetadata, error) {
				return &registry.ImageMetadata{
					Labels: map[string]string{
						"io.headjack.init":        "/lib/systemd/systemd",
						"io.headjack.apple.flags": "network=bridge memory=2g",
					},
				}, nil
			},
		}

		mgr := NewManager(nil, nil, nil, nil, reg, ManagerConfig{
			RuntimeType: RuntimePodman,
		})

		cfg := mgr.getImageRuntimeConfig(ctx, "myimage:systemd")

		assert.Equal(t, "/lib/systemd/systemd", cfg.Init)
		assert.Nil(t, cfg.Flags, "apple flags should be ignored for podman runtime")
	})

	t.Run("returns empty config when labels are nil", func(t *testing.T) {
		reg := &registrymocks.ClientMock{
			GetMetadataFunc: func(ctx context.Context, ref string) (*registry.ImageMetadata, error) {
				return &registry.ImageMetadata{
					Labels: nil,
				}, nil
			},
		}

		mgr := NewManager(nil, nil, nil, nil, reg, ManagerConfig{
			RuntimeType: RuntimePodman,
		})

		cfg := mgr.getImageRuntimeConfig(ctx, "myimage:latest")

		assert.Empty(t, cfg.Init)
		assert.Nil(t, cfg.Flags)
	})
}

func TestManager_GetGlobalMRUSession(t *testing.T) {
	ctx := context.Background()

	t.Run("returns most recently accessed session across all instances", func(t *testing.T) {
		oldTime := time.Now().Add(-2 * time.Hour)
		recentTime := time.Now().Add(-1 * time.Minute)

		store := &catalogmocks.StoreMock{
			ListFunc: func(ctx context.Context, filter catalog.ListFilter) ([]catalog.Entry, error) {
				return []catalog.Entry{
					{
						ID: "inst1",
						Sessions: []catalog.Session{
							{ID: "sess1", Name: "old-session", LastAccessed: oldTime},
						},
					},
					{
						ID: "inst2",
						Sessions: []catalog.Session{
							{ID: "sess2", Name: "recent-session", LastAccessed: recentTime},
						},
					},
					{
						ID: "inst3",
						Sessions: []catalog.Session{
							{ID: "sess3", Name: "older-session", LastAccessed: oldTime.Add(-time.Hour)},
						},
					},
				}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		result, err := mgr.GetGlobalMRUSession(ctx)

		require.NoError(t, err)
		assert.Equal(t, "inst2", result.InstanceID)
		assert.Equal(t, "recent-session", result.Session.Name)
	})

	t.Run("returns ErrNoSessionsAvailable when no sessions exist", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			ListFunc: func(ctx context.Context, filter catalog.ListFilter) ([]catalog.Entry, error) {
				return []catalog.Entry{
					{ID: "inst1", Sessions: []catalog.Session{}},
					{ID: "inst2", Sessions: []catalog.Session{}},
				}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		_, err := mgr.GetGlobalMRUSession(ctx)

		assert.ErrorIs(t, err, ErrNoSessionsAvailable)
	})

	t.Run("returns ErrNoSessionsAvailable when no instances exist", func(t *testing.T) {
		store := &catalogmocks.StoreMock{
			ListFunc: func(ctx context.Context, filter catalog.ListFilter) ([]catalog.Entry, error) {
				return []catalog.Entry{}, nil
			},
		}

		mgr := NewManager(store, nil, nil, nil, nil, ManagerConfig{})

		_, err := mgr.GetGlobalMRUSession(ctx)

		assert.ErrorIs(t, err, ErrNoSessionsAvailable)
	})
}
