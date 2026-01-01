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
