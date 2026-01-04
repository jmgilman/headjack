#!/bin/bash
set -e

echo "Installing golangci-lint..."
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)/bin"

echo "Installing Go dependencies..."
go mod download

echo "Installing docs dependencies..."
cd docs && npm ci && cd ..

echo "Verifying installation..."
echo "  Go: $(go version)"
echo "  Node: $(node --version)"
echo "  golangci-lint: $(golangci-lint --version)"
echo "  just: $(just --version)"
echo "  hadolint: $(hadolint --version)"
echo "  gh: $(gh --version | head -1)"
echo "  docker: $(docker --version)"

echo "Development environment ready!"
