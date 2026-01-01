# Headjack

Headjack is a CLI tool for spawning isolated CLI-based LLM coding agents in predefined container environments. Each agent runs in its own VM-isolated container with a dedicated git worktree, enabling safe parallel development across multiple branches.

## Repository Structure

```
├── main.go                 # Entry point
├── internal/               # Application code
│   ├── cmd/                # CLI commands (cobra)
│   ├── catalog/            # Instance catalog/persistence
│   ├── config/             # Configuration management
│   ├── container/          # Container runtime abstraction
│   ├── exec/               # Command execution utilities
│   ├── git/                # Git operations
│   ├── instance/           # Instance lifecycle management
│   ├── logging/            # Log file handling
│   ├── multiplexer/        # Terminal multiplexer (zellij)
│   ├── names/              # Random name generation
│   └── version/            # Version information
├── images/                 # Container images
│   └── base/               # Base container image Dockerfile
├── docs/                   # Documentation
│   ├── designs/            # Design documents
│   └── adrs/               # Architecture Decision Records
├── ref/                    # Reference documentation
│   └── go/                 # Go style guides
├── .github/workflows/      # CI/CD workflows
├── .golangci.yml           # Linter configuration
└── justfile                # Development task runner
```

## Development Commands

Use `just` to run common development tasks:

```bash
just check    # Run all checks (format, lint, test)
just fmt      # Format code
just lint     # Run linter
just test     # Run tests
```

## Code Quality Requirements

### Go Code

All changes to Go code MUST:

1. **Pass all checks**: Run `just check` before considering any change complete
2. **Follow style guidelines**: Conform to `ref/go/style.md`
3. **Follow testing guidelines**: Conform to `ref/go/testing.md`

### Dockerfiles

All changes to Dockerfiles MUST pass hadolint:

```bash
hadolint images/base/Dockerfile
```

### Pull Requests

All PRs MUST pass CI before being considered done. Monitor PR status with:

```bash
gh pr checks <pr-number> --watch
```

Wait for all checks to pass. If checks fail, fix the issues and push again.
