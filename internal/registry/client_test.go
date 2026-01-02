package registry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient(ClientConfig{})
	require.NotNil(t, client)
}

func TestClient_GetMetadata(t *testing.T) {
	ctx := context.Background()

	t.Run("fetches metadata from registry", func(t *testing.T) {
		// Start a test registry
		reg := registry.New()
		server := httptest.NewServer(reg)
		defer server.Close()

		// Create and push a test image
		img, err := random.Image(1024, 1)
		require.NoError(t, err)

		// Get the registry host from the server URL
		regHost := strings.TrimPrefix(server.URL, "http://")
		ref, err := name.ParseReference(regHost + "/test/image:latest")
		require.NoError(t, err)

		// Push the image to the test registry
		err = remote.Write(ref, img)
		require.NoError(t, err)

		// Test GetMetadata
		client := NewClient(ClientConfig{Insecure: true})
		metadata, err := client.GetMetadata(ctx, regHost+"/test/image:latest")

		require.NoError(t, err)
		assert.NotEmpty(t, metadata.Digest)
		assert.True(t, strings.HasPrefix(metadata.Digest, "sha256:"))
		// Note: random.Image may not set Architecture/OS, so we just verify the call succeeded
	})

	t.Run("returns ErrInvalidRef for malformed reference", func(t *testing.T) {
		client := NewClient(ClientConfig{})
		_, err := client.GetMetadata(ctx, ":::invalid:::reference")

		assert.ErrorIs(t, err, ErrInvalidRef)
	})

	t.Run("returns ErrImageNotFound for missing image", func(t *testing.T) {
		// Start a test registry
		reg := registry.New()
		server := httptest.NewServer(reg)
		defer server.Close()

		regHost := strings.TrimPrefix(server.URL, "http://")
		client := NewClient(ClientConfig{Insecure: true})
		_, err := client.GetMetadata(ctx, regHost+"/nonexistent/image:latest")

		assert.ErrorIs(t, err, ErrImageNotFound)
	})
}

func TestClient_mapError(t *testing.T) {
	c := &client{}

	t.Run("maps UNAUTHORIZED error code to ErrUnauthorized", func(t *testing.T) {
		transportErr := &transport.Error{
			StatusCode: http.StatusUnauthorized,
			Errors: []transport.Diagnostic{
				{Code: transport.UnauthorizedErrorCode},
			},
		}
		result := c.mapError(transportErr)

		assert.ErrorIs(t, result, ErrUnauthorized)
	})

	t.Run("maps 401 status to ErrUnauthorized", func(t *testing.T) {
		transportErr := &transport.Error{
			StatusCode: http.StatusUnauthorized,
		}
		result := c.mapError(transportErr)

		assert.ErrorIs(t, result, ErrUnauthorized)
	})

	t.Run("maps 403 status to ErrUnauthorized", func(t *testing.T) {
		transportErr := &transport.Error{
			StatusCode: http.StatusForbidden,
		}
		result := c.mapError(transportErr)

		assert.ErrorIs(t, result, ErrUnauthorized)
	})

	t.Run("maps MANIFEST_UNKNOWN error code to ErrImageNotFound", func(t *testing.T) {
		transportErr := &transport.Error{
			StatusCode: http.StatusNotFound,
			Errors: []transport.Diagnostic{
				{Code: transport.ManifestUnknownErrorCode},
			},
		}
		result := c.mapError(transportErr)

		assert.ErrorIs(t, result, ErrImageNotFound)
	})

	t.Run("maps NAME_UNKNOWN error code to ErrImageNotFound", func(t *testing.T) {
		transportErr := &transport.Error{
			StatusCode: http.StatusNotFound,
			Errors: []transport.Diagnostic{
				{Code: transport.NameUnknownErrorCode},
			},
		}
		result := c.mapError(transportErr)

		assert.ErrorIs(t, result, ErrImageNotFound)
	})

	t.Run("maps 404 status to ErrImageNotFound", func(t *testing.T) {
		transportErr := &transport.Error{
			StatusCode: http.StatusNotFound,
		}
		result := c.mapError(transportErr)

		assert.ErrorIs(t, result, ErrImageNotFound)
	})

	t.Run("wraps unknown errors", func(t *testing.T) {
		unknownErr := errors.New("some unknown error")
		result := c.mapError(unknownErr)

		assert.Contains(t, result.Error(), "registry error")
		assert.ErrorIs(t, result, unknownErr)
	})
}
