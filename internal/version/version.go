// Package version provides build-time version information.
// These variables are set via ldflags during build:
//
//	go build -ldflags "-X github.com/jmgilman/headjack/internal/version.Version=v1.0.0 \
//	                   -X github.com/jmgilman/headjack/internal/version.Commit=abc123 \
//	                   -X github.com/jmgilman/headjack/internal/version.Date=2025-01-01"
package version

var (
	// Version is the semantic version of the build.
	Version = "dev"

	// Commit is the git commit SHA of the build.
	Commit = "none"

	// Date is the build date in ISO 8601 format.
	Date = "unknown"
)
