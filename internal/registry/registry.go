// Package registry provides operations for fetching OCI image metadata from container registries.
package registry

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors for registry operations.
var (
	// ErrImageNotFound is returned when the requested image does not exist.
	ErrImageNotFound = errors.New("image not found")

	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInvalidRef is returned when the image reference is malformed.
	ErrInvalidRef = errors.New("invalid image reference")
)

// ImageMetadata contains OCI image metadata fetched from a registry.
type ImageMetadata struct {
	// Digest is the image's content-addressable digest (e.g., "sha256:...").
	Digest string

	// Labels from the image config (OCI annotations).
	Labels map[string]string

	// Created is when the image was created.
	Created time.Time

	// Architecture is the CPU architecture (e.g., "amd64", "arm64").
	Architecture string

	// OS is the operating system (e.g., "linux").
	OS string
}

// ClientConfig configures the registry client.
type ClientConfig struct {
	// Insecure allows HTTP (non-TLS) connections to registries.
	Insecure bool
}

// Client fetches image metadata from OCI registries.
//
//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/client.go . Client
type Client interface {
	// GetMetadata fetches metadata for an image reference.
	// The reference can be a tag (e.g., "ghcr.io/foo/bar:latest")
	// or digest (e.g., "ghcr.io/foo/bar@sha256:...").
	GetMetadata(ctx context.Context, ref string) (*ImageMetadata, error)
}
