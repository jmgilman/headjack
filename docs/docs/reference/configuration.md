---
sidebar_position: 2
title: Configuration
description: Configuration file format and options
---

# Configuration Reference

Headjack uses a YAML configuration file to store settings. The configuration file is automatically created with default values on first run.

## File Location

The configuration file is located at:

```
~/.config/headjack/config.yaml
```

## Configuration Structure

The configuration file has the following top-level sections:

| Section | Description |
|---------|-------------|
| `default` | Default values for new instances |
| `agents` | Agent-specific configuration |
| `storage` | Storage location configuration |
| `runtime` | Container runtime configuration |

## Configuration Options

### default

Default values applied when creating new instances.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `default.agent` | string | `""` (empty) | Default agent to use. Valid values: `claude`, `gemini`, `codex`. Empty means no default. |
| `default.base_image` | string | `ghcr.io/gilmanlab/headjack:base` | Container image to use for instances. Available variants: `:base` (minimal), `:systemd` (with init), `:dind` (with Docker). |

### agents

Agent-specific configuration. Each agent can have custom environment variables.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `agents.claude.env` | map[string]string | `{"CLAUDE_CODE_MAX_TURNS": "100"}` | Environment variables for Claude agent sessions. |
| `agents.gemini.env` | map[string]string | `{}` | Environment variables for Gemini agent sessions. |
| `agents.codex.env` | map[string]string | `{}` | Environment variables for Codex agent sessions. |

### storage

Storage location configuration. Paths support `~` for home directory expansion.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `storage.worktrees` | string | `~/.local/share/headjack/git` | Directory for git worktrees. |
| `storage.catalog` | string | `~/.local/share/headjack/catalog.json` | Path to the instance catalog file. |
| `storage.logs` | string | `~/.local/share/headjack/logs` | Directory for session log files. |

### runtime

Container runtime configuration.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `runtime.name` | string | `docker` | Container runtime to use. Valid values: `podman`, `apple`, `docker`. |
| `runtime.flags` | map[string]any | `{}` | Additional flags to pass to the container runtime. |

## Example Configuration

A complete configuration file with all options:

```yaml
default:
  agent: claude
  base_image: ghcr.io/gilmanlab/headjack:base

agents:
  claude:
    env:
      CLAUDE_CODE_MAX_TURNS: "100"
  gemini:
    env: {}
  codex:
    env: {}

storage:
  worktrees: ~/.local/share/headjack/git
  catalog: ~/.local/share/headjack/catalog.json
  logs: ~/.local/share/headjack/logs

runtime:
  name: docker
  flags: {}
```

## Managing Configuration

Use the `hjk config` command to view and modify configuration.

### View All Configuration

```bash
hjk config
```

### View a Specific Key

```bash
hjk config default.agent
hjk config storage.worktrees
```

### Set a Value

```bash
hjk config default.agent claude
hjk config runtime.name docker
```

### Edit Configuration File

Open the configuration file in your `$EDITOR`:

```bash
hjk config --edit
```

## Environment Variable Overrides

Configuration values can be overridden using environment variables. See the [Environment Variables](./environment.md) reference for details.

The following environment variables override their corresponding configuration keys:

| Environment Variable | Configuration Key |
|---------------------|-------------------|
| `HEADJACK_DEFAULT_AGENT` | `default.agent` |
| `HEADJACK_BASE_IMAGE` | `default.base_image` |
| `HEADJACK_WORKTREE_DIR` | `storage.worktrees` |

## Validation

Headjack validates configuration values when loading and setting them:

- `default.agent` must be one of: `claude`, `gemini`, `codex` (or empty)
- `default.base_image` is required and cannot be empty
- `runtime.name` must be one of: `podman`, `apple`, `docker`
- All storage paths are required

Invalid values will result in an error message describing the validation failure.
