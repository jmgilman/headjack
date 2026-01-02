---
sidebar_position: 4
title: hjk logs
description: View session output
---

# hjk logs

View output from a session without attaching.

## Synopsis

```bash
hjk logs <branch> <session> [flags]
```

## Description

Reads from the session's log file, useful for checking on detached agents without interrupting them. All session output is automatically captured to log files regardless of attached/detached mode.

## Arguments

| Argument | Description |
|----------|-------------|
| `branch` | Git branch name of the instance (required) |
| `session` | Session name to view logs for (required) |

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--follow` | `-f` | bool | `false` | Follow log output in real-time |
| `--lines` | `-n` | int | `100` | Number of lines to show |
| `--full` | | bool | `false` | Show entire log from session start |

### Inherited Flags

| Flag | Type | Description |
|------|------|-------------|
| `--multiplexer` | string | Terminal multiplexer to use (`tmux`, `zellij`) |

## Examples

```bash
# View recent output (last 100 lines)
hjk logs feat/auth happy-panda

# Follow output in real-time
hjk logs feat/auth happy-panda -f

# Show last 500 lines
hjk logs feat/auth happy-panda -n 500

# Show entire log from session start
hjk logs feat/auth happy-panda --full
```

## Behavior

- **Default mode**: Shows the last N lines (default 100) and exits
- **Follow mode** (`-f`): Shows the last N lines, then streams new output as it appears (similar to `tail -f`)
- **Full mode** (`--full`): Shows the entire log file from the beginning

The `--full` flag takes precedence over `--lines` when both are specified.

## Log Storage

Session logs are stored at the path configured in `storage.logs` (default: `~/.local/share/headjack/logs/`). Each session has its own log file identified by instance ID and session ID.

## See Also

- [hjk attach](attach.md) - Attach to a session interactively
- [hjk ps](ps.md) - List sessions to find session names
- [hjk run](run.md) - Create a new session
