package container

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmgilman/headjack/internal/exec"
	"github.com/jmgilman/headjack/internal/exec/mocks"
)

func TestNewAppleRuntime(t *testing.T) {
	mockExec := &mocks.ExecutorMock{}
	runtime := NewAppleRuntime(mockExec, AppleConfig{})

	require.NotNil(t, runtime)
}

func TestAppleRuntime_Run(t *testing.T) {
	ctx := context.Background()

	t.Run("creates container successfully with default init command", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "container", opts.Name)
				assert.Contains(t, opts.Args, "run")
				assert.Contains(t, opts.Args, "--detach")
				assert.Contains(t, opts.Args, "--name")
				assert.Contains(t, opts.Args, "test-container")
				assert.Contains(t, opts.Args, "ubuntu:24.04")
				// Default init command should be "sleep infinity"
				assert.Contains(t, opts.Args, "sleep")
				assert.Contains(t, opts.Args, "infinity")

				return &exec.Result{
					Stdout:   []byte("abc123def456\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		container, err := runtime.Run(ctx, &RunConfig{
			Name:  "test-container",
			Image: "ubuntu:24.04",
		})

		require.NoError(t, err)
		assert.Equal(t, "abc123def456", container.ID)
		assert.Equal(t, "test-container", container.Name)
		assert.Equal(t, "ubuntu:24.04", container.Image)
		assert.Equal(t, StatusRunning, container.Status)
	})

	t.Run("uses custom init command when specified", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				// Custom init command should be at the end
				assert.Contains(t, opts.Args, "/lib/systemd/systemd")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
			Init:  "/lib/systemd/systemd",
		})

		require.NoError(t, err)
	})

	t.Run("includes image-specific flags from RunConfig", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "--custom-flag")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
			Flags: []string{"--custom-flag"},
		})

		require.NoError(t, err)
	})

	t.Run("includes privileged flag when configured", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "--privileged")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
			Flags: []string{"--privileged"},
		})

		require.NoError(t, err)
	})

	t.Run("includes custom flags from RunConfig", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "--memory=2g")
				assert.Contains(t, opts.Args, "--cpus=2")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
			Flags: []string{"--memory=2g", "--cpus=2"},
		})

		require.NoError(t, err)
	})

	t.Run("includes volume mounts", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "-v")
				assert.Contains(t, opts.Args, "/host/path:/container/path")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
			Mounts: []Mount{
				{Source: "/host/path", Target: "/container/path"},
			},
		})

		require.NoError(t, err)
	})

	t.Run("includes read-only mount flag", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "/host:/container:ro")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
			Mounts: []Mount{
				{Source: "/host", Target: "/container", ReadOnly: true},
			},
		})

		require.NoError(t, err)
	})

	t.Run("includes environment variables", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "-e")
				assert.Contains(t, opts.Args, "FOO=bar")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
			Env:   []string{"FOO=bar"},
		})

		require.NoError(t, err)
	})

	t.Run("returns ErrAlreadyExists when container exists", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("container already exists"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "existing",
			Image: "ubuntu",
		})

		assert.ErrorIs(t, err, ErrAlreadyExists)
	})
}

func TestAppleRuntime_Exec(t *testing.T) {
	ctx := context.Background()

	t.Run("executes command in running container", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				callCount++
				if callCount == 1 {
					// Get call - Apple Container format
					return &exec.Result{
						Stdout: []byte(`[{"status":"running","configuration":{"id":"abc123","image":{"reference":"ubuntu"}}}]`),
					}, nil
				}
				// Exec call
				assert.Equal(t, "container", opts.Name)
				assert.Contains(t, opts.Args, "exec")
				assert.Contains(t, opts.Args, "abc123")
				assert.Contains(t, opts.Args, "bash")

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Exec(ctx, "abc123", ExecConfig{
			Command: []string{"bash"},
		})

		require.NoError(t, err)
	})

	t.Run("includes workdir when specified", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				callCount++
				if callCount == 1 {
					// Get call - Apple Container format
					return &exec.Result{
						Stdout: []byte(`[{"status":"running","configuration":{"id":"abc123","image":{"reference":"ubuntu"}}}]`),
					}, nil
				}
				assert.Contains(t, opts.Args, "-w")
				assert.Contains(t, opts.Args, "/app")

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Exec(ctx, "abc123", ExecConfig{
			Command: []string{"ls"},
			Workdir: "/app",
		})

		require.NoError(t, err)
	})

	t.Run("returns ErrNotFound when container missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("container not found"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Exec(ctx, "missing", ExecConfig{
			Command: []string{"bash"},
		})

		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns ErrNotRunning when container stopped", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				// Get call - Apple Container format with stopped status
				return &exec.Result{
					Stdout: []byte(`[{"status":"stopped","configuration":{"id":"abc123","image":{"reference":"ubuntu"}}}]`),
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Exec(ctx, "abc123", ExecConfig{
			Command: []string{"bash"},
		})

		assert.ErrorIs(t, err, ErrNotRunning)
	})
}

func TestAppleRuntime_Stop(t *testing.T) {
	ctx := context.Background()

	t.Run("stops running container", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				callCount++
				if callCount == 1 {
					// Get call - Apple Container format
					return &exec.Result{
						Stdout: []byte(`[{"status":"running","configuration":{"id":"abc123","image":{"reference":"ubuntu"}}}]`),
					}, nil
				}
				// Stop call
				assert.Equal(t, "container", opts.Name)
				assert.Equal(t, []string{"stop", "abc123"}, opts.Args)

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Stop(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, 2, callCount)
	})

	t.Run("no-op for already stopped container", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				callCount++
				// Get call - Apple Container format with stopped status
				return &exec.Result{
					Stdout: []byte(`[{"status":"stopped","configuration":{"id":"abc123","image":{"reference":"ubuntu"}}}]`),
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Stop(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, 1, callCount) // Only Get call, no Stop call
	})

	t.Run("returns ErrNotFound when container missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("container not found"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Stop(ctx, "missing")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestAppleRuntime_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("removes container", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "container", opts.Name)
				assert.Equal(t, []string{"rm", "abc123"}, opts.Args)

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Remove(ctx, "abc123")

		require.NoError(t, err)
	})

	t.Run("returns ErrNotFound when container missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("no such container"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Remove(ctx, "missing")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestAppleRuntime_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("returns container info", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "container", opts.Name)
				assert.Contains(t, opts.Args, "inspect")
				assert.Contains(t, opts.Args, "abc123")

				// Apple Container format
				return &exec.Result{
					Stdout: []byte(`[{"status":"running","configuration":{"id":"abc123def456","image":{"reference":"ubuntu:24.04"}}}]`),
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		container, err := runtime.Get(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, "abc123def456", container.ID)
		assert.Equal(t, "abc123def456", container.Name) // Name is set to ID in Apple Container format
		assert.Equal(t, "ubuntu:24.04", container.Image)
		assert.Equal(t, StatusRunning, container.Status)
	})

	t.Run("parses stopped state", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				// Apple Container format with exited status
				return &exec.Result{
					Stdout: []byte(`[{"status":"exited","configuration":{"id":"abc123","image":{"reference":"ubuntu"}}}]`),
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		container, err := runtime.Get(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, StatusStopped, container.Status)
	})

	t.Run("returns ErrNotFound when container missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("not found"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		_, err := runtime.Get(ctx, "missing")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestAppleRuntime_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty list", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stdout: []byte("[]"),
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		containers, err := runtime.List(ctx, ListFilter{})

		require.NoError(t, err)
		assert.Empty(t, containers)
	})

	t.Run("returns container list", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				// Apple Container format
				return &exec.Result{
					Stdout: []byte(`[{"status":"running","configuration":{"id":"abc","image":{"reference":"ubuntu"}}},{"status":"stopped","configuration":{"id":"def","image":{"reference":"alpine"}}}]`),
				}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		containers, err := runtime.List(ctx, ListFilter{})

		require.NoError(t, err)
		require.Len(t, containers, 2)
		assert.Equal(t, "abc", containers[0].ID)
		assert.Equal(t, "abc", containers[0].Name) // Name is set to ID in Apple Container format
		assert.Equal(t, StatusRunning, containers[0].Status)
		assert.Equal(t, "def", containers[1].ID)
		assert.Equal(t, StatusStopped, containers[1].Status)
	})

	t.Run("includes name filter", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "--filter")
				assert.Contains(t, opts.Args, "name=my-prefix")

				return &exec.Result{Stdout: []byte("[]")}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		_, err := runtime.List(ctx, ListFilter{Name: "my-prefix"})

		require.NoError(t, err)
	})
}

func TestAppleRuntime_Build(t *testing.T) {
	ctx := context.Background()

	t.Run("builds image", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "container", opts.Name)
				assert.Contains(t, opts.Args, "build")
				assert.Contains(t, opts.Args, "-t")
				assert.Contains(t, opts.Args, "myimage:latest")
				assert.Contains(t, opts.Args, "/build/context")

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Build(ctx, &BuildConfig{
			Context: "/build/context",
			Tag:     "myimage:latest",
		})

		require.NoError(t, err)
	})

	t.Run("includes dockerfile path", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "-f")
				assert.Contains(t, opts.Args, "custom.Dockerfile")

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Build(ctx, &BuildConfig{
			Context:    "/build/context",
			Dockerfile: "custom.Dockerfile",
			Tag:        "myimage:latest",
		})

		require.NoError(t, err)
	})

	t.Run("returns ErrBuildFailed on failure", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("build error: missing base image"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		runtime := NewAppleRuntime(mockExec, AppleConfig{})
		err := runtime.Build(ctx, &BuildConfig{
			Context: "/build/context",
			Tag:     "myimage:latest",
		})

		require.ErrorIs(t, err, ErrBuildFailed)
		assert.Contains(t, err.Error(), "missing base image")
	})
}
