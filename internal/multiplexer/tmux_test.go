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

const tmuxCmdListSessions = "list-sessions"

func TestNewTmux(t *testing.T) {
	mockExec := &mocks.ExecutorMock{}
	tm := NewTmux(mockExec)

	require.NotNil(t, tm)
}

func TestTmux_CreateSession(t *testing.T) {
	ctx := context.Background()

	t.Run("creates session with name only", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				if opts.Args[0] == tmuxCmdListSessions {
					// No existing sessions
					return &exec.Result{
						Stderr:   []byte("no server running on /tmp/tmux"),
						ExitCode: 1,
					}, errors.New("exit code 1")
				}
				// Create session
				assert.Equal(t, "tmux", opts.Name)
				assert.Equal(t, []string{"new-session", "-d", "-s", "test-session"}, opts.Args)
				return &exec.Result{ExitCode: 0}, nil
			},
		}

		tm := NewTmux(mockExec)
		session, err := tm.CreateSession(ctx, &CreateSessionOpts{
			Name: "test-session",
		})

		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Equal(t, "test-session", session.Name)
		assert.Equal(t, "test-session", session.ID)
	})

	t.Run("creates session with all options", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				callCount++
				if callCount == 1 {
					// list-sessions call
					return &exec.Result{
						Stderr:   []byte("no server running"),
						ExitCode: 1,
					}, errors.New("exit code 1")
				}
				if callCount == 2 {
					// new-session call
					assert.Equal(t, "tmux", opts.Name)
					assert.Contains(t, opts.Args, "new-session")
					assert.Contains(t, opts.Args, "-d")
					assert.Contains(t, opts.Args, "-s")
					assert.Contains(t, opts.Args, "my-session")
					assert.Contains(t, opts.Args, "-c")
					assert.Contains(t, opts.Args, "/workspace")
					assert.Contains(t, opts.Args, "-e")
					assert.Contains(t, opts.Args, "FOO=bar")
					assert.Contains(t, opts.Args, "bash")
					return &exec.Result{ExitCode: 0}, nil
				}
				// pipe-pane call - verify shell-escaped path
				assert.Contains(t, opts.Args, "pipe-pane")
				assert.Equal(t, "cat >> '/var/log/session.log'", opts.Args[3])
				return &exec.Result{ExitCode: 0}, nil
			},
		}

		tm := NewTmux(mockExec)
		session, err := tm.CreateSession(ctx, &CreateSessionOpts{
			Name:    "my-session",
			Command: []string{"bash"},
			Cwd:     "/workspace",
			Env:     []string{"FOO=bar"},
			LogPath: "/var/log/session.log",
		})

		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Equal(t, "my-session", session.Name)
	})

	t.Run("escapes log path with special characters", func(t *testing.T) {
		callCount := 0
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				callCount++
				if callCount == 1 {
					return &exec.Result{
						Stderr:   []byte("no server running"),
						ExitCode: 1,
					}, errors.New("exit code 1")
				}
				if callCount == 2 {
					return &exec.Result{ExitCode: 0}, nil
				}
				// pipe-pane call - verify path with spaces and quotes is properly escaped
				assert.Equal(t, "cat >> '/var/log/my session'\\''s log.txt'", opts.Args[3])
				return &exec.Result{ExitCode: 0}, nil
			},
		}

		tm := NewTmux(mockExec)
		_, err := tm.CreateSession(ctx, &CreateSessionOpts{
			Name:    "test-session",
			LogPath: "/var/log/my session's log.txt",
		})

		require.NoError(t, err)
	})

	t.Run("returns ErrSessionExists when session exists", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				if opts.Args[0] == tmuxCmdListSessions {
					return &exec.Result{
						Stdout:   []byte("existing-session\n"),
						ExitCode: 0,
					}, nil
				}
				return &exec.Result{ExitCode: 0}, nil
			},
		}

		tm := NewTmux(mockExec)
		_, err := tm.CreateSession(ctx, &CreateSessionOpts{
			Name: "existing-session",
		})

		require.ErrorIs(t, err, ErrSessionExists)
	})

	t.Run("returns ErrCreateFailed when name is empty", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{}

		tm := NewTmux(mockExec)
		_, err := tm.CreateSession(ctx, &CreateSessionOpts{})

		require.ErrorIs(t, err, ErrCreateFailed)
	})

	t.Run("returns ErrCreateFailed when opts is nil", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{}

		tm := NewTmux(mockExec)
		_, err := tm.CreateSession(ctx, nil)

		require.ErrorIs(t, err, ErrCreateFailed)
	})

	t.Run("returns ErrSessionExists on duplicate session error", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				if opts.Args[0] == tmuxCmdListSessions {
					return &exec.Result{
						Stderr:   []byte("no server running"),
						ExitCode: 1,
					}, errors.New("exit code 1")
				}
				return &exec.Result{
					Stderr:   []byte("duplicate session: test-session"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		tm := NewTmux(mockExec)
		_, err := tm.CreateSession(ctx, &CreateSessionOpts{
			Name: "test-session",
		})

		require.ErrorIs(t, err, ErrSessionExists)
	})
}

func TestTmux_ListSessions(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty list when no sessions", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "tmux", opts.Name)
				assert.Equal(t, []string{"list-sessions", "-F", "#{session_name}"}, opts.Args)

				return &exec.Result{
					Stderr:   []byte("no server running on /tmp/tmux"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		tm := NewTmux(mockExec)
		sessions, err := tm.ListSessions(ctx)

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

		tm := NewTmux(mockExec)
		sessions, err := tm.ListSessions(ctx)

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

		tm := NewTmux(mockExec)
		sessions, err := tm.ListSessions(ctx)

		require.NoError(t, err)
		require.Len(t, sessions, 3)
		assert.Equal(t, "session-1", sessions[0].Name)
		assert.Equal(t, "session-2", sessions[1].Name)
		assert.Equal(t, "session-3", sessions[2].Name)
	})

	t.Run("handles no sessions message gracefully", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("no sessions"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		tm := NewTmux(mockExec)
		sessions, err := tm.ListSessions(ctx)

		require.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("handles error connecting to socket gracefully", func(t *testing.T) {
		// "error connecting to" means the tmux server socket doesn't exist or
		// can't be reached, which is functionally equivalent to "no sessions".
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("error connecting to /tmp/tmux-1000/default"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		tm := NewTmux(mockExec)
		sessions, err := tm.ListSessions(ctx)

		require.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("returns error on exit code 1 with unexpected stderr", func(t *testing.T) {
		// Exit code 1 should only be treated as "no sessions" if stderr contains
		// known messages like "no server running", "no sessions", or "error connecting to".
		// Other exit code 1 errors should be surfaced.
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("permission denied: /tmp/tmux-1000/default"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		tm := NewTmux(mockExec)
		_, err := tm.ListSessions(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "list sessions")
	})

	t.Run("returns error on unexpected command failure", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("unexpected error"),
					ExitCode: 2,
				}, errors.New("exit code 2")
			},
		}

		tm := NewTmux(mockExec)
		_, err := tm.ListSessions(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "list sessions")
	})
}

func TestTmux_KillSession(t *testing.T) {
	ctx := context.Background()

	t.Run("kills session successfully", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "tmux", opts.Name)
				assert.Equal(t, []string{"kill-session", "-t", "my-session"}, opts.Args)

				return &exec.Result{
					ExitCode: 0,
				}, nil
			},
		}

		tm := NewTmux(mockExec)
		err := tm.KillSession(ctx, "my-session")

		require.NoError(t, err)
	})

	t.Run("returns ErrSessionNotFound when session missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("can't find session: missing"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		tm := NewTmux(mockExec)
		err := tm.KillSession(ctx, "missing")

		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("returns ErrSessionNotFound for no session error", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				return &exec.Result{
					Stderr:   []byte("no session: nonexistent"),
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		tm := NewTmux(mockExec)
		err := tm.KillSession(ctx, "nonexistent")

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

		tm := NewTmux(mockExec)
		err := tm.KillSession(ctx, "my-session")

		require.Error(t, err)
		require.NotErrorIs(t, err, ErrSessionNotFound)
		assert.Contains(t, err.Error(), "kill session")
	})
}

func TestTmux_AttachSession(t *testing.T) {
	ctx := context.Background()

	t.Run("attaches to session with correct args", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				assert.Equal(t, "tmux", opts.Name)
				assert.Equal(t, []string{"attach-session", "-t", "my-session"}, opts.Args)

				return &exec.Result{
					ExitCode: 0,
				}, nil
			},
		}

		tm := NewTmux(mockExec)
		// Note: This test won't fully exercise TTY handling since we're not in a terminal
		err := tm.AttachSession(ctx, "my-session")

		require.NoError(t, err)
	})

	t.Run("returns ErrSessionNotFound when session missing", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				// Write to the stderr writer (io.MultiWriter) that AttachSession provides
				if opts.Stderr != nil {
					_, _ = opts.Stderr.Write([]byte("can't find session: missing"))
				}
				return &exec.Result{
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		tm := NewTmux(mockExec)
		err := tm.AttachSession(ctx, "missing")

		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("returns ErrSessionNotFound for no session error", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				if opts.Stderr != nil {
					_, _ = opts.Stderr.Write([]byte("no session: nonexistent"))
				}
				return &exec.Result{
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		tm := NewTmux(mockExec)
		err := tm.AttachSession(ctx, "nonexistent")

		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("returns ErrAttachFailed on command error", func(t *testing.T) {
		mockExec := &mocks.ExecutorMock{
			RunFunc: func(ctx context.Context, opts *exec.RunOptions) (*exec.Result, error) {
				if opts.Stderr != nil {
					_, _ = opts.Stderr.Write([]byte("attach failed"))
				}
				return &exec.Result{
					ExitCode: 1,
				}, errors.New("exit code 1")
			},
		}

		tm := NewTmux(mockExec)
		err := tm.AttachSession(ctx, "my-session")

		assert.ErrorIs(t, err, ErrAttachFailed)
	})
}
