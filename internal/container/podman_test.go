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

func TestNewPodmanRuntime(t *testing.T) {
	mockExec := &mocks.ExecutorMock{}
	runtime := NewPodmanRuntime(mockExec, PodmanConfig{})

	require.NotNil(t, runtime)
}

func TestPodmanRuntime_Run(t *testing.T) {
	ctx := context.Background()

	t.Run("creates container successfully with default init command", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "podman", opts.Name)
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

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
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

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
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
				assert.Contains(t, opts.Args, "--systemd=always")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
			Flags: []string{"--systemd=always"},
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

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{Privileged: true})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
		})

		require.NoError(t, err)
	})

	t.Run("includes custom flags from config", func(t *testing.T) {
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

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{
			Flags: []string{"--memory=2g", "--cpus=2"},
		})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
		})

		require.NoError(t, err)
	})

	t.Run("includes volume mounts", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "-v")
				assert.Contains(t, opts.Args, "/host/path:/container/path")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
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
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "/host:/container:ro")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
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
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "-e")
				assert.Contains(t, opts.Args, "FOO=bar")

				return &exec.Result{
					Stdout:   []byte("abc123\n"),
					ExitCode: 0,
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "test",
			Image: "ubuntu",
			Env:   []string{"FOO=bar"},
		})

		require.NoError(t, err)
	})

	t.Run("returns ErrAlreadyExists when container exists", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("name already in use"),
					ExitCode: 125,
				}, errors.New("exit code 125")
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		_, err := runtime.Run(ctx, &RunConfig{
			Name:  "existing",
			Image: "ubuntu",
		})

		assert.ErrorIs(t, err, ErrAlreadyExists)
	})
}

func TestPodmanRuntime_Exec(t *testing.T) {
	ctx := context.Background()

	t.Run("executes command in running container", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				callCount++
				if callCount == 1 {
					// Get call - Podman format
					return &exec.Result{
						Stdout: []byte(`[{"Id":"abc123","Name":"test","State":{"Status":"running"},"Config":{"Image":"ubuntu"}}]`),
					}, nil
				}
				// Exec call
				assert.Equal(t, "podman", opts.Name)
				assert.Contains(t, opts.Args, "exec")
				assert.Contains(t, opts.Args, "abc123")
				assert.Contains(t, opts.Args, "bash")

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Exec(ctx, "abc123", ExecConfig{
			Command: []string{"bash"},
		})

		require.NoError(t, err)
	})

	t.Run("includes workdir when specified", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				callCount++
				if callCount == 1 {
					// Get call - Podman format
					return &exec.Result{
						Stdout: []byte(`[{"Id":"abc123","Name":"test","State":{"Status":"running"},"Config":{"Image":"ubuntu"}}]`),
					}, nil
				}
				assert.Contains(t, opts.Args, "-w")
				assert.Contains(t, opts.Args, "/app")

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Exec(ctx, "abc123", ExecConfig{
			Command: []string{"ls"},
			Workdir: "/app",
		})

		require.NoError(t, err)
	})

	t.Run("returns ErrNotFound when container missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("no such container"),
					ExitCode: 125,
				}, errors.New("exit code 125")
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Exec(ctx, "missing", ExecConfig{
			Command: []string{"bash"},
		})

		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns ErrNotRunning when container stopped", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				// Get call - Podman format with exited status
				return &exec.Result{
					Stdout: []byte(`[{"Id":"abc123","Name":"test","State":{"Status":"exited"},"Config":{"Image":"ubuntu"}}]`),
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Exec(ctx, "abc123", ExecConfig{
			Command: []string{"bash"},
		})

		assert.ErrorIs(t, err, ErrNotRunning)
	})
}

func TestPodmanRuntime_Stop(t *testing.T) {
	ctx := context.Background()

	t.Run("stops running container", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				callCount++
				if callCount == 1 {
					// Get call - Podman format
					return &exec.Result{
						Stdout: []byte(`[{"Id":"abc123","Name":"test","State":{"Status":"running"},"Config":{"Image":"ubuntu"}}]`),
					}, nil
				}
				// Stop call
				assert.Equal(t, "podman", opts.Name)
				assert.Equal(t, []string{"stop", "abc123"}, opts.Args)

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Stop(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, 2, callCount)
	})

	t.Run("no-op for already stopped container", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				callCount++
				// Get call - Podman format with exited status
				return &exec.Result{
					Stdout: []byte(`[{"Id":"abc123","Name":"test","State":{"Status":"exited"},"Config":{"Image":"ubuntu"}}]`),
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Stop(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, 1, callCount) // Only Get call, no Stop call
	})

	t.Run("returns ErrNotFound when container missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("no such container"),
					ExitCode: 125,
				}, errors.New("exit code 125")
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Stop(ctx, "missing")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestPodmanRuntime_Start(t *testing.T) {
	ctx := context.Background()

	t.Run("starts stopped container", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				callCount++
				if callCount == 1 {
					// Get call - Podman format with exited status
					return &exec.Result{
						Stdout: []byte(`[{"Id":"abc123","Name":"test","State":{"Status":"exited"},"Config":{"Image":"ubuntu"}}]`),
					}, nil
				}
				// Start call
				assert.Equal(t, "podman", opts.Name)
				assert.Equal(t, []string{"start", "abc123"}, opts.Args)

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Start(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, 2, callCount)
	})

	t.Run("no-op for already running container", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				callCount++
				// Get call - Podman format
				return &exec.Result{
					Stdout: []byte(`[{"Id":"abc123","Name":"test","State":{"Status":"running"},"Config":{"Image":"ubuntu"}}]`),
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Start(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, 1, callCount) // Only Get call, no Start call
	})
}

func TestPodmanRuntime_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("removes container", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "podman", opts.Name)
				assert.Equal(t, []string{"rm", "abc123"}, opts.Args)

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Remove(ctx, "abc123")

		require.NoError(t, err)
	})

	t.Run("returns ErrNotFound when container missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("no such container"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Remove(ctx, "missing")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestPodmanRuntime_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("returns container info", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "podman", opts.Name)
				assert.Equal(t, []string{"inspect", "abc123"}, opts.Args)

				// Podman format with RFC3339Nano timestamp
				return &exec.Result{
					Stdout: []byte(`[{"Id":"abc123def456","Name":"/test-container","State":{"Status":"running"},"Config":{"Image":"ubuntu:24.04"},"ImageName":"ubuntu:24.04","Created":"2024-01-15T10:30:00.123456789Z"}]`),
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		container, err := runtime.Get(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, "abc123def456", container.ID)
		assert.Equal(t, "test-container", container.Name) // Leading "/" is stripped
		assert.Equal(t, "ubuntu:24.04", container.Image)
		assert.Equal(t, StatusRunning, container.Status)
	})

	t.Run("parses stopped state", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				// Podman format with exited status
				return &exec.Result{
					Stdout: []byte(`[{"Id":"abc123","Name":"test","State":{"Status":"exited"},"Config":{"Image":"ubuntu"}}]`),
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		container, err := runtime.Get(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, StatusStopped, container.Status)
	})

	t.Run("parses RFC3339 timestamp fallback", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				// Podman format with RFC3339 timestamp (no nanoseconds)
				return &exec.Result{
					Stdout: []byte(`[{"Id":"abc123","Name":"test","State":{"Status":"running"},"Config":{"Image":"ubuntu"},"Created":"2024-01-15T10:30:00Z"}]`),
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		container, err := runtime.Get(ctx, "abc123")

		require.NoError(t, err)
		assert.False(t, container.CreatedAt.IsZero())
	})

	t.Run("returns ErrNotFound when container missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("no such container"),
					ExitCode: 125,
				}, errors.New("exit code 125")
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		_, err := runtime.Get(ctx, "missing")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestPodmanRuntime_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty list", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stdout: []byte("[]"),
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		containers, err := runtime.List(ctx, ListFilter{})

		require.NoError(t, err)
		assert.Empty(t, containers)
	})

	t.Run("returns container list", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				// Podman ps --format json output
				return &exec.Result{
					Stdout: []byte(`[{"Id":"abc","Names":["container1"],"Image":"ubuntu","State":"running","Created":1705320600},{"Id":"def","Names":["container2"],"Image":"alpine","State":"exited","Created":1705320500}]`),
				}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		containers, err := runtime.List(ctx, ListFilter{})

		require.NoError(t, err)
		require.Len(t, containers, 2)
		assert.Equal(t, "abc", containers[0].ID)
		assert.Equal(t, "container1", containers[0].Name)
		assert.Equal(t, StatusRunning, containers[0].Status)
		assert.Equal(t, "def", containers[1].ID)
		assert.Equal(t, "container2", containers[1].Name)
		assert.Equal(t, StatusStopped, containers[1].Status)
	})

	t.Run("includes name filter", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "--filter")
				assert.Contains(t, opts.Args, "name=my-prefix")

				return &exec.Result{Stdout: []byte("[]")}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		_, err := runtime.List(ctx, ListFilter{Name: "my-prefix"})

		require.NoError(t, err)
	})

	t.Run("uses ps -a for listing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "ps")
				assert.Contains(t, opts.Args, "-a")
				assert.Contains(t, opts.Args, "--format")
				assert.Contains(t, opts.Args, "json")

				return &exec.Result{Stdout: []byte("[]")}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		_, err := runtime.List(ctx, ListFilter{})

		require.NoError(t, err)
	})
}

func TestPodmanRuntime_Build(t *testing.T) {
	ctx := context.Background()

	t.Run("builds image", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "podman", opts.Name)
				assert.Contains(t, opts.Args, "build")
				assert.Contains(t, opts.Args, "-t")
				assert.Contains(t, opts.Args, "myimage:latest")
				assert.Contains(t, opts.Args, "/build/context")

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Build(ctx, &BuildConfig{
			Context: "/build/context",
			Tag:     "myimage:latest",
		})

		require.NoError(t, err)
	})

	t.Run("includes dockerfile path", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Contains(t, opts.Args, "-f")
				assert.Contains(t, opts.Args, "custom.Dockerfile")

				return &exec.Result{ExitCode: 0}, nil
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Build(ctx, &BuildConfig{
			Context:    "/build/context",
			Dockerfile: "custom.Dockerfile",
			Tag:        "myimage:latest",
		})

		require.NoError(t, err)
	})

	t.Run("returns ErrBuildFailed on failure", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(_ context.Context, _ *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("build error: missing base image"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		runtime := NewPodmanRuntime(mockExec, PodmanConfig{})
		err := runtime.Build(ctx, &BuildConfig{
			Context: "/build/context",
			Tag:     "myimage:latest",
		})

		require.ErrorIs(t, err, ErrBuildFailed)
		assert.Contains(t, err.Error(), "missing base image")
	})
}
