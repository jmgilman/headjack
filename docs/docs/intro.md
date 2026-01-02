---
sidebar_position: 1
slug: /
title: Introduction
description: Spawn isolated LLM coding agents in container environments
---

# Headjack

Headjack is a macOS CLI tool for spawning isolated CLI-based LLM coding agents in container environments. Each agent runs in its own VM-isolated container with a dedicated git worktree, enabling safe parallel development across multiple branches.

## Why Headjack?

Running multiple AI coding agents simultaneously presents challenges: they can interfere with each other's work, create conflicting changes, and consume shared resources unpredictably. Headjack solves this by giving each agent its own isolated environment.

- **Isolation**: Each agent runs in a VM-isolated container with its own filesystem
- **Branch-based workflows**: Every instance is tied to a git branch via dedicated worktrees
- **Parallel development**: Run multiple agents on different features simultaneously
- **Supported agents**: Claude Code, Gemini CLI, and Codex CLI

## Key Concepts

### Instance

An **instance** is a running container environment tied to a specific git branch. When you create an instance, Headjack:

1. Creates a git worktree for the branch
2. Spawns a container with the worktree mounted at `/workspace`
3. Tracks the instance in a local catalog

Instances persist across sessions and can be stopped, started, or removed as needed.

### Session

A **session** is a terminal multiplexer pane running inside an instance. Each instance can have multiple sessions, allowing you to run an agent alongside a shell for debugging or run multiple agents with different prompts.

Sessions can run in attached mode (interactive) or detached mode (background).

### Agent

An **agent** is one of the supported CLI-based LLM coding tools:

- `claude` - Claude Code from Anthropic
- `gemini` - Gemini CLI from Google
- `codex` - Codex CLI from OpenAI

Agents are authenticated via the `hjk auth` command before first use.

## Quick Example

```bash
# Start Claude on a feature branch with a prompt
hjk run feat/auth --agent claude "Implement JWT authentication"

# Run another agent in the background on the same branch
hjk run feat/auth --agent claude -d "Write tests for the auth module"

# List all running instances
hjk ls

# Attach to an existing session
hjk attach feat/auth
```

:::tip
The `hjk run` command is idempotent for instances: if an instance already exists for the branch, it creates a new session within it rather than a new instance.
:::

## Next Steps

- [Getting Started Tutorial](/tutorials/getting-started) - Set up Headjack and run your first agent
- [CLI Reference](/reference/cli/run) - Complete command documentation
