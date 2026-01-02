---
sidebar_position: 11
title: Recover from Container Crashes
description: Handle container or session failures and resume work
---

# Recover from Container Crashes

This guide shows you how to recover when a container stops unexpectedly or a session becomes unresponsive.

## Check instance status

First, check the status of your instances:

```bash
hjk ps
```

Look for instances with a `stopped` status:

```
BRANCH      STATUS   SESSIONS  CREATED
feat/auth   stopped  0         2h ago
feat/api    running  2         1h ago
```

## Resume a stopped instance

If the container stopped but the instance still exists, simply run a new session:

```bash
hjk run feat/auth --agent claude "Continue where we left off"
```

Headjack automatically restarts the container and creates a new session. Your git worktree is preserved with all previous work.

## Check the last session's output

Before starting a new session, review what the previous session was doing:

```bash
hjk logs feat/auth previous-session-name --full
```

If you don't remember the session name, the logs directory is at `~/.local/share/headjack/logs/`.

## Kill an unresponsive session

If a session is stuck but the container is still running:

```bash
hjk kill feat/auth/stuck-session
```

Then start a fresh session:

```bash
hjk run feat/auth --agent claude
```

## Force remove a broken instance

If the instance is in a bad state and won't respond to normal commands:

```bash
hjk rm feat/auth --force
```

This removes the instance from Headjack's catalog and cleans up the worktree. You can then start fresh:

```bash
hjk run feat/auth --agent claude
```

## Recover work from the worktree

If you need to recover uncommitted changes before removing an instance:

1. Find the worktree location from `hjk ps` output or check your worktree configuration.

2. Copy any uncommitted files you need.

3. Then remove the instance:

   ```bash
   hjk rm feat/auth --force
   ```

## Prevent data loss

To minimize impact from crashes:

- Commit work frequently within your agent sessions
- Use detached mode (`-d`) and monitor with `hjk logs -f` so crashes don't close your terminal
- Push to remote regularly to back up your work

## See also

- [Stop and Remove Instances](stop-cleanup.md) - normal cleanup procedures
- [Manage Sessions](manage-sessions.md) - watch for problems in real-time
