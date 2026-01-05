---
sidebar_position: 2
title: hjk attach
description: Attach to an existing session
---

# hjk attach

Attach to an existing session using MRU (Most-Recently-Used) selection.

## Synopsis

```bash
hjk attach [branch] [session]
```

## Description

Attaches to an existing session using a most-recently-used (MRU) strategy:

- **No arguments**: Attach to the most recently accessed session across all instances
- **Branch only**: Attach to the most recently accessed session for that instance
- **Branch and session**: Attach to the specified session

If no sessions exist for the resolved scope, the command displays an error suggesting `hjk agent` or `hjk exec` to create one.

To detach from a session without terminating it, use the tmux detach keybinding (`Ctrl+B, d`). This returns you to your host terminal while the session continues running.

## Arguments

| Argument | Description |
|----------|-------------|
| `branch` | Git branch name to filter by (optional) |
| `session` | Session name within the instance (optional, requires branch) |

## Examples

```bash
# Attach to most recently accessed session (global MRU)
hjk attach

# Attach to most recent session for feat/auth instance
hjk attach feat/auth

# Attach to specific session
hjk attach feat/auth claude-main
```

## MRU Selection Strategy

The attach command tracks session access times and uses this to determine which session to attach to:

1. When no arguments are provided, it finds the session with the most recent access time across all instances in all repositories
2. When a branch is provided, it finds the session with the most recent access time within that specific instance
3. Session access times are updated each time you attach

## See Also

- [hjk run](run.md) - Create an instance
- [hjk agent](agent.md) - Start an agent session
- [hjk exec](exec.md) - Execute commands or start shell sessions
- [hjk ps](ps.md) - List instances and sessions
- [hjk kill](kill.md) - Kill a session
