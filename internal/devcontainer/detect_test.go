package devcontainer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	t.Run("finds devcontainer.json in .devcontainer directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))

		configPath := filepath.Join(devcontainerDir, "devcontainer.json")
		require.NoError(t, os.WriteFile(configPath, []byte(`{"name": "test"}`), 0644))

		result := Detect(tmpDir)
		assert.Equal(t, configPath, result)
	})

	t.Run("finds devcontainer.json in root directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".devcontainer.json")
		require.NoError(t, os.WriteFile(configPath, []byte(`{"name": "test"}`), 0644))

		result := Detect(tmpDir)
		assert.Equal(t, configPath, result)
	})

	t.Run("prefers .devcontainer/devcontainer.json over .devcontainer.json", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create both files
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))
		preferredPath := filepath.Join(devcontainerDir, "devcontainer.json")
		require.NoError(t, os.WriteFile(preferredPath, []byte(`{"name": "preferred"}`), 0644))

		rootPath := filepath.Join(tmpDir, ".devcontainer.json")
		require.NoError(t, os.WriteFile(rootPath, []byte(`{"name": "root"}`), 0644))

		result := Detect(tmpDir)
		assert.Equal(t, preferredPath, result)
	})

	t.Run("returns empty string when no config found", func(t *testing.T) {
		tmpDir := t.TempDir()

		result := Detect(tmpDir)
		assert.Empty(t, result)
	})

	t.Run("returns empty string for non-existent directory", func(t *testing.T) {
		result := Detect("/non/existent/path")
		assert.Empty(t, result)
	})
}

func TestHasConfig(t *testing.T) {
	t.Run("returns true when config exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))

		configPath := filepath.Join(devcontainerDir, "devcontainer.json")
		require.NoError(t, os.WriteFile(configPath, []byte(`{"name": "test"}`), 0644))

		assert.True(t, HasConfig(tmpDir))
	})

	t.Run("returns false when no config exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		assert.False(t, HasConfig(tmpDir))
	})
}
