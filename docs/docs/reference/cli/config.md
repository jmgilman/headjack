---
sidebar_position: 10
title: hjk config
description: View and modify configuration
---

# hjk config

View and modify Headjack configuration.

## Synopsis

```bash
hjk config [key] [value] [flags]
```

## Description

View and modify Headjack configuration:

- **No arguments**: Display all configuration
- **One argument**: Display the value for the specified key
- **Two arguments**: Set the value for the specified key

## Arguments

| Argument | Description |
|----------|-------------|
| `key` | Configuration key in dot notation (e.g., `default.agent`) (optional) |
| `value` | Value to set for the key (optional, requires key) |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--edit` | bool | `false` | Open config file in `$EDITOR` |

### Inherited Flags

| Flag | Type | Description |
|------|------|-------------|
| `--multiplexer` | string | Terminal multiplexer to use (`tmux`, `zellij`) |

## Examples

```bash
# Show all configuration
hjk config

# Show value for a specific key
hjk config default.agent

# Set a value
hjk config default.agent claude

# Open config file in editor
hjk config --edit
```

## Configuration Keys

Common configuration keys:

| Key | Type | Description |
|-----|------|-------------|
| `default.agent` | string | Default agent when `--agent` is used without a value |
| `default.base_image` | string | Default container base image |
| `default.multiplexer` | string | Default terminal multiplexer (`tmux`, `zellij`) |
| `storage.worktrees` | string | Directory for git worktrees |
| `storage.catalog` | string | Path to the instance catalog file |
| `storage.logs` | string | Directory for session logs |
| `runtime.name` | string | Container runtime (`podman`, `apple`) |

## Configuration File

The configuration file is located at `~/.config/headjack/config.yaml`. If the file does not exist, it is created with default values when you first run `hjk config`.

## Editor Mode

The `--edit` flag opens the configuration file in your preferred editor as specified by the `$EDITOR` environment variable. If `$EDITOR` is not set, the command returns an error.

## Output Format

- String values are printed directly
- Complex values (maps, arrays) are printed as YAML

## See Also

- [hjk run](run.md) - Uses `default.agent` and `default.base_image`
- [hjk auth](auth.md) - Configure agent authentication
