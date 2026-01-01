// Package multiplexer provides an abstraction over terminal multiplexer operations.
// It defines a generic interface that can be implemented by different backends
// (Zellij, tmux, etc.) to manage persistent, attachable terminal sessions.
package multiplexer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SessionPrefix is the namespace prefix for all headjack multiplexer sessions.
const SessionPrefix = "hjk"

// Sentinel errors for multiplexer operations.
var (
	ErrSessionNotFound   = errors.New("session not found")
	ErrSessionExists     = errors.New("session already exists")
	ErrAttachFailed      = errors.New("failed to attach to session")
	ErrCreateFailed      = errors.New("failed to create session")
	ErrInvalidInstanceID = errors.New("instance ID cannot contain hyphens")
)

// Session represents a multiplexer session.
type Session struct {
	ID        string    // Unique session identifier (multiplexer-assigned)
	Name      string    // Human-readable name
	CreatedAt time.Time // Creation timestamp
}

// CreateSessionOpts configures session creation.
type CreateSessionOpts struct {
	// Name is the session name (required).
	// Callers should use FormatSessionName to create properly namespaced names.
	Name    string
	Command []string // Initial command to run (optional, defaults to shell)
	Cwd     string   // Working directory (optional)
	Env     []string // Environment variables (KEY=VALUE format)
}

// Multiplexer provides terminal multiplexer operations.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/multiplexer.go . Multiplexer
type Multiplexer interface {
	// CreateSession creates a new multiplexer session.
	// The session is created in detached mode and can be attached to later.
	// Returns ErrSessionExists if a session with the same name already exists.
	// Returns ErrCreateFailed if session creation fails.
	CreateSession(ctx context.Context, opts *CreateSessionOpts) (*Session, error)

	// AttachSession attaches to an existing session.
	// This is a blocking operation that takes over the terminal.
	// Returns ErrSessionNotFound if session doesn't exist.
	// Returns ErrAttachFailed if attachment fails.
	AttachSession(ctx context.Context, sessionName string) error

	// ListSessions returns all active sessions.
	ListSessions(ctx context.Context) ([]Session, error)

	// KillSession terminates a session.
	// Returns ErrSessionNotFound if session doesn't exist.
	KillSession(ctx context.Context, sessionName string) error
}

// FormatSessionName creates a namespaced session name using the format:
// hjk-<instanceID>-<sessionID>
//
// This ensures session names are unique across instances and easily identifiable
// as belonging to headjack.
//
// The instanceID must not contain hyphens to ensure the name can be parsed
// back with ParseSessionName. Returns an error if instanceID contains hyphens.
func FormatSessionName(instanceID, sessionID string) (string, error) {
	if strings.Contains(instanceID, "-") {
		return "", ErrInvalidInstanceID
	}
	return fmt.Sprintf("%s-%s-%s", SessionPrefix, instanceID, sessionID), nil
}

// ParseSessionName extracts the instance ID and session ID from a namespaced session name.
// Returns empty strings if the name doesn't match the expected format.
func ParseSessionName(name string) (instanceID, sessionID string) {
	// Expected format: hjk-<instanceID>-<sessionID>
	// Minimum length: "hjk-X-Y" = 7 characters
	if len(name) < 7 {
		return "", ""
	}

	// Check prefix
	if name[:4] != SessionPrefix+"-" {
		return "", ""
	}

	// Find the second hyphen after the prefix
	rest := name[4:] // Skip "hjk-"
	for i := range len(rest) {
		if rest[i] == '-' {
			instanceID = rest[:i]
			sessionID = rest[i+1:]
			return instanceID, sessionID
		}
	}

	return "", ""
}
