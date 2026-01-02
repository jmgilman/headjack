---
sidebar_position: 3
title: hjk ps
description: List instances and sessions
---

# hjk ps

List instances or sessions managed by Headjack.

## Synopsis

```bash
hjk ps [branch] [flags]
```

## Description

Lists instances or sessions managed by Headjack.

By default, lists instances for the current repository. If a branch is specified, lists sessions for that instance instead.

Use `--all` to list instances across all repositories (only applies when listing instances, not sessions).

## Arguments

| Argument | Description |
|----------|-------------|
| `branch` | Git branch name to list sessions for (optional) |

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--all` | `-a` | bool | `false` | List instances across all repositories |

### Inherited Flags

| Flag | Type | Description |
|------|------|-------------|
| `--multiplexer` | string | Terminal multiplexer to use (`tmux`, `zellij`) |

## Output

### Instance Listing

When listing instances, displays a table with:

| Column | Description |
|--------|-------------|
| BRANCH | Git branch name |
| STATUS | Instance status (`running`, `stopped`) |
| SESSIONS | Number of sessions in the instance |
| CREATED | Relative time since creation |

### Session Listing

When listing sessions (branch argument provided), displays a table with:

| Column | Description |
|--------|-------------|
| SESSION | Session name |
| TYPE | Session type (`shell`, `claude`, `gemini`, `codex`) |
| STATUS | Session status (`detached`) |
| CREATED | Relative time since creation |
| ACCESSED | Relative time since last access |

## Examples

```bash
# List instances for current repository
hjk ps

# List all instances across all repositories
hjk ps --all

# List sessions for a specific instance
hjk ps feat/auth
```

## Aliases

This command is also available as `hjk ls`.

## See Also

- [hjk run](run.md) - Create a new instance/session
- [hjk attach](attach.md) - Attach to a session
- [hjk stop](stop.md) - Stop an instance
- [hjk rm](rm.md) - Remove an instance
