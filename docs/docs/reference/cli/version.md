---
sidebar_position: 11
title: hjk version
description: Display version information
---

# hjk version

Display version information for Headjack.

## Synopsis

```bash
hjk version
```

## Description

Displays the version, commit hash, and build date of the Headjack installation.

## Arguments

This command takes no arguments.

## Flags

### Inherited Flags

| Flag | Type | Description |
|------|------|-------------|
| `--multiplexer` | string | Terminal multiplexer to use (`tmux`, `zellij`) |

## Examples

```bash
# Display version information
hjk version
```

## Output

The command outputs three lines:

```
headjack v1.2.3
  commit: abc1234
  built:  2024-01-15T10:30:00Z
```

| Field | Description |
|-------|-------------|
| version | Semantic version number |
| commit | Git commit hash the binary was built from |
| built | Build timestamp in ISO 8601 format |

## See Also

- [Headjack releases](https://github.com/GilmanLab/headjack/releases) - View available versions
