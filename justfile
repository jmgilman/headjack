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

# =============================================================================
# Integration Tests
# =============================================================================

# Run integration tests (auto-detect container runtime)
integration-test runtime="auto":
    #!/usr/bin/env bash
    set -euo pipefail

    # Build the binary first
    go build -o hjk .

    # Determine runtime
    if [ "{{runtime}}" = "auto" ]; then
        if command -v container &>/dev/null && [ "$(uname)" = "Darwin" ]; then
            RUNTIME=apple
        elif command -v docker &>/dev/null; then
            RUNTIME=docker
        elif command -v podman &>/dev/null; then
            RUNTIME=podman
        else
            echo "No container runtime found"
            exit 1
        fi
    else
        RUNTIME="{{runtime}}"
    fi

    echo "Running integration tests with runtime: $RUNTIME"
    HEADJACK_TEST_RUNTIME=$RUNTIME \
    HJK_BINARY=$(pwd)/hjk \
    go test -v -tags=integration -timeout=30m ./integration/...

# Run integration tests with Docker
integration-test-docker:
    just integration-test docker

# Run integration tests with Podman
integration-test-podman:
    just integration-test podman

# Run integration tests with Apple Containerization (macOS only)
integration-test-apple:
    just integration-test apple

# Run a specific integration test script by name
integration-test-one script runtime="auto":
    #!/usr/bin/env bash
    set -euo pipefail

    go build -o hjk .

    if [ "{{runtime}}" = "auto" ]; then
        if command -v container &>/dev/null && [ "$(uname)" = "Darwin" ]; then
            RUNTIME=apple
        elif command -v docker &>/dev/null; then
            RUNTIME=docker
        elif command -v podman &>/dev/null; then
            RUNTIME=podman
        else
            RUNTIME=docker
        fi
    else
        RUNTIME="{{runtime}}"
    fi

    HEADJACK_TEST_RUNTIME=$RUNTIME \
    HJK_BINARY=$(pwd)/hjk \
    go test -v -tags=integration -run="TestScripts/{{script}}" ./integration/...

# Run integration tests in Lima VM (Linux ARM64 testing on macOS)
integration-test-lima:
    #!/usr/bin/env bash
    set -euo pipefail

    LIMA_INSTANCE="hjk-integration-$$"

    cleanup() {
        echo "Cleaning up Lima instance $LIMA_INSTANCE..."
        limactl stop "$LIMA_INSTANCE" 2>/dev/null || true
        limactl delete "$LIMA_INSTANCE" --force 2>/dev/null || true
    }
    trap cleanup EXIT

    echo "Creating Lima instance $LIMA_INSTANCE with Docker..."
    limactl create --name="$LIMA_INSTANCE" template://docker --tty=false

    echo "Starting Lima instance..."
    limactl start "$LIMA_INSTANCE"

    echo "Syncing project to Lima instance..."
    limactl copy -r . "$LIMA_INSTANCE":/tmp/headjack-test

    echo "Installing Go..."
    limactl shell "$LIMA_INSTANCE" -- bash -c '
        curl -fsSL https://go.dev/dl/go1.23.4.linux-arm64.tar.gz | sudo tar -C /usr/local -xzf -
        echo "export PATH=\$PATH:/usr/local/go/bin" | sudo tee /etc/profile.d/go.sh
    '

    echo "Pulling container image..."
    limactl shell "$LIMA_INSTANCE" -- docker pull ghcr.io/gilmanlab/headjack:base

    echo "Running tests in Lima..."
    limactl shell "$LIMA_INSTANCE" -- bash -c '
        export PATH=$PATH:/usr/local/go/bin
        export DOCKER_HOST=unix:///run/user/$(id -u)/docker.sock
        cd /tmp/headjack-test
        go build -o hjk .
        HEADJACK_TEST_RUNTIME=docker \
        HJK_BINARY=$(pwd)/hjk \
        go test -v -tags=integration -timeout=30m ./integration/...
    '
