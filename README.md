# Headjack

![Headjack Banner](docs/static/img/banner.png)

A macOS and Linux CLI tool for spawning isolated LLM coding agents in container environments.

## Overview

Headjack gives each AI coding agent its own VM-isolated container with a dedicated git worktree, enabling safe parallel development across multiple branches. Run Claude Code, Gemini CLI, or Codex CLI simultaneously on different features without conflicts.

## Features

- **Isolation** - Each agent runs in its own container with a separate filesystem
- **Branch-based Workflows** - One instance per git branch with dedicated worktrees
- **Session Persistence** - Attach and detach from long-running agent sessions
- **Multi-agent Support** - Claude Code, Gemini CLI, and Codex CLI
- **Secure Authentication** - Credentials stored in the system keychain

## Installation

```bash
brew install GilmanLab/tap/headjack
```

## Quick Start

```bash
# Authenticate your agent
hjk auth claude

# Start an agent on a feature branch
hjk run feat/auth --agent claude "Implement JWT authentication"

# List running instances
hjk ps

# Attach to an existing session
hjk attach feat/auth
```

## Documentation

Full documentation is available at [headjack.gilman.io](https://headjack.gilman.io).

- [Getting Started Tutorial](https://headjack.gilman.io/tutorials/getting-started) - Installation and first agent
- [CLI Reference](https://headjack.gilman.io/reference/cli/run) - Complete command documentation
- [Architecture](https://headjack.gilman.io/explanation/architecture) - How Headjack works

## Requirements

- macOS or Linux
- [Docker](https://www.docker.com/) (default) or [Podman](https://podman.io/)
- Git

## License

[MIT](LICENSE)
