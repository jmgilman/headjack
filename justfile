# Default recipe - runs all checks
default: check

# Run all checks: format, lint, and test
check: fmt lint test

# Run linter
lint:
    golangci-lint run

# Format code
fmt:
    gofmt -w .
    goimports -w -local github.com/jmgilman/headjack .

# Run tests
test:
    go test ./...

# Run tests with verbose output
test-verbose:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# =============================================================================
# Container Images
# =============================================================================

# Build base image locally
build-base:
    docker build -t headjack:base images/base

# Build systemd image locally (depends on base)
build-systemd: build-base
    docker build -t headjack:systemd --build-arg BASE_IMAGE=headjack:base images/systemd

# Build dind image locally (depends on systemd)
build-dind: build-systemd
    docker build -t headjack:dind --build-arg BASE_IMAGE=headjack:systemd images/dind

# Build all container images
build-images: build-dind

# Lint all Dockerfiles
lint-dockerfiles:
    hadolint images/base/Dockerfile
    hadolint images/systemd/Dockerfile
    hadolint images/dind/Dockerfile
