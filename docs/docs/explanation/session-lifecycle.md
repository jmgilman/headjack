---
sidebar_position: 5
title: Session Lifecycle
description: Session states, the MRU model, and persistence
---

# Session Lifecycle

Sessions are Headjack's mechanism for persistent, attachable terminal access to agents running inside containers. Understanding how sessions are created, persisted, and managed helps explain the interaction model and why certain commands behave the way they do.

## What Is a Session?

A session represents a persistent terminal process running inside a container. Technically, it's a tmux session that wraps either:

- An agent CLI process (`claude`, `gemini`, `codex`)
- A shell process (`/bin/bash`)

The session abstraction provides:

- **Persistence**: The process continues running when you disconnect
- **Reattachment**: You can reconnect and see everything that happened while away
- **Multiplexing**: Multiple sessions can run in the same instance
- **Output capture**: Session output is logged for later review

## Session Creation

When you run `hjk run feature-branch`, Headjack creates both an instance and an initial session:

```
+------------------------------------------------------------+
|                    hjk run                                 |
|                       |                                    |
|         +-------------+-------------+                      |
|         v             v             v                      |
|   Create worktree   Launch     Create session              |
|                    container        |                      |
|         |             |             |                      |
|         +-------------+-------------+                      |
|                       v                                    |
|              Instance running                              |
|              with one session                              |
+------------------------------------------------------------+
```

The session is created in "detached" mode, meaning tmux starts the session but doesn't attach your terminal. Then Headjack immediately attaches, giving the appearance of a single operation.

You can also create additional sessions in an existing instance:

```bash
# Create a shell session while a Claude session is running
hjk session create my-instance --type shell
```

This creates a second tmux session in the same container. Both sessions share the container's filesystem, network, and resources.

## Session Types

Sessions have types that determine their initial command:

| Type | Command | Use Case |
|------|---------|----------|
| `shell` | `/bin/bash` | Manual work, debugging, monitoring |
| `claude` | `claude` | Claude Code agent |
| `gemini` | `gemini` | Gemini CLI agent |
| `codex` | `codex` | OpenAI Codex agent |

The type is recorded in the catalog and displayed in listings:

```
INSTANCE    SESSION      TYPE     CREATED
abc123      main         claude   2 hours ago
abc123      debug        shell    10 minutes ago
```

Session type affects only the initial command. Once running, a session is just a terminal process.

## Attaching and Detaching

### Attach

When you attach to a session, Headjack:

1. Looks up the session's tmux session name
2. Configures the terminal for raw mode (so keystrokes pass through correctly)
3. Runs `tmux attach-session -t <session-name>`
4. Updates the session's `last_accessed` timestamp

Your terminal is now connected to the tmux session. You see the agent's output and can type input.

### Detach

To detach without terminating the session, press the tmux detach sequence:

```
Ctrl+B, then D
```

This returns you to your shell while the session continues running. The agent keeps working, generating output that accumulates in the terminal buffer.

### Exit

If you exit the session normally (type `exit`, Ctrl+D, or the agent terminates), the tmux session ends. Headjack detects this and removes the session from the catalog:

```go
// After attach returns, check if session still exists
sessions, err := m.mux.ListSessions(ctx)
// If session not found, clean up catalog entry
```

## The MRU Model

Headjack tracks when you last accessed each session. This enables the "most recently used" (MRU) model for quick attachment.

### Per-Instance MRU

Each `hjk attach` without a session name attaches to that instance's most recently used session:

```bash
# Create instance with claude session
hjk run feature-a

# Later: detach (Ctrl+B, D)

# Reattach to the same (MRU) session
hjk attach feature-a
```

### Global MRU

Running `hjk attach` with no arguments attaches to the globally most recently used session:

```bash
# Work on feature-a
hjk run feature-a
# Detach

# Work on feature-b
hjk run feature-b
# Detach

# Resume most recent work (feature-b)
hjk attach
```

This enables a rapid context-switching workflow:

1. Start multiple agents working on different branches
2. Check on each periodically with `hjk attach <branch>`
3. Resume the last one you were working on with just `hjk attach`

### Timestamp Updates

The `last_accessed` timestamp updates on:

- Session creation (initial timestamp)
- Attach (each time you connect)

It does not update during continuous attachment or on detach. This means "most recently used" reflects when you last started interacting, not when you stopped.

## Session Persistence

Sessions persist through:

- **Terminal disconnection**: Close your terminal; the session keeps running
- **SSH disconnection**: Working remotely? Disconnect; the session continues
- **Headjack restart**: The catalog tracks session metadata; tmux runs independently

Sessions do not persist through:

- **Container stop**: Stopping the container terminates all sessions inside it
- **Container restart**: Sessions must be recreated after container restart
- **Host reboot**: tmux sessions are terminated on system restart

## Output Logging

Each session's output is captured to a log file:

```
~/.local/share/headjack/logs/<instance-id>/<session-id>.log
```

This happens via tmux's `pipe-pane` feature:

```go
pipeArgs := []string{"pipe-pane", "-t", opts.Name, "cat >> " + escapedPath}
```

All output that appears in the terminal is also written to the log. This enables:

- Reviewing what an agent did while you were away
- Debugging issues after the fact
- Auditing agent behavior

Logs are removed when sessions are killed or instances are removed.

## Multiple Sessions per Instance

A single instance can have multiple concurrent sessions:

```
+-----------------------------------------------------------+
|                      Instance                             |
|  +-----------------------------------------------------+  |
|  |                    Container                        |  |
|  |                                                     |  |
|  |  +---------------+  +---------------+               |  |
|  |  |    Session    |  |    Session    |               |  |
|  |  |   (claude)    |  |   (shell)     |               |  |
|  |  |               |  |               |               |  |
|  |  |  Working on   |  |  Running      |               |  |
|  |  |  feature      |  |  tests        |               |  |
|  |  +---------------+  +---------------+               |  |
|  |                                                     |  |
|  |  Shared: /workspace, network, packages              |  |
|  +-----------------------------------------------------+  |
+-----------------------------------------------------------+
```

Common patterns:

- **Agent + shell**: Run an agent in one session, use another shell for manual intervention
- **Agent + monitoring**: Watch test output in one session while an agent codes in another
- **Multiple agents**: Run different agents (Claude, Gemini) on the same codebase for comparison

Sessions share the container's filesystem, so changes made by one session are visible to others immediately.

## Session Naming

Sessions can be named for easier reference:

```bash
# Auto-generated name (e.g., "happy-panda")
hjk session create my-instance

# Custom name
hjk session create my-instance --name testing
```

Names must be unique within an instance. The auto-generated names use a word list to create memorable combinations.

Named sessions can be attached by name:

```bash
hjk attach my-instance --session testing
```

## Session State Machine

```
                    create
                       |
                       v
              +---------------+
              |   Running     |<--------+
              +-------+-------+         |
                      |                 |
           +----------+----------+      |
           |          |          |      |
       detach    terminate    error    |
           |          |          |      |
           |          v          |      |
           |    +-----------+    |      |
           |    |  Removed  |    |      |
           |    |   from    |    |      |
           |    |  catalog  |    |      |
           |    +-----------+    |      |
           |                     |      |
           +--------attach-------+------+
                                 |
                                 v
                           +-----------+
                           |  Removed  |
                           |   from    |
                           |  catalog  |
                           +-----------+
```

Sessions exist only in "running" state from Headjack's perspective. Detachment doesn't change the state; the session continues running. Termination (normal exit or error) removes the session entirely.

## Related

- [Architecture Overview](./architecture) - How sessions fit into the instance model
- [Authentication](./authentication) - How credentials are injected into sessions
