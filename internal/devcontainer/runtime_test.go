package devcontainer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmgilman/headjack/internal/container"
	containermocks "github.com/jmgilman/headjack/internal/container/mocks"
	"github.com/jmgilman/headjack/internal/exec"
	execmocks "github.com/jmgilman/headjack/internal/exec/mocks"
)

func TestNewRuntime(t *testing.T) {
	mockRT := &containermocks.RuntimeMock{}
	mockExec := &execmocks.ExecutorMock{}

	runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")

	require.NotNil(t, runtime)
}

func TestRuntime_Run(t *testing.T) {
	ctx := context.Background()

	t.Run("creates container using devcontainer up", func(t *testing.T) {
		mockRT := &containermocks.RuntimeMock{}
		mockExec := &execmocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "/usr/bin/devcontainer", opts.Name)
				assert.Contains(t, opts.Args, "up")
				assert.Contains(t, opts.Args, "--workspace-folder")
				assert.Contains(t, opts.Args, "/path/to/workspace")
				assert.Contains(t, opts.Args, "--docker-path")
				assert.Contains(t, opts.Args, "docker")

				return &exec.Result{
					Stdout: []byte(`{"outcome":"success","containerId":"abc123","remoteUser":"vscode","remoteWorkspaceFolder":"/workspaces/project"}`),
				}, nil
			},
		}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		c, err := runtime.Run(ctx, &container.RunConfig{
			Name:            "test-container",
			WorkspaceFolder: "/path/to/workspace",
		})

		require.NoError(t, err)
		assert.Equal(t, "abc123", c.ID)
		assert.Equal(t, "test-container", c.Name)
		assert.Equal(t, container.StatusRunning, c.Status)
	})

	t.Run("returns error when WorkspaceFolder is empty", func(t *testing.T) {
		mockRT := &containermocks.RuntimeMock{}
		mockExec := &execmocks.ExecutorMock{}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		_, err := runtime.Run(ctx, &container.RunConfig{
			Name: "test-container",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "WorkspaceFolder is required")
	})

	t.Run("returns error when devcontainer up fails", func(t *testing.T) {
		mockRT := &containermocks.RuntimeMock{}
		mockExec := &execmocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stdout: []byte(`{"outcome":"error"}`),
				}, nil
			},
		}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		_, err := runtime.Run(ctx, &container.RunConfig{
			Name:            "test-container",
			WorkspaceFolder: "/path/to/workspace",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "devcontainer up failed")
	})
}

func TestRuntime_Exec(t *testing.T) {
	ctx := context.Background()

	t.Run("executes command using devcontainer exec", func(t *testing.T) {
		mockRT := &containermocks.RuntimeMock{
			GetFunc: func(_ context.Context, id string) (*container.Container, error) {
				return &container.Container{
					ID:     id,
					Status: container.StatusRunning,
				}, nil
			},
		}
		mockExec := &execmocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "/usr/bin/devcontainer", opts.Name)
				assert.Contains(t, opts.Args, "exec")
				assert.Contains(t, opts.Args, "--container-id")
				assert.Contains(t, opts.Args, "abc123")
				assert.Contains(t, opts.Args, "bash")

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		err := runtime.Exec(ctx, "abc123", container.ExecConfig{
			Command: []string{"bash"},
		})

		require.NoError(t, err)
	})

	t.Run("returns error when container not found", func(t *testing.T) {
		mockRT := &containermocks.RuntimeMock{
			GetFunc: func(_ context.Context, _ string) (*container.Container, error) {
				return nil, container.ErrNotFound
			},
		}
		mockExec := &execmocks.ExecutorMock{}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		err := runtime.Exec(ctx, "missing", container.ExecConfig{
			Command: []string{"bash"},
		})

		assert.ErrorIs(t, err, container.ErrNotFound)
	})

	t.Run("returns error when container not running", func(t *testing.T) {
		mockRT := &containermocks.RuntimeMock{
			GetFunc: func(_ context.Context, id string) (*container.Container, error) {
				return &container.Container{
					ID:     id,
					Status: container.StatusStopped,
				}, nil
			},
		}
		mockExec := &execmocks.ExecutorMock{}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		err := runtime.Exec(ctx, "abc123", container.ExecConfig{
			Command: []string{"bash"},
		})

		assert.ErrorIs(t, err, container.ErrNotRunning)
	})
}

func TestRuntime_DelegatedMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("Stop delegates to underlying runtime", func(t *testing.T) {
		stopCalled := false
		mockRT := &containermocks.RuntimeMock{
			StopFunc: func(_ context.Context, id string) error {
				stopCalled = true
				assert.Equal(t, "abc123", id)
				return nil
			},
		}
		mockExec := &execmocks.ExecutorMock{}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		err := runtime.Stop(ctx, "abc123")

		require.NoError(t, err)
		assert.True(t, stopCalled)
	})

	t.Run("Start delegates to underlying runtime", func(t *testing.T) {
		startCalled := false
		mockRT := &containermocks.RuntimeMock{
			StartFunc: func(_ context.Context, id string) error {
				startCalled = true
				assert.Equal(t, "abc123", id)
				return nil
			},
		}
		mockExec := &execmocks.ExecutorMock{}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		err := runtime.Start(ctx, "abc123")

		require.NoError(t, err)
		assert.True(t, startCalled)
	})

	t.Run("Remove delegates to underlying runtime", func(t *testing.T) {
		removeCalled := false
		mockRT := &containermocks.RuntimeMock{
			RemoveFunc: func(_ context.Context, id string) error {
				removeCalled = true
				assert.Equal(t, "abc123", id)
				return nil
			},
		}
		mockExec := &execmocks.ExecutorMock{}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		err := runtime.Remove(ctx, "abc123")

		require.NoError(t, err)
		assert.True(t, removeCalled)
	})

	t.Run("Get delegates to underlying runtime", func(t *testing.T) {
		getCalled := false
		mockRT := &containermocks.RuntimeMock{
			GetFunc: func(_ context.Context, id string) (*container.Container, error) {
				getCalled = true
				assert.Equal(t, "abc123", id)
				return &container.Container{ID: id}, nil
			},
		}
		mockExec := &execmocks.ExecutorMock{}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		c, err := runtime.Get(ctx, "abc123")

		require.NoError(t, err)
		assert.True(t, getCalled)
		assert.Equal(t, "abc123", c.ID)
	})

	t.Run("List delegates to underlying runtime", func(t *testing.T) {
		listCalled := false
		mockRT := &containermocks.RuntimeMock{
			ListFunc: func(_ context.Context, filter container.ListFilter) ([]container.Container, error) {
				listCalled = true
				assert.Equal(t, "test", filter.Name)
				return []container.Container{{ID: "abc123"}}, nil
			},
		}
		mockExec := &execmocks.ExecutorMock{}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		containers, err := runtime.List(ctx, container.ListFilter{Name: "test"})

		require.NoError(t, err)
		assert.True(t, listCalled)
		assert.Len(t, containers, 1)
	})

	t.Run("Build delegates to underlying runtime", func(t *testing.T) {
		buildCalled := false
		mockRT := &containermocks.RuntimeMock{
			BuildFunc: func(_ context.Context, cfg *container.BuildConfig) error {
				buildCalled = true
				assert.Equal(t, "test:latest", cfg.Tag)
				return nil
			},
		}
		mockExec := &execmocks.ExecutorMock{}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		err := runtime.Build(ctx, &container.BuildConfig{Tag: "test:latest"})

		require.NoError(t, err)
		assert.True(t, buildCalled)
	})

	t.Run("ExecCommand delegates to underlying runtime", func(t *testing.T) {
		mockRT := &containermocks.RuntimeMock{
			ExecCommandFunc: func() []string {
				return []string{"docker", "exec"}
			},
		}
		mockExec := &execmocks.ExecutorMock{}

		runtime := NewRuntime(mockRT, mockExec, "/usr/bin/devcontainer", "docker")
		cmd := runtime.ExecCommand()

		assert.Equal(t, []string{"docker", "exec"}, cmd)
	})
}
