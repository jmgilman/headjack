---
sidebar_position: 6
title: hjk kill
description: Kill a specific session
---

# hjk kill

Kill a specific session within an instance.

## Synopsis

```bash
hjk kill <branch>/<session>
```

## Description

Terminates the multiplexer session and removes it from the catalog. The instance and other sessions are unaffected.

The argument must be in the format `<branch>/<session>`, where branch is the instance branch name and session is the session name. Since branch names can contain slashes (e.g., `feat/auth`), the command splits on the last slash to separate branch from session.

## Arguments

| Argument | Description |
|----------|-------------|
| `branch/session` | Combined branch and session name separated by `/` (required) |

## Flags

### Inherited Flags

| Flag | Type | Description |
|------|------|-------------|
| `--multiplexer` | string | Terminal multiplexer to use (`tmux`, `zellij`) |

## Examples

```bash
# Kill a session
hjk kill feat/auth/debug-shell

# Kill a session in main branch instance
hjk kill main/claude-experiment

# Branch names with slashes are supported
hjk kill feature/user-auth/my-session
```

## Argument Format

The `<branch>/<session>` format allows branch names that contain slashes:

| Input | Branch | Session |
|-------|--------|---------|
| `main/my-session` | `main` | `my-session` |
| `feat/auth/debug` | `feat/auth` | `debug` |
| `feature/api/v2/test` | `feature/api/v2` | `test` |

## See Also

- [hjk ps](ps.md) - List sessions to find session names
- [hjk stop](stop.md) - Stop the entire instance
- [hjk rm](rm.md) - Remove an instance entirely
