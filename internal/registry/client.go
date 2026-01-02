package registry

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// client implements the Client interface using go-containerregistry.
type client struct {
	config ClientConfig
}

// NewClient creates a new registry client with the given configuration.
func NewClient(cfg ClientConfig) Client {
	return &client{config: cfg}
}

// GetMetadata fetches metadata for an image reference.
func (c *client) GetMetadata(ctx context.Context, ref string) (*ImageMetadata, error) {
	// Build name options for reference parsing
	var nameOpts []name.Option
	if c.config.Insecure {
		nameOpts = append(nameOpts, name.Insecure)
	}

	// Parse the image reference
	parsedRef, err := name.ParseReference(ref, nameOpts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRef, err)
	}

	// Build remote options
	opts := []remote.Option{
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
		remote.WithContext(ctx),
		// Use current platform to handle multi-arch images
		remote.WithPlatform(v1.Platform{
			Architecture: runtime.GOARCH,
			OS:           "linux",
		}),
	}

	// Allow insecure (HTTP and skip TLS verify) connections if configured
	if c.config.Insecure {
		// Clone http.DefaultTransport to preserve proxy, keep-alive, and timeout settings.
		// Fall back to a basic transport if the type assertion fails (shouldn't happen in practice).
		var insecureTransport *http.Transport
		if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
			insecureTransport = defaultTransport.Clone()
		} else {
			insecureTransport = &http.Transport{}
		}
		insecureTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // intentional for insecure mode
		opts = append(opts, remote.WithTransport(insecureTransport))
	}

	// Fetch the image
	img, err := remote.Image(parsedRef, opts...)
	if err != nil {
		return nil, c.mapError(err)
	}

	// Get the image digest
	digest, err := img.Digest()
	if err != nil {
		return nil, fmt.Errorf("failed to get image digest: %w", err)
	}

	// Get the config file (contains labels, architecture, OS, etc.)
	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get image config: %w", err)
	}

	// Build the metadata response
	metadata := &ImageMetadata{
		Digest:       digest.String(),
		Labels:       configFile.Config.Labels,
		Architecture: configFile.Architecture,
		OS:           configFile.OS,
	}

	// Set created time if available
	if !configFile.Created.IsZero() {
		metadata.Created = configFile.Created.Time
	}

	return metadata, nil
}

// mapError converts go-containerregistry errors to sentinel errors.
func (c *client) mapError(err error) error {
	// Check for transport errors (HTTP status codes)
	var transportErr *transport.Error
	if errors.As(err, &transportErr) {
		for _, diagnostic := range transportErr.Errors {
			switch diagnostic.Code {
			case transport.UnauthorizedErrorCode:
				return fmt.Errorf("%w: %s", ErrUnauthorized, err)
			case transport.ManifestUnknownErrorCode, transport.NameUnknownErrorCode:
				return fmt.Errorf("%w: %s", ErrImageNotFound, err)
			}
		}
		// Check HTTP status code as fallback
		switch transportErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Errorf("%w: %s", ErrUnauthorized, err)
		case http.StatusNotFound:
			return fmt.Errorf("%w: %s", ErrImageNotFound, err)
		}
	}

	return fmt.Errorf("registry error: %w", err)
}
