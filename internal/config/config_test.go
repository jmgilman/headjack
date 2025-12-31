package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Load_CreatesDefaultIfMissing(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	loader, err := NewLoader()
	require.NoError(t, err)

	cfg, err := loader.Load()
	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, "", cfg.Default.Agent)
	assert.Equal(t, defaultBaseImage, cfg.Default.BaseImage)
	assert.Contains(t, cfg.Storage.Worktrees, "headjack")
	assert.Contains(t, cfg.Storage.Catalog, "catalog.json")
	assert.Contains(t, cfg.Storage.Logs, "logs")

	// Verify file was created
	_, err = os.Stat(loader.Path())
	assert.NoError(t, err)
}

func TestLoader_Load_ReadsExistingConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create config manually
	configDir := filepath.Join(tmpHome, ".config", "headjack")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configContent := `
default:
  agent: claude
  base_image: custom:latest
storage:
  worktrees: ~/custom/worktrees
  catalog: ~/custom/catalog.json
  logs: ~/custom/logs
agents:
  claude:
    env:
      FOO: bar
`
	require.NoError(t, os.WriteFile(
		filepath.Join(configDir, "config.yaml"),
		[]byte(configContent),
		0644,
	))

	loader, err := NewLoader()
	require.NoError(t, err)

	cfg, err := loader.Load()
	require.NoError(t, err)

	assert.Equal(t, "claude", cfg.Default.Agent)
	assert.Equal(t, "custom:latest", cfg.Default.BaseImage)
	assert.Equal(t, filepath.Join(tmpHome, "custom", "worktrees"), cfg.Storage.Worktrees)
	assert.Equal(t, filepath.Join(tmpHome, "custom", "catalog.json"), cfg.Storage.Catalog)
	assert.Equal(t, filepath.Join(tmpHome, "custom", "logs"), cfg.Storage.Logs)

	// Test agent env via GetAgentEnv helper
	// Note: viper lowercases all keys
	env := loader.GetAgentEnv("claude")
	assert.Equal(t, "bar", env["foo"])
}

func TestLoader_Load_EnvVarOverride(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("HEADJACK_DEFAULT_AGENT", "gemini")
	t.Setenv("HEADJACK_BASE_IMAGE", "env:image")

	loader, err := NewLoader()
	require.NoError(t, err)

	cfg, err := loader.Load()
	require.NoError(t, err)

	// Env vars should override file defaults
	assert.Equal(t, "gemini", cfg.Default.Agent)
	assert.Equal(t, "env:image", cfg.Default.BaseImage)
}

func TestLoader_Path(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	loader, err := NewLoader()
	require.NoError(t, err)

	expected := filepath.Join(tmpHome, ".config", "headjack", "config.yaml")
	assert.Equal(t, expected, loader.Path())
}

func TestLoader_Get(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	loader, err := NewLoader()
	require.NoError(t, err)

	_, err = loader.Load()
	require.NoError(t, err)

	t.Run("valid key returns value", func(t *testing.T) {
		val, err := loader.Get("default.base_image")
		require.NoError(t, err)
		assert.Equal(t, defaultBaseImage, val)
	})

	t.Run("invalid key returns error", func(t *testing.T) {
		_, err := loader.Get("invalid.key")
		assert.ErrorIs(t, err, ErrInvalidKey)
	})
}

func TestLoader_Set(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	loader, err := NewLoader()
	require.NoError(t, err)

	_, err = loader.Load()
	require.NoError(t, err)

	t.Run("sets valid key", func(t *testing.T) {
		err := loader.Set("default.agent", "gemini")
		require.NoError(t, err)

		val, err := loader.Get("default.agent")
		require.NoError(t, err)
		assert.Equal(t, "gemini", val)
	})

	t.Run("rejects invalid key", func(t *testing.T) {
		err := loader.Set("invalid.key", "value")
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("rejects invalid agent", func(t *testing.T) {
		err := loader.Set("default.agent", "invalid")
		assert.ErrorIs(t, err, ErrInvalidAgent)
	})

	t.Run("allows empty agent", func(t *testing.T) {
		err := loader.Set("default.agent", "")
		assert.NoError(t, err)
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config with agent", func(t *testing.T) {
		cfg := &Config{
			Default: DefaultConfig{Agent: "claude", BaseImage: "test:latest"},
			Agents:  map[string]AgentConfig{"claude": {}},
			Storage: StorageConfig{Worktrees: "/tmp/worktrees", Catalog: "/tmp/catalog.json", Logs: "/tmp/logs"},
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("valid config without agent", func(t *testing.T) {
		cfg := &Config{
			Default: DefaultConfig{Agent: "", BaseImage: "test:latest"},
			Storage: StorageConfig{Worktrees: "/tmp/worktrees", Catalog: "/tmp/catalog.json", Logs: "/tmp/logs"},
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("invalid default agent", func(t *testing.T) {
		cfg := &Config{
			Default: DefaultConfig{Agent: "invalid", BaseImage: "test:latest"},
			Storage: StorageConfig{Worktrees: "/tmp/worktrees", Catalog: "/tmp/catalog.json", Logs: "/tmp/logs"},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Agent")
	})

	t.Run("invalid agent in map", func(t *testing.T) {
		cfg := &Config{
			Default: DefaultConfig{BaseImage: "test:latest"},
			Agents:  map[string]AgentConfig{"unknown": {}},
			Storage: StorageConfig{Worktrees: "/tmp/worktrees", Catalog: "/tmp/catalog.json", Logs: "/tmp/logs"},
		}
		err := cfg.Validate()
		assert.Error(t, err)
	})

	t.Run("missing required base_image", func(t *testing.T) {
		cfg := &Config{
			Default: DefaultConfig{Agent: ""},
			Storage: StorageConfig{Worktrees: "/tmp/worktrees", Catalog: "/tmp/catalog.json", Logs: "/tmp/logs"},
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "BaseImage")
	})
}

func TestConfig_IsValidAgent(t *testing.T) {
	cfg := &Config{}

	assert.True(t, cfg.IsValidAgent("claude"))
	assert.True(t, cfg.IsValidAgent("gemini"))
	assert.True(t, cfg.IsValidAgent("codex"))
	assert.False(t, cfg.IsValidAgent("invalid"))
	assert.False(t, cfg.IsValidAgent(""))
}

func TestConfig_ValidAgentNames(t *testing.T) {
	cfg := &Config{}
	names := cfg.ValidAgentNames()

	assert.Contains(t, names, "claude")
	assert.Contains(t, names, "gemini")
	assert.Contains(t, names, "codex")
	assert.Len(t, names, 3)
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{"default.agent is valid", "default.agent", nil},
		{"default.base_image is valid", "default.base_image", nil},
		{"storage.worktrees is valid", "storage.worktrees", nil},
		{"storage.catalog is valid", "storage.catalog", nil},
		{"storage.logs is valid", "storage.logs", nil},
		{"agents is valid", "agents", nil},
		{"default is valid", "default", nil},
		{"storage is valid", "storage", nil},
		{"agents.claude is valid", "agents.claude", nil},
		{"agents.claude.env is valid", "agents.claude.env", nil},
		{"agents.gemini is valid", "agents.gemini", nil},
		{"agents.codex is valid", "agents.codex", nil},
		{"agents.invalid returns error", "agents.invalid", ErrInvalidAgent},
		{"unknown.key returns error", "unknown.key", ErrInvalidKey},
		{"empty key returns error", "", ErrInvalidKey},
		{"random key returns error", "foo", ErrInvalidKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoader_expandPath(t *testing.T) {
	tmpHome := "/home/test"
	loader := &Loader{homeDir: tmpHome}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"expands ~/ prefix", "~/foo", filepath.Join(tmpHome, "foo")},
		{"expands ~ alone", "~", tmpHome},
		{"preserves absolute path", "/absolute/path", "/absolute/path"},
		{"preserves relative path", "relative/path", "relative/path"},
		{"handles nested paths", "~/foo/bar/baz", filepath.Join(tmpHome, "foo", "bar", "baz")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.expandPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
