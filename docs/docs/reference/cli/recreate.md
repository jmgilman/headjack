---
sidebar_position: 8
title: hjk recreate
description: Recreate an instance container
---

# hjk recreate

Recreate the container for an instance while preserving the worktree.

## Synopsis

```bash
hjk recreate <branch> [flags]
```

## Description

Recreates the container for an instance. This command:

1. Stops and deletes the existing container
2. Creates a new container with the same worktree

Useful when the container environment is corrupted or needs a fresh state. The worktree (and all git-tracked and untracked files) is preserved.

## Arguments

| Argument | Description |
|----------|-------------|
| `branch` | Git branch name of the instance to recreate (required) |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--base` | string | | Use a different base image for the new container |

### Inherited Flags

| Flag | Type | Description |
|------|------|-------------|
| `--multiplexer` | string | Terminal multiplexer to use (`tmux`, `zellij`) |

## Examples

```bash
# Recreate with same image
hjk recreate feat/auth

# Recreate with new image
hjk recreate feat/auth --base my-registry.io/new-image:v2
```

## Use Cases

- **Corrupted container**: When a container's environment becomes corrupted or unstable
- **Image update**: When you want to use a newer version of the base image
- **Clean slate**: When you want to reset the container state without losing code changes
- **Configuration change**: When you need to apply new container configuration

## Behavior

- All running sessions in the container are terminated
- The container is deleted and a new one is created
- The git worktree directory is preserved and remounted
- A new instance ID is generated for the new container

## See Also

- [hjk stop](stop.md) - Stop without recreating
- [hjk rm](rm.md) - Remove instance entirely
- [hjk run](run.md) - Create a new session after recreating
