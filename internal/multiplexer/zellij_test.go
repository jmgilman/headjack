package multiplexer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmgilman/headjack/internal/exec"
	"github.com/jmgilman/headjack/internal/exec/mocks"
)

func TestNewZellij(t *testing.T) {
	mockExec := &mocks.ExecutorMock{}
	z := NewZellij(mockExec)

	require.NotNil(t, z)
}

func TestZellij_CreateSession(t *testing.T) {
	ctx := context.Background()

	t.Run("returns ErrDetachedModeNotSupported", func(t *testing.T) {
		// Zellij does not support detached session creation
		mockExec := &mocks.ExecutorMock{}

		z := NewZellij(mockExec)
		_, err := z.CreateSession(ctx, &CreateSessionOpts{
			Name: "test-session",
		})

		require.ErrorIs(t, err, ErrDetachedModeNotSupported)
	})
}

func TestZellij_ListSessions(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty list when no sessions", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "zellij", opts.Name)
				assert.Equal(t, []string{"list-sessions"}, opts.Args)

				return &exec.Result{
					Stdout:   []byte(""),
					ExitCode: 0,
				}, nil
			},
		}

		z := NewZellij(mockExec)
		sessions, err := z.ListSessions(ctx)

		require.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("parses single session", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stdout:   []byte("my-session\n"),
					ExitCode: 0,
				}, nil
			},
		}

		z := NewZellij(mockExec)
		sessions, err := z.ListSessions(ctx)

		require.NoError(t, err)
		require.Len(t, sessions, 1)
		assert.Equal(t, "my-session", sessions[0].ID)
		assert.Equal(t, "my-session", sessions[0].Name)
	})

	t.Run("parses multiple sessions", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stdout:   []byte("session-1\nsession-2\nsession-3\n"),
					ExitCode: 0,
				}, nil
			},
		}

		z := NewZellij(mockExec)
		sessions, err := z.ListSessions(ctx)

		require.NoError(t, err)
		require.Len(t, sessions, 3)
		assert.Equal(t, "session-1", sessions[0].Name)
		assert.Equal(t, "session-2", sessions[1].Name)
		assert.Equal(t, "session-3", sessions[2].Name)
	})

	t.Run("parses session with metadata", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				// Zellij output format: "session-name [Created 2h ago] (current)"
				return &exec.Result{
					Stdout:   []byte("my-session [Created 2h ago] (current)\nother-session [Created 1d ago]\n"),
					ExitCode: 0,
				}, nil
			},
		}

		z := NewZellij(mockExec)
		sessions, err := z.ListSessions(ctx)

		require.NoError(t, err)
		require.Len(t, sessions, 2)
		assert.Equal(t, "my-session", sessions[0].Name)
		assert.Equal(t, "other-session", sessions[1].Name)
	})

	t.Run("handles no active sessions error gracefully", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("No active zellij sessions found"),
					ExitCode: 0, // zellij may return 0 with this message
				}, nil
			},
		}

		z := NewZellij(mockExec)
		sessions, err := z.ListSessions(ctx)

		require.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("returns error on command failure", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("zellij: command not found"),
					ExitCode: 127,
				}, errors.New("exit code 127")
			},
		}

		z := NewZellij(mockExec)
		_, err := z.ListSessions(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "list sessions")
	})
}

func TestZellij_KillSession(t *testing.T) {
	ctx := context.Background()

	t.Run("kills session successfully", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "zellij", opts.Name)
				assert.Equal(t, []string{"kill-session", "my-session"}, opts.Args)

				return &exec.Result{
					ExitCode: 0,
				}, nil
			},
		}

		z := NewZellij(mockExec)
		err := z.KillSession(ctx, "my-session")

		require.NoError(t, err)
	})

	t.Run("returns ErrSessionNotFound when session missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("Session 'missing' not found"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		z := NewZellij(mockExec)
		err := z.KillSession(ctx, "missing")

		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("returns ErrSessionNotFound for no session error", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("No session with that name"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		z := NewZellij(mockExec)
		err := z.KillSession(ctx, "missing")

		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("returns generic error for other failures", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("unexpected error"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		z := NewZellij(mockExec)
		err := z.KillSession(ctx, "my-session")

		require.Error(t, err)
		require.NotErrorIs(t, err, ErrSessionNotFound)
		assert.Contains(t, err.Error(), "kill session")
	})
}

func TestZellij_AttachSession(t *testing.T) {
	ctx := context.Background()

	t.Run("attaches to session with correct args", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "zellij", opts.Name)
				assert.Equal(t, []string{"attach", "my-session"}, opts.Args)

				return &exec.Result{
					ExitCode: 0,
				}, nil
			},
		}

		z := NewZellij(mockExec)
		// Note: This test won't fully exercise TTY handling since we're not in a terminal
		err := z.AttachSession(ctx, "my-session")

		require.NoError(t, err)
	})

	t.Run("returns ErrSessionNotFound when session missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				// Write to the stderr writer (io.MultiWriter) that AttachSession provides
				if opts.Stderr != nil {
					_, _ = opts.Stderr.Write([]byte("Session not found"))
				}
				return &exec.Result{
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		z := NewZellij(mockExec)
		err := z.AttachSession(ctx, "missing")

		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("returns ErrAttachFailed on command error", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				// Write to the stderr writer (io.MultiWriter) that AttachSession provides
				if opts.Stderr != nil {
					_, _ = opts.Stderr.Write([]byte("attach failed"))
				}
				return &exec.Result{
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		z := NewZellij(mockExec)
		err := z.AttachSession(ctx, "my-session")

		assert.ErrorIs(t, err, ErrAttachFailed)
	})
}
