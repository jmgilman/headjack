---
sidebar_position: 3
title: hjk exec
description: Execute a command in an instance's container
---

# hjk exec

Execute a command within an existing instance's container.

## Synopsis

```bash
hjk exec <branch> [command...] [flags]
```

## Description

Executes a command within an existing instance's container. The instance must already exist (created with `hjk run`).

By default, creates a new tmux session and attaches to it. Use `--no-mux` to bypass the multiplexer and execute the command directly (like `docker exec`).

If no command is specified, the default shell (`/bin/bash`) is started.

If the instance is stopped, it is automatically restarted before executing.

## Arguments

| Argument | Description |
|----------|-------------|
| `branch` | Git branch name of the instance (required) |
| `command` | Command and arguments to execute (optional, defaults to shell) |

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--no-mux` | | bool | `false` | Bypass tmux and execute directly |
| `--name` | `-n` | string | | Override the auto-generated session name (ignored with `--no-mux`) |
| `--detached` | `-d` | bool | `false` | Create session but do not attach (ignored with `--no-mux`) |

## Examples

```bash
# Start an interactive shell session (in tmux)
hjk exec feat/auth

# Run a command in tmux
hjk exec feat/auth npm test

# Run a command directly (bypass tmux, output to terminal)
hjk exec feat/auth --no-mux ls -la

# Start shell directly (bypass tmux)
hjk exec feat/auth --no-mux

# Run with custom session name
hjk exec feat/auth --name build-session npm run build

# Run command in background (detached tmux session)
hjk exec feat/auth -d npm run build
```

## Modes

### Multiplexer Mode (default)

Without `--no-mux`, the command runs inside a tmux session:

- Session is created and attached
- Output is logged to a file
- You can detach (`Ctrl+B, d`) and reattach later
- Session persists even if the command completes

### Direct Mode (`--no-mux`)

With `--no-mux`, the command runs directly:

- Output prints directly to your terminal
- No tmux session is created
- No log file is created
- Similar behavior to `docker exec`

Use direct mode for quick one-off commands that don't need session persistence.

## Error Handling

If no instance exists for the branch, you'll see an error with a helpful hint:

```
Error: no instance found for branch "feat/auth"
hint: run 'hjk run feat/auth' to create one
```

## Use Cases

### Interactive Shell

```bash
# Start a persistent shell session
hjk exec feat/auth

# Detach with Ctrl+B, d
# Reattach later with:
hjk attach feat/auth
```

### Running Tests

```bash
# Run tests in a persistent session (can reattach if disconnected)
hjk exec feat/auth npm test

# Or run directly for quick feedback
hjk exec feat/auth --no-mux npm test
```

### Build Commands

```bash
# Run build in background
hjk exec feat/auth -d npm run build

# Check build logs
hjk logs feat/auth <session-name>
```

## See Also

- [hjk run](run.md) - Create an instance
- [hjk agent](agent.md) - Start an agent session
- [hjk attach](attach.md) - Attach to an existing session
- [hjk logs](logs.md) - View session output
- [hjk kill](kill.md) - Kill a session
