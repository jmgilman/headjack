---
sidebar_position: 5
title: hjk stop
description: Stop a running instance
---

# hjk stop

Stop the container associated with an instance.

## Synopsis

```bash
hjk stop <branch>
```

## Description

Stops the container associated with the specified instance. The worktree is preserved and the instance can be resumed later with `hjk run`.

This command is useful for freeing up system resources when an instance is not actively being used. All session state within the container is lost, but the git worktree with your code changes remains intact.

## Arguments

| Argument | Description |
|----------|-------------|
| `branch` | Git branch name of the instance to stop (required) |

## Flags

### Inherited Flags

| Flag | Type | Description |
|------|------|-------------|
| `--multiplexer` | string | Terminal multiplexer to use (`tmux`, `zellij`) |

## Examples

```bash
# Stop an instance
hjk stop feat/auth
```

## Behavior

When an instance is stopped:

- The container is stopped but not deleted
- The git worktree remains on disk with all changes
- Session state (running processes, terminal state) is lost
- Running `hjk run` on the same branch will restart the container

## See Also

- [hjk run](run.md) - Restart a stopped instance
- [hjk rm](rm.md) - Remove an instance entirely
- [hjk ps](ps.md) - List instances and their status
- [hjk recreate](recreate.md) - Recreate the container
