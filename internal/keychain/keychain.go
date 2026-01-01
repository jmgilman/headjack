// Package keychain provides access to macOS Keychain for secure credential storage.
package keychain

import "errors"

// ErrNotFound is returned when a credential is not found in the keychain.
var ErrNotFound = errors.New("credential not found in keychain")

// ErrUnsupportedPlatform is returned when keychain operations are attempted on non-macOS platforms.
var ErrUnsupportedPlatform = errors.New("keychain is only supported on macOS")

// Keychain provides secure credential storage using macOS Keychain.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/keychain.go . Keychain
type Keychain interface {
	// Set stores a credential in the keychain.
	Set(account, secret string) error

	// Get retrieves a credential from the keychain.
	// Returns ErrNotFound if the credential does not exist.
	Get(account string) (string, error)

	// Delete removes a credential from the keychain.
	// Returns nil if the credential does not exist.
	Delete(account string) error
}
