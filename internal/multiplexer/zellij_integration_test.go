//go:build integration

// Package multiplexer integration tests for Zellij.
//
// These tests require:
// 1. Zellij to be installed
// 2. A real TTY (terminal) - they will NOT work in CI/non-interactive environments
//
// To run these tests manually:
//
//	go test -tags=integration -v ./internal/multiplexer/...
//
// Note: Zellij does not support detached session creation without a TTY,
// which is why these tests may fail in headless environments.
package multiplexer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"

	hjexec "github.com/jmgilman/headjack/internal/exec"
)

// skipIfNoZellij skips the test if Zellij is not installed.
func skipIfNoZellij(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("zellij"); err != nil {
		t.Skip("Zellij not installed, skipping integration test")
	}
}

// skipIfNoTTY skips the test if not running in a terminal.
// Zellij requires a TTY to create sessions.
func skipIfNoTTY(t *testing.T) {
	t.Helper()
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("Not running in a terminal, Zellij requires a TTY for session creation")
	}
}

// cleanupSession is a helper to ensure sessions are cleaned up after tests.
func cleanupSession(t *testing.T, z Multiplexer, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = z.KillSession(ctx, name)
}

func TestZellijIntegration_SessionLifecycle(t *testing.T) {
	skipIfNoZellij(t)
	skipIfNoTTY(t)

	executor := hjexec.New()
	z := NewZellij(executor)
	ctx := context.Background()

	sessionName := "hjk-integration-test-" + time.Now().Format("20060102150405")
	defer cleanupSession(t, z, sessionName)

	t.Run("create and list session", func(t *testing.T) {
		session, err := z.CreateSession(ctx, &CreateSessionOpts{
			Name: sessionName,
		})

		require.NoError(t, err)
		assert.Equal(t, sessionName, session.Name)
		assert.Equal(t, sessionName, session.ID)
		assert.False(t, session.CreatedAt.IsZero())

		// Verify session appears in list
		sessions, err := z.ListSessions(ctx)
		require.NoError(t, err)

		var found bool
		for _, s := range sessions {
			if s.Name == sessionName {
				found = true
				break
			}
		}
		assert.True(t, found, "Created session should appear in list")
	})

	t.Run("returns ErrSessionExists for duplicate", func(t *testing.T) {
		_, err := z.CreateSession(ctx, &CreateSessionOpts{
			Name: sessionName,
		})

		require.ErrorIs(t, err, ErrSessionExists)
	})

	t.Run("kill session", func(t *testing.T) {
		err := z.KillSession(ctx, sessionName)
		require.NoError(t, err)

		// Small delay to allow session to terminate
		time.Sleep(200 * time.Millisecond)

		// Verify session no longer in list
		sessions, err := z.ListSessions(ctx)
		require.NoError(t, err)

		for _, s := range sessions {
			assert.NotEqual(t, sessionName, s.Name, "Killed session should not appear in list")
		}
	})

	t.Run("kill missing session returns ErrSessionNotFound", func(t *testing.T) {
		err := z.KillSession(ctx, "nonexistent-session-12345")
		require.ErrorIs(t, err, ErrSessionNotFound)
	})
}

func TestZellijIntegration_SessionWithCwd(t *testing.T) {
	skipIfNoZellij(t)
	skipIfNoTTY(t)

	executor := hjexec.New()
	z := NewZellij(executor)
	ctx := context.Background()

	sessionName := "hjk-cwd-test-" + time.Now().Format("20060102150405")
	defer cleanupSession(t, z, sessionName)

	tmpDir := t.TempDir()

	session, err := z.CreateSession(ctx, &CreateSessionOpts{
		Name: sessionName,
		Cwd:  tmpDir,
	})

	require.NoError(t, err)
	assert.Equal(t, sessionName, session.Name)

	// Verify session was created
	sessions, err := z.ListSessions(ctx)
	require.NoError(t, err)

	var found bool
	for _, s := range sessions {
		if s.Name == sessionName {
			found = true
			break
		}
	}
	assert.True(t, found, "Session with cwd should be created")
}

func TestZellijIntegration_SessionWithLogging(t *testing.T) {
	skipIfNoZellij(t)
	skipIfNoTTY(t)

	executor := hjexec.New()
	z := NewZellij(executor)
	ctx := context.Background()

	sessionName := "hjk-log-test-" + time.Now().Format("20060102150405")
	defer cleanupSession(t, z, sessionName)

	// Create a temp directory for logs
	logsDir := t.TempDir()
	logPath := filepath.Join(logsDir, "session.log")

	session, err := z.CreateSession(ctx, &CreateSessionOpts{
		Name:    sessionName,
		LogPath: logPath,
	})

	require.NoError(t, err)
	assert.Equal(t, sessionName, session.Name)

	// Give the session a moment to initialize
	time.Sleep(500 * time.Millisecond)

	// Check that the log file was created
	_, err = os.Stat(logPath)
	assert.NoError(t, err, "Log file should be created when LogPath is provided")

	// Verify session is running
	sessions, err := z.ListSessions(ctx)
	require.NoError(t, err)

	var found bool
	for _, s := range sessions {
		if s.Name == sessionName {
			found = true
			break
		}
	}
	assert.True(t, found, "Logged session should be running")
}

func TestZellijIntegration_SessionWithCommand(t *testing.T) {
	skipIfNoZellij(t)
	skipIfNoTTY(t)

	executor := hjexec.New()
	z := NewZellij(executor)
	ctx := context.Background()

	sessionName := "hjk-cmd-test-" + time.Now().Format("20060102150405")
	defer cleanupSession(t, z, sessionName)

	// Create a session with a custom command (an interactive shell)
	session, err := z.CreateSession(ctx, &CreateSessionOpts{
		Name:    sessionName,
		Command: []string{"/bin/sh"},
	})

	require.NoError(t, err)
	assert.Equal(t, sessionName, session.Name)

	// Give the session a moment to initialize
	time.Sleep(300 * time.Millisecond)

	// Verify session is running
	sessions, err := z.ListSessions(ctx)
	require.NoError(t, err)

	var found bool
	for _, s := range sessions {
		if s.Name == sessionName {
			found = true
			break
		}
	}
	assert.True(t, found, "Session with custom command should be running")
}
