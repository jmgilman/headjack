// Package names provides Docker-style random name generation for sessions.
package names

import (
	"fmt"

	"github.com/docker/docker/pkg/namesgenerator"
)

// ExistsFn checks if a name already exists.
type ExistsFn func(name string) bool

// Generate returns a random adjective_surname name (e.g., "focused_turing").
func Generate() string {
	return namesgenerator.GetRandomName(0)
}

// GenerateUnique returns a name that doesn't exist according to existsFn.
// Returns an error if unable to find a unique name after maxAttempts tries.
func GenerateUnique(existsFn ExistsFn, maxAttempts int) (string, error) {
	if maxAttempts <= 0 {
		maxAttempts = 100
	}

	for range maxAttempts {
		name := Generate()
		if !existsFn(name) {
			return name, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique name after %d attempts", maxAttempts)
}
