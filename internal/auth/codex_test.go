package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCodexProvider(t *testing.T) {
	p := NewCodexProvider()
	assert.NotNil(t, p)
}

func TestReadCodexAuth(t *testing.T) {
	// Save and restore original path
	originalDir := codexConfigDir
	t.Cleanup(func() { codexConfigDir = originalDir })

	t.Run("valid auth.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		codexConfigDir = tmpDir

		authData := `{"access_token":"test-token","refresh_token":"test-refresh"}`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "auth.json"), []byte(authData), 0o600))

		got, err := readCodexAuth()
		require.NoError(t, err)
		assert.JSONEq(t, authData, string(got))
	})

	t.Run("auth.json not found", func(t *testing.T) {
		codexConfigDir = "/nonexistent/path"

		got, err := readCodexAuth()
		assert.Nil(t, got)
		assert.ErrorContains(t, err, "codex auth.json not found")
	})

	t.Run("empty auth.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		codexConfigDir = tmpDir

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "auth.json"), []byte(""), 0o600))

		got, err := readCodexAuth()
		assert.Nil(t, got)
		assert.ErrorContains(t, err, "auth.json is empty")
	})
}
