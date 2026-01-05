---
sidebar_position: 1
slug: /
title: Introduction
description: Spawn isolated LLM coding agents in container environments
---

# Headjack

Headjack is a macOS and Linux CLI tool for spawning isolated CLI-based LLM coding agents in container environments. Each agent runs in its own VM-isolated container with a dedicated git worktree, enabling safe parallel development across multiple branches.

## Why Headjack?

Running multiple AI coding agents simultaneously presents challenges: they can interfere with each other's work, create conflicting changes, and consume shared resources unpredictably. Headjack solves this by giving each agent its own isolated environment.

- **Isolation**: Each agent runs in a VM-isolated container with its own filesystem
- **Branch-based workflows**: Every instance is tied to a git branch via dedicated worktrees
- **Parallel development**: Run multiple agents on different features simultaneously
- **Supported agents**: Claude Code, Gemini CLI, and Codex CLI

## Key Concepts

### Instance

An **instance** is a running container environment tied to a specific git branch. When you create an instance with `hjk run`, Headjack:

1. Creates a git worktree for the branch
2. Spawns a container with the worktree mounted at `/workspace`
3. Tracks the instance in a local catalog

Instances persist across sessions and can be stopped, started, or removed as needed.

### Session

A **session** is a terminal multiplexer pane running inside an instance. Sessions are created with:

- `hjk agent` - Start an agent session (Claude, Gemini, or Codex)
- `hjk exec` - Start a shell session or run commands

Each instance can have multiple sessions, allowing you to run an agent alongside a shell for debugging or run multiple agents with different prompts.

Sessions can run in attached mode (interactive) or detached mode (background).

### Agent

An **agent** is one of the supported CLI-based LLM coding tools:

- `claude` - Claude Code from Anthropic
- `gemini` - Gemini CLI from Google
- `codex` - Codex CLI from OpenAI

Agents are authenticated via the `hjk auth` command before first use.

## Quick Example

```bash
# Create an instance for the feature branch
hjk run feat/auth

# Start Claude agent with a prompt
hjk agent feat/auth claude --prompt "Implement JWT authentication"

# Run another agent in the background on the same instance
hjk agent feat/auth claude -d --prompt "Write tests for the auth module"

# Start a shell session for debugging
hjk exec feat/auth

# List all running instances
hjk ps

# Attach to an existing session
hjk attach feat/auth
```

:::tip
Instance creation (`hjk run`) is separate from session management (`hjk agent`, `hjk exec`). This gives you flexibility to create instances without immediately starting sessions, and to choose between agent or shell sessions.
:::

## Workflow Overview

The typical Headjack workflow has three steps:

```bash
# Step 1: Create an instance
hjk run feat/auth

# Step 2: Start an agent or shell session
hjk agent feat/auth claude --prompt "Your task description"

# Step 3: Manage sessions
hjk attach         # Reattach to session
hjk logs feat/auth # View session output
hjk ps feat/auth   # List sessions
```

## Next Steps

- [Getting Started Tutorial](/tutorials/getting-started) - Set up Headjack and run your first agent
- [CLI Reference](/reference/cli/run) - Complete command documentation
