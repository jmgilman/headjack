package logging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathManager_BaseDir(t *testing.T) {
	pm := NewPathManager("/var/log/headjack")
	assert.Equal(t, "/var/log/headjack", pm.BaseDir())
}

func TestPathManager_InstanceDir(t *testing.T) {
	pm := NewPathManager("/var/log/headjack")
	assert.Equal(t, "/var/log/headjack/abc123", pm.InstanceDir("abc123"))
}

func TestPathManager_SessionLogPath(t *testing.T) {
	pm := NewPathManager("/var/log/headjack")
	path := pm.SessionLogPath("abc123", "session456")
	assert.Equal(t, "/var/log/headjack/abc123/session456.log", path)
}

func TestPathManager_EnsureInstanceDir(t *testing.T) {
	baseDir := t.TempDir()
	pm := NewPathManager(baseDir)

	dir, err := pm.EnsureInstanceDir("test-instance")
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(baseDir, "test-instance"), dir)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestPathManager_EnsureSessionLog(t *testing.T) {
	baseDir := t.TempDir()
	pm := NewPathManager(baseDir)

	path, err := pm.EnsureSessionLog("inst1", "sess1")
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(baseDir, "inst1", "sess1.log"), path)

	// Verify directory was created
	info, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestPathManager_LogExists(t *testing.T) {
	baseDir := t.TempDir()
	pm := NewPathManager(baseDir)

	// Log doesn't exist yet
	assert.False(t, pm.LogExists("inst1", "sess1"))

	// Create the log file
	path, err := pm.EnsureSessionLog("inst1", "sess1")
	require.NoError(t, err)

	err = os.WriteFile(path, []byte("test"), 0644)
	require.NoError(t, err)

	// Now it should exist
	assert.True(t, pm.LogExists("inst1", "sess1"))
}

func TestPathManager_RemoveSessionLog(t *testing.T) {
	baseDir := t.TempDir()
	pm := NewPathManager(baseDir)

	// Create a log file
	path, err := pm.EnsureSessionLog("inst1", "sess1")
	require.NoError(t, err)

	err = os.WriteFile(path, []byte("test"), 0644)
	require.NoError(t, err)

	assert.True(t, pm.LogExists("inst1", "sess1"))

	// Remove it
	err = pm.RemoveSessionLog("inst1", "sess1")
	require.NoError(t, err)

	assert.False(t, pm.LogExists("inst1", "sess1"))

	// Removing non-existent should not error
	err = pm.RemoveSessionLog("inst1", "nonexistent")
	require.NoError(t, err)
}

func TestPathManager_RemoveInstanceLogs(t *testing.T) {
	baseDir := t.TempDir()
	pm := NewPathManager(baseDir)

	// Create multiple log files for an instance
	for _, sess := range []string{"sess1", "sess2", "sess3"} {
		path, err := pm.EnsureSessionLog("inst1", sess)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Verify they exist
	sessions, err := pm.ListSessionLogs("inst1")
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// Remove all
	err = pm.RemoveInstanceLogs("inst1")
	require.NoError(t, err)

	// Verify directory is gone
	_, err = os.Stat(pm.InstanceDir("inst1"))
	assert.True(t, os.IsNotExist(err))

	// Removing non-existent instance should not error
	err = pm.RemoveInstanceLogs("nonexistent")
	require.NoError(t, err)
}

func TestPathManager_ListSessionLogs(t *testing.T) {
	baseDir := t.TempDir()
	pm := NewPathManager(baseDir)

	// Empty directory should return nil
	sessions, err := pm.ListSessionLogs("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, sessions)

	// Create some log files
	for _, sess := range []string{"alpha", "beta", "gamma"} {
		path, err := pm.EnsureSessionLog("inst1", sess)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Also create a non-log file (should be ignored)
	err = os.WriteFile(filepath.Join(pm.InstanceDir("inst1"), "other.txt"), []byte("not a log"), 0644)
	require.NoError(t, err)

	// List sessions
	sessions, err = pm.ListSessionLogs("inst1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, sessions)
}
