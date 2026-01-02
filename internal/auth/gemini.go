package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// geminiAccountName is the storage key for Gemini credentials.
const geminiAccountName = "gemini-oauth-creds"

// geminiConfigDir is the path where Gemini CLI stores its configuration.
var geminiConfigDir = filepath.Join(os.Getenv("HOME"), ".gemini")

// GeminiConfig holds all configuration needed to authenticate Gemini CLI.
type GeminiConfig struct {
	OAuthCreds     json.RawMessage `json:"oauth_creds"`
	GoogleAccounts json.RawMessage `json:"google_accounts"`
}

// GeminiProvider authenticates with Gemini CLI.
type GeminiProvider struct{}

// NewGeminiProvider creates a new Gemini authentication provider.
func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{}
}

// Authenticate reads cached Gemini CLI config files and stores them in the keychain.
// If credentials don't exist, it returns an error instructing the user to run gemini first.
func (p *GeminiProvider) Authenticate(_ context.Context, storage Storage) error {
	// Read the cached config from Gemini CLI
	config, err := readGeminiConfig()
	if err != nil {
		return err
	}

	// Store the config as JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := storage.Set(geminiAccountName, string(configJSON)); err != nil {
		return fmt.Errorf("store config: %w", err)
	}

	return nil
}

// Get retrieves the stored Gemini config.
func (p *GeminiProvider) Get(storage Storage) (string, error) {
	return storage.Get(geminiAccountName)
}

// readGeminiConfig reads OAuth credentials and account info from Gemini CLI's cache.
func readGeminiConfig() (*GeminiConfig, error) {
	// Read oauth_creds.json (required)
	oauthPath := filepath.Join(geminiConfigDir, "oauth_creds.json")
	oauthData, err := os.ReadFile(oauthPath) //nolint:gosec // Path is constructed from HOME env var
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("gemini credentials not found: please run 'gemini' and complete the OAuth login first")
		}
		return nil, fmt.Errorf("read oauth_creds.json: %w", err)
	}

	// Validate OAuth creds have a refresh token
	var oauthCreds struct {
		RefreshToken string `json:"refresh_token"`
	}
	if unmarshalErr := json.Unmarshal(oauthData, &oauthCreds); unmarshalErr != nil {
		return nil, fmt.Errorf("parse oauth_creds.json: %w", unmarshalErr)
	}
	if oauthCreds.RefreshToken == "" {
		return nil, errors.New("gemini credentials missing refresh token: please run 'gemini' and complete the OAuth login")
	}

	// Read google_accounts.json (required for OAuth)
	accountsPath := filepath.Join(geminiConfigDir, "google_accounts.json")
	accountsData, err := os.ReadFile(accountsPath) //nolint:gosec // Path is constructed from HOME env var
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("google_accounts.json not found: please run 'gemini' and complete the OAuth login first")
		}
		return nil, fmt.Errorf("read google_accounts.json: %w", err)
	}

	return &GeminiConfig{
		OAuthCreds:     oauthData,
		GoogleAccounts: accountsData,
	}, nil
}
