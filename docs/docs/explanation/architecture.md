---
sidebar_position: 1
title: Architecture Overview
description: How Headjack connects instances, sessions, worktrees, and containers
---

# Architecture Overview

Headjack orchestrates isolated coding agents through a layered architecture that connects several key concepts: instances, sessions, worktrees, and containers. Understanding how these pieces fit together helps explain why Headjack works the way it does.

## The Core Abstraction: Instances

An **instance** is Headjack's central concept. It represents a complete, isolated working environment for a coding agent. Each instance combines:

- A **git worktree** that provides a dedicated copy of your repository
- A **container** that provides an isolated execution environment
- One or more **sessions** that provide persistent terminal access

When you run `hjk run feature-branch`, Headjack creates an instance by wiring these three components together. The instance remains linked to your repository and branch throughout its lifecycle.

import ThemedImage from '@theme/ThemedImage';
import useBaseUrl from '@docusaurus/useBaseUrl';

<ThemedImage
  alt="Instance Architecture"
  sources={{
    light: useBaseUrl('/img/architecture-instance.png'),
    dark: useBaseUrl('/img/architecture-instance-dark.png'),
  }}
/>

The instance abstraction serves several purposes:

1. **Lifecycle management**: Starting, stopping, and removing all components together
2. **Identity**: A consistent reference by branch name regardless of container or session changes
3. **Persistence**: Catalog storage tracks instance state across CLI invocations

## One Branch, One Instance

Headjack enforces a strict constraint: **one instance per branch per repository**. This design decision stems from git's own constraints and from the goal of enabling parallel work.

Git doesn't allow the same branch to be checked out in multiple worktrees simultaneously. Rather than fight this constraint, Headjack embraces it as a feature. The branch name becomes the instance identifier:

```bash
hjk attach feature-branch
```

This constraint also prevents confusion. If multiple instances could share a branch, commits from one agent might unexpectedly appear in another agent's working directory. The one-to-one mapping ensures each agent has a clear, isolated workspace.

## Sessions: Persistent Terminal Access

Within each instance, **sessions** provide persistent terminal access. A session is a tmux session that runs inside the container, allowing you to:

- Attach and detach without losing state
- Run multiple sessions in parallel (an agent and a shell)
- Resume work after disconnecting

Sessions come in types that determine their initial command:

| Type | Command | Purpose |
|------|---------|---------|
| `shell` | `/bin/bash` | General-purpose terminal |
| `claude` | `claude` | Claude Code agent |
| `gemini` | `gemini` | Gemini CLI agent |
| `codex` | `codex` | OpenAI Codex agent |

The session abstraction allows agents and shells to coexist. A typical workflow might involve one Claude session for autonomous coding and one shell session for manual intervention or monitoring.

## The Data Flow

When you run a command like `hjk run feature-branch`, here's how data flows through the system:

1. **Repository identification**: Headjack opens your git repository and computes a stable identifier from the remote URL or path

2. **Worktree creation**: A new worktree is created at `~/.local/share/headjack/git/<repo-id>/<branch>`, checking out the specified branch

3. **Container launch**: A container starts with the worktree mounted at `/workspace`. The container runs `sleep infinity` as its init process, keeping it alive indefinitely

4. **Session creation**: A tmux session starts inside the container, running the specified agent CLI (or shell)

5. **Catalog persistence**: The instance metadata is written to `~/.local/share/headjack/catalog.json`

6. **Terminal attachment**: Your terminal attaches to the tmux session

When you detach (Ctrl+B, D), the session continues running in the container. The agent keeps working. When you `hjk attach` later, you reconnect to that same session and see everything that happened while you were away.

## The Catalog

The **catalog** is Headjack's persistent database of instance state. It's a JSON file that tracks:

- Instance ID, repository, and branch
- Container ID
- Current status (running, stopped, error)
- Sessions and their last-accessed timestamps

The catalog enables Headjack to survive restarts. When you run `hjk list`, Headjack reads the catalog and then queries the container runtime for current status. This two-phase approach means the catalog can be slightly stale (a container might have crashed), but Headjack reconciles on read.

## Why This Architecture?

This architecture emerges from several design goals:

**Isolation first**: Each agent operates in its own container with its own worktree. Changes in one agent's environment never affect another. This is essential when running autonomous agents that might install packages, modify configs, or make breaking changes.

**Git as the source of truth**: The repository remains the authoritative source. Worktrees are derived views. Multiple agents can work on different branches of the same repository, and their changes converge through normal git workflows (pull requests, merges).

**Persistence without complexity**: tmux provides battle-tested session persistence without requiring Headjack to implement its own state machine. Containers provide persistent filesystems. The catalog just tracks the wiring.

**Familiar building blocks**: Developers already understand git, containers, and terminal multiplexers. Headjack composes these familiar tools rather than inventing new abstractions.

## Related

- [Worktree Strategy](./worktree-strategy) - Why git worktrees instead of clones
- [Session Lifecycle](./session-lifecycle) - How sessions persist and resume
