# Refactor: Sessions

This document provides context for implementing the session-based refactor. Delete this file once the refactor is complete.

## Summary

We are introducing **sessions** as a new abstraction layer between users and instances. A session is a persistent, attachable/detachable process (shell, agent CLI, etc.) running within an instance, managed by Zellij.

## Why

The original design allowed multiple `resume` calls to the same instance but provided no way to:
- List what's currently running in an instance
- Reattach to a specific process after detaching
- Name or organize concurrent processes
- Track which session was most recently used

## Key Changes

### New Concepts

| Concept | Definition |
|---------|------------|
| Instance | Git worktree + container (unchanged) |
| Session | A Zellij-managed process within an instance (new) |

An instance can have zero or more sessions. Sessions persist across detach/attach cycles.

### CLI Command Changes

| Old | New | Notes |
|-----|-----|-------|
| `new` | `run` | Creates instance if needed, always creates new session. Supports `--detached` |
| `resume` | `attach` | Reattaches to existing session (MRU-based) |
| `list` | `ps` | Lists instances, or sessions if branch specified |
| — | `logs` | View session output without attaching |
| — | `kill` | Terminates a specific session |
| `stop` | `stop` | Unchanged (stops container, kills all sessions) |
| `rm` | `rm` | Unchanged |
| `recreate` | `recreate` | Unchanged (kills all sessions) |

### MRU (Most Recently Used) Behavior

`hjk attach` uses MRU strategy:
- No args: attach to most recent session globally
- Branch only: attach to most recent session for that instance
- Branch + session: attach to explicit session

Requires tracking `last_accessed` timestamp per session.

### Session Naming

Auto-generated names (Docker-style, e.g., `happy-panda`). Override with `--name` flag on `run`.

### Detached Mode

`hjk run` supports `-d, --detached` flag to create a session without attaching:

```bash
hjk run feat/auth --agent claude -d "Refactor auth module"
hjk run feat/auth --agent claude -d "Write tests"
# Two agents now running in parallel
```

This enables parallel agent workflows.

### Session Logging

All session output is captured to log files regardless of attached/detached mode:

```
~/.local/share/headjack/logs/<instance-id>/<session-id>.log
```

`hjk logs` reads these files:

```bash
hjk logs feat/auth happy-panda      # View recent output
hjk logs feat/auth happy-panda -f   # Follow in real-time
```

This is implemented by teeing stdout/stderr when spawning the session process.

### Technology

Sessions are implemented using [Zellij](https://zellij.dev/). Zellij handles:
- Terminal multiplexing
- Attach/detach mechanics
- Process lifecycle within sessions

Headjack manages session metadata (name, type, timestamps) in the catalog.

## Catalog Schema Changes

Before:
```json
{
  "instances": [
    {
      "id": "...",
      "branch": "feat/auth",
      "container_id": "...",
      "status": "running"
    }
  ]
}
```

After:
```json
{
  "instances": [
    {
      "id": "...",
      "branch": "feat/auth",
      "container_id": "...",
      "status": "running",
      "sessions": [
        {
          "id": "sess-abc",
          "name": "happy-panda",
          "type": "claude",
          "zellij_session": "hjk-<instance-id>-sess-abc",
          "created_at": "2025-12-30T10:00:00Z",
          "last_accessed": "2025-12-30T14:30:00Z"
        }
      ]
    }
  ]
}
```

## Files Likely Affected

### Must Change

- `internal/cmd/` — Replace `new.go`, `resume.go`, `list.go` with `run.go`, `attach.go`, `ps.go`, `logs.go`, `kill.go`
- `internal/catalog/` — Add session tracking, `last_accessed` updates
- `internal/instance/` — Session CRUD operations, Zellij integration, log file management

### May Need Changes

- `internal/container/` — Ensure Zellij is available in containers
- `docs/designs/base-image.md` — Add Zellij to base image

### New Code Needed

- Zellij interaction layer (create session, attach, list, kill)
- Session name generator (word-based, Docker-style)
- Session logging layer (tee output to log files, read logs for `hjk logs`)

## Reference

See `docs/designs/cli-interface.md` for the complete CLI specification.
