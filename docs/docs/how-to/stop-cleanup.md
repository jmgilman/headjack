---
sidebar_position: 10
title: Stop and Remove Instances
description: Stop containers and clean up instances when finished
---

# Stop and Remove Instances

This guide shows you how to stop running containers and remove instances when you're done with them.

## Stop an instance

Stop the container but preserve the worktree:

```bash
hjk stop feat/auth
```

The instance remains in the catalog. Resume it later by running any command for that branch—Headjack automatically restarts stopped instances:

```bash
hjk agent feat/auth claude --prompt "Continue working on auth"
```

## Kill a specific session

Terminate a single session without affecting other sessions or the instance:

```bash
hjk kill feat/auth/debug-shell
```

The format is `<branch>/<session>`. The branch can contain slashes (e.g., `feat/auth/v2`), so the session name is everything after the last slash.

## Remove an instance entirely

Remove an instance, its container, and the git worktree:

```bash
hjk rm feat/auth
```

You'll be prompted to confirm:

```
This will remove instance abc123 for branch feat/auth.
Worktree at /path/to/worktrees/feat-auth will be deleted.
Are you sure? [y/N]
```

**Warning:** This deletes any uncommitted work in the worktree.

## Force remove without confirmation

Skip the confirmation prompt:

```bash
hjk rm feat/auth --force
```

## Check what's running before cleanup

List all instances:

```bash
hjk ps
```

List sessions for a specific instance:

```bash
hjk ps feat/auth
```

## Clean up workflow

A typical cleanup workflow after finishing a feature:

1. Make sure your work is committed and pushed in the container session.

2. Kill any remaining sessions:

   ```bash
   hjk kill feat/auth/claude-main
   hjk kill feat/auth/debug-shell
   ```

3. Remove the instance:

   ```bash
   hjk rm feat/auth --force
   ```

## See also

- [Manage Sessions](manage-sessions.md) - check on sessions before stopping
- [Recover from Container Crashes](recover-from-crash.md) - handle unexpected failures
