package instance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jmgilman/headjack/internal/catalog"
	catalogmocks "github.com/jmgilman/headjack/internal/catalog/mocks"
	"github.com/jmgilman/headjack/internal/container"
	containermocks "github.com/jmgilman/headjack/internal/container/mocks"
	"github.com/jmgilman/headjack/internal/git"
	gitmocks "github.com/jmgilman/headjack/internal/git/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager(nil, nil, nil, ManagerConfig{WorktreesDir: "/data/worktrees"})

	require.NotNil(t, mgr)
	assert.Equal(t, "/data/worktrees", mgr.worktreesDir)
}

func TestManager_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates instance successfully", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return "myrepo-abc123" },
			RootFunc:       func() string { return "/path/to/repo" },
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
			AddFunc: func(ctx context.Context, entry catalog.Entry) error {
				return nil
			},
			UpdateFunc: func(ctx context.Context, entry catalog.Entry) error {
				return nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			RunFunc: func(ctx context.Context, cfg container.RunConfig) (*container.Container, error) {
				return &container.Container{
					ID:     "container-123",
					Name:   cfg.Name,
					Image:  cfg.Image,
					Status: container.StatusRunning,
				}, nil
			},
		}

		mgr := NewManager(store, runtime, opener, ManagerConfig{WorktreesDir: "/data/worktrees"})

		inst, err := mgr.Create(ctx, "/path/to/repo", CreateConfig{
			Branch: "feature/auth",
			Image:  "myimage:latest",
		})

		require.NoError(t, err)
		require.NotNil(t, inst)
		assert.Equal(t, "myrepo-abc123", inst.RepoID)
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
			IdentifierFunc: func() string { return "myrepo-abc123" },
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

		mgr := NewManager(store, nil, opener, ManagerConfig{WorktreesDir: "/data/worktrees"})

		_, err := mgr.Create(ctx, "/path/to/repo", CreateConfig{Branch: "main"})

		assert.ErrorIs(t, err, ErrAlreadyExists)
	})

	t.Run("cleans up on worktree failure", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return "myrepo-abc123" },
			RootFunc:       func() string { return "/path/to/repo" },
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
			AddFunc: func(ctx context.Context, entry catalog.Entry) error {
				return nil
			},
			RemoveFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}

		mgr := NewManager(store, nil, opener, ManagerConfig{WorktreesDir: "/data/worktrees"})

		_, err := mgr.Create(ctx, "/path/to/repo", CreateConfig{Branch: "main"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "worktree error")
		// Verify cleanup was called
		require.Len(t, store.RemoveCalls(), 1)
	})

	t.Run("cleans up on container failure", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return "myrepo-abc123" },
			RootFunc:       func() string { return "/path/to/repo" },
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
			AddFunc: func(ctx context.Context, entry catalog.Entry) error {
				return nil
			},
			RemoveFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}
		runtime := &containermocks.RuntimeMock{
			RunFunc: func(ctx context.Context, cfg container.RunConfig) (*container.Container, error) {
				return nil, errors.New("container error")
			},
		}

		mgr := NewManager(store, runtime, opener, ManagerConfig{WorktreesDir: "/data/worktrees"})

		_, err := mgr.Create(ctx, "/path/to/repo", CreateConfig{Branch: "main"})

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
					Repo:        "/path/to/repo",
					RepoID:      "myrepo-abc123",
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

		mgr := NewManager(store, runtime, nil, ManagerConfig{})

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

		mgr := NewManager(store, nil, nil, ManagerConfig{})

		_, err := mgr.Get(ctx, "nonexistent")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestManager_GetByBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("returns instance by repo and branch", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return "myrepo-abc123" },
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

		mgr := NewManager(store, runtime, opener, ManagerConfig{})

		inst, err := mgr.GetByBranch(ctx, "/path/to/repo", "main")

		require.NoError(t, err)
		assert.Equal(t, "abc123", inst.ID)
		assert.Equal(t, "main", inst.Branch)
	})

	t.Run("returns ErrNotFound for missing branch", func(t *testing.T) {
		repo := &gitmocks.RepositoryMock{
			IdentifierFunc: func() string { return "myrepo-abc123" },
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

		mgr := NewManager(store, nil, opener, ManagerConfig{})

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

		mgr := NewManager(store, runtime, nil, ManagerConfig{})

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

		mgr := NewManager(store, runtime, nil, ManagerConfig{})

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
			UpdateFunc: func(ctx context.Context, entry catalog.Entry) error {
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

		mgr := NewManager(store, runtime, nil, ManagerConfig{})

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

		mgr := NewManager(store, nil, nil, ManagerConfig{})

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

		mgr := NewManager(store, runtime, opener, ManagerConfig{})

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

		mgr := NewManager(store, nil, nil, ManagerConfig{})

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
					RepoID:      "myrepo-abc123",
					Branch:      "main",
					Worktree:    "/data/git/myrepo/main",
					ContainerID: "old-container",
					CreatedAt:   time.Now(),
					Status:      catalog.StatusRunning,
				}, nil
			},
			UpdateFunc: func(ctx context.Context, entry catalog.Entry) error {
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
			RunFunc: func(ctx context.Context, cfg container.RunConfig) (*container.Container, error) {
				return &container.Container{
					ID:     "new-container",
					Name:   cfg.Name,
					Image:  cfg.Image,
					Status: container.StatusRunning,
				}, nil
			},
		}

		mgr := NewManager(store, runtime, nil, ManagerConfig{})

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

		mgr := NewManager(store, nil, nil, ManagerConfig{})

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

		mgr := NewManager(store, runtime, nil, ManagerConfig{})

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

		mgr := NewManager(store, runtime, nil, ManagerConfig{})

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

		mgr := NewManager(store, runtime, nil, ManagerConfig{})

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
