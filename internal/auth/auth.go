// Package auth provides authentication for agent CLIs.
package auth

import (
	"encoding/json"
	"fmt"
)

// CredentialType distinguishes between subscription-based and API key authentication.
type CredentialType string

const (
	// CredentialTypeSubscription represents OAuth/subscription-based authentication.
	CredentialTypeSubscription CredentialType = "subscription"

	// CredentialTypeAPIKey represents direct API key authentication.
	CredentialTypeAPIKey CredentialType = "apikey"
)

// Credential holds a provider's authentication credential with its type.
type Credential struct {
	Type  CredentialType `json:"type"`
	Value string         `json:"value"`
}

// MarshalJSON implements json.Marshaler.
func (c Credential) MarshalJSON() ([]byte, error) {
	type credentialAlias Credential
	return json.Marshal(credentialAlias(c))
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *Credential) UnmarshalJSON(data []byte) error {
	type credentialAlias Credential
	var alias credentialAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*c = Credential(alias)
	return nil
}

// ProviderInfo describes a provider's authentication options and configuration.
type ProviderInfo struct {
	// Name is the provider identifier (e.g., "claude", "gemini", "codex").
	Name string

	// SubscriptionEnvVar is the environment variable for subscription credentials.
	SubscriptionEnvVar string

	// APIKeyEnvVar is the environment variable for API key credentials.
	APIKeyEnvVar string

	// KeychainAccount is the keychain account name for storing credentials.
	KeychainAccount string

	// RequiresContainerSetup indicates whether subscription credentials need
	// file setup in the container (e.g., writing config files).
	RequiresContainerSetup bool
}

// Storage abstracts credential storage backends.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/storage.go . Storage
type Storage interface {
	// Set stores a credential.
	Set(account, secret string) error

	// Get retrieves a credential.
	Get(account string) (string, error)

	// Delete removes a credential.
	Delete(account string) error
}

// Provider authenticates with an agent CLI and manages credentials.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/provider.go . Provider
type Provider interface {
	// Info returns metadata about this provider.
	Info() ProviderInfo

	// CheckSubscription checks if subscription credentials exist and returns them.
	// For providers that auto-detect credentials (Gemini, Codex), this reads from
	// the expected file locations. For providers requiring manual entry (Claude),
	// this returns an error with instructions.
	CheckSubscription() (string, error)

	// ValidateSubscription validates a subscription credential value.
	ValidateSubscription(value string) error

	// ValidateAPIKey validates an API key credential value.
	ValidateAPIKey(value string) error

	// Store saves a credential to storage.
	Store(storage Storage, cred Credential) error

	// Load retrieves the stored credential for this provider.
	Load(storage Storage) (*Credential, error)
}

// StoreCredential is a helper function to store a credential in JSON format.
func StoreCredential(storage Storage, account string, cred Credential) error {
	data, err := json.Marshal(cred)
	if err != nil {
		return fmt.Errorf("marshal credential: %w", err)
	}
	return storage.Set(account, string(data))
}

// LoadCredential is a helper function to load a credential from JSON format.
func LoadCredential(storage Storage, account string) (*Credential, error) {
	data, err := storage.Get(account)
	if err != nil {
		return nil, err
	}

	var cred Credential
	if err := json.Unmarshal([]byte(data), &cred); err != nil {
		return nil, fmt.Errorf("unmarshal credential: %w", err)
	}
	return &cred, nil
}
