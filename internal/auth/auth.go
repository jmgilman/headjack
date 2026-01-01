// Package auth provides authentication for agent CLIs.
package auth

import "context"

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
	// Authenticate runs the provider's authentication flow and stores the credential.
	// This typically involves interactive user input (browser login, code entry, etc.).
	Authenticate(ctx context.Context, storage Storage) error

	// Get retrieves the stored credential for this provider.
	Get(storage Storage) (string, error)
}
