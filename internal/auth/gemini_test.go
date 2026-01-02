package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiConfigJSON(t *testing.T) {
	config := GeminiConfig{
		OAuthCreds:     json.RawMessage(`{"access_token":"test","refresh_token":"1//test"}`),
		GoogleAccounts: json.RawMessage(`{"active":"test@example.com"}`),
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var parsed GeminiConfig
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, config.OAuthCreds, parsed.OAuthCreds)
	assert.Equal(t, config.GoogleAccounts, parsed.GoogleAccounts)
}

func TestReadGeminiConfig(t *testing.T) {
	// Save and restore original path
	originalDir := geminiConfigDir
	t.Cleanup(func() { geminiConfigDir = originalDir })

	t.Run("valid config files", func(t *testing.T) {
		tmpDir := t.TempDir()
		geminiConfigDir = tmpDir

		oauthCreds := `{"access_token":"ya29.test","refresh_token":"1//refresh","scope":"openid"}`
		googleAccounts := `{"active":"test@example.com"}`

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "oauth_creds.json"), []byte(oauthCreds), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "google_accounts.json"), []byte(googleAccounts), 0o600))

		got, err := readGeminiConfig()
		require.NoError(t, err)
		assert.JSONEq(t, oauthCreds, string(got.OAuthCreds))
		assert.JSONEq(t, googleAccounts, string(got.GoogleAccounts))
	})

	t.Run("oauth_creds.json not found", func(t *testing.T) {
		geminiConfigDir = "/nonexistent/path"

		got, err := readGeminiConfig()
		assert.Nil(t, got)
		assert.ErrorContains(t, err, "gemini credentials not found")
	})

	t.Run("invalid oauth json", func(t *testing.T) {
		tmpDir := t.TempDir()
		geminiConfigDir = tmpDir

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "oauth_creds.json"), []byte("{invalid}"), 0o600))

		got, err := readGeminiConfig()
		assert.Nil(t, got)
		assert.ErrorContains(t, err, "parse oauth_creds.json")
	})

	t.Run("missing refresh token", func(t *testing.T) {
		tmpDir := t.TempDir()
		geminiConfigDir = tmpDir

		oauthCreds := `{"access_token":"ya29.test"}`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "oauth_creds.json"), []byte(oauthCreds), 0o600))

		got, err := readGeminiConfig()
		assert.Nil(t, got)
		assert.ErrorContains(t, err, "missing refresh token")
	})

	t.Run("google_accounts.json not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		geminiConfigDir = tmpDir

		oauthCreds := `{"access_token":"ya29.test","refresh_token":"1//refresh"}`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "oauth_creds.json"), []byte(oauthCreds), 0o600))

		got, err := readGeminiConfig()
		assert.Nil(t, got)
		assert.ErrorContains(t, err, "google_accounts.json not found")
	})
}

func TestNewGeminiProvider(t *testing.T) {
	p := NewGeminiProvider()
	assert.NotNil(t, p)
}
