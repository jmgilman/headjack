---
sidebar_position: 1
title: hjk run
description: Create an instance for a branch
---

# hjk run

Create a new instance (worktree + container) for a specified branch.

## Synopsis

```bash
hjk run <branch> [flags]
```

## Description

Creates a new instance for the specified branch. An instance consists of:

1. A git worktree for the branch
2. A container with the worktree mounted at `/workspace`
3. A catalog entry tracking the instance

The container environment is determined by:

1. **Devcontainer (default)**: If the repository contains a `devcontainer.json`, it is used to build and run the container environment automatically.
2. **Base image**: Use `--image` to specify a container image directly, bypassing devcontainer detection.

This command only creates the instance. To start a session within the instance, use:

- [`hjk agent`](agent.md) - Start an agent session (Claude, Gemini, or Codex)
- [`hjk exec`](exec.md) - Execute a command or start a shell session

If an instance already exists for the branch, it is reused. If the instance is stopped, it is automatically restarted.

## Arguments

| Argument | Description |
|----------|-------------|
| `branch` | Git branch name for the instance (required) |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--image` | string | | Use a container image instead of devcontainer |

## Examples

```bash
# Auto-detect devcontainer.json (recommended)
hjk run feat/auth

# Use a specific container image (bypasses devcontainer)
hjk run feat/auth --image my-registry.io/custom-image:latest

# Typical workflow: create instance, then start agent
hjk run feat/auth
hjk agent feat/auth claude --prompt "Implement JWT authentication"

# Or start a shell session
hjk run feat/auth
hjk exec feat/auth
```

## Workflow

The typical workflow separates instance creation from session management:

```bash
# Step 1: Create the instance
hjk run feat/auth

# Step 2: Start an agent session
hjk agent feat/auth claude --prompt "Your prompt here"

# Step 3: Later, attach to the session
hjk attach feat/auth
```

This separation allows you to:

- Create instances without immediately starting sessions
- Choose between agent sessions (`hjk agent`) or shell sessions (`hjk exec`)
- Run quick commands without creating persistent sessions (`hjk exec --no-mux`)

## See Also

- [hjk agent](agent.md) - Start an agent session
- [hjk exec](exec.md) - Execute commands or start shell sessions
- [hjk ps](ps.md) - List instances and sessions
- [hjk stop](stop.md) - Stop an instance
- [hjk rm](rm.md) - Remove an instance
