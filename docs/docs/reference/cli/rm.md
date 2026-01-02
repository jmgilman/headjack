---
sidebar_position: 7
title: hjk rm
description: Remove an instance entirely
---

# hjk rm

Remove an instance, including its container and git worktree.

## Synopsis

```bash
hjk rm <branch> [flags]
```

## Description

Removes an instance entirely. This command:

1. Stops the container if running
2. Deletes the container
3. Deletes the git worktree
4. Removes the instance from the catalog

**Warning**: This deletes uncommitted work in the worktree. Make sure to commit or stash any changes you want to keep before removing an instance.

## Arguments

| Argument | Description |
|----------|-------------|
| `branch` | Git branch name of the instance to remove (required) |

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--force` | `-f` | bool | `false` | Skip confirmation prompt |

### Inherited Flags

| Flag | Type | Description |
|------|------|-------------|
| `--multiplexer` | string | Terminal multiplexer to use (`tmux`, `zellij`) |

## Examples

```bash
# Remove with confirmation prompt
hjk rm feat/auth

# Force remove without confirmation
hjk rm feat/auth --force
```

## Confirmation Prompt

Without the `--force` flag, the command displays:

```
This will remove instance <id> for branch <branch>.
Worktree at <path> will be deleted.
Are you sure? [y/N]
```

Type `y` or `yes` to confirm, or any other input (including Enter) to cancel.

## See Also

- [hjk stop](stop.md) - Stop without removing
- [hjk ps](ps.md) - List instances
- [hjk recreate](recreate.md) - Recreate container without removing worktree
