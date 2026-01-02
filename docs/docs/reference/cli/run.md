---
sidebar_position: 1
title: hjk run
description: Create an instance and session for a branch
---

# hjk run

Create a new session within an instance for a specified branch.

## Synopsis

```bash
hjk run <branch> [prompt] [flags]
```

## Description

Creates a new session within an instance for the specified branch. If no instance exists for the branch, one is created first by:

- Creating a git worktree at the configured location
- Spawning a new container with the worktree mounted

A new session is always created within the instance. If `--agent` is specified, the agent is started with an optional prompt. Otherwise, the default shell is started.

Unless `--detached` is specified, the terminal attaches to the session. All session output is captured to a log file regardless of attached/detached mode.

If an instance exists but is stopped, it is automatically restarted before creating the new session.

## Arguments

| Argument | Description |
|----------|-------------|
| `branch` | Git branch name for the instance (required) |
| `prompt` | Instructions to pass to the agent (optional, only used with `--agent`) |

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--agent` | | string | | Start the specified agent instead of a shell. Valid values: `claude`, `gemini`, `codex`. If specified without a value, uses the configured `default.agent`. |
| `--name` | | string | | Override the auto-generated session name |
| `--base` | | string | | Override the default base image |
| `--detached` | `-d` | bool | `false` | Create session but do not attach (run in background) |

### Inherited Flags

| Flag | Type | Description |
|------|------|-------------|
| `--multiplexer` | string | Terminal multiplexer to use (`tmux`, `zellij`) |

## Examples

```bash
# Create instance with shell session
hjk run feat/auth

# Create instance with Claude agent
hjk run feat/auth --agent claude "Implement JWT authentication"

# Create additional session in existing instance
hjk run feat/auth --agent gemini --name gemini-experiment

# Create shell session with custom name
hjk run feat/auth --name debug-shell

# Create detached sessions (run in background)
hjk run feat/auth --agent claude -d "Refactor the auth module"
hjk run feat/auth --agent claude -d "Write tests for auth module"

# Use a custom base image
hjk run feat/auth --base my-registry.io/custom-image:latest

# Use default agent from config
hjk run feat/auth --agent
```

## Authentication

When using an agent, the command requires authentication to be configured first:

- **Claude**: Run `hjk auth claude` first
- **Gemini**: Run `hjk auth gemini` first
- **Codex**: Run `hjk auth codex` first

Authentication tokens are automatically injected into the container environment.

## See Also

- [hjk attach](attach.md) - Attach to an existing session
- [hjk ps](ps.md) - List instances and sessions
- [hjk logs](logs.md) - View session output
- [hjk auth](auth.md) - Configure agent authentication
