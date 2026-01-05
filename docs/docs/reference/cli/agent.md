---
sidebar_position: 2
title: hjk agent
description: Start an agent session in an existing instance
---

# hjk agent

Start an agent session (Claude, Gemini, or Codex) in an existing instance.

## Synopsis

```bash
hjk agent <branch> <agent_name> [prompt] [flags]
```

## Description

Creates a new session running the specified agent within an existing instance. The instance must already exist (created with `hjk run`).

This command:

1. Looks up the instance for the specified branch
2. Validates the agent name and authentication
3. Creates a new tmux session running the agent
4. Attaches to the session (unless `--detached` is specified)

If the instance is stopped, it is automatically restarted before creating the session.

All session output is captured to a log file regardless of attached/detached mode.

## Arguments

| Argument | Description |
|----------|-------------|
| `branch` | Git branch name of the instance (required) |
| `agent_name` | Agent to start: `claude`, `gemini`, or `codex` (required) |
| `prompt` | Instructions to pass to the agent (optional) |

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--name` | `-n` | string | | Override the auto-generated session name |
| `--detached` | `-d` | bool | `false` | Create session but do not attach (run in background) |

## Examples

```bash
# Start Claude agent on existing instance
hjk agent feat/auth claude

# Start Claude agent with a prompt
hjk agent feat/auth claude "Implement JWT authentication"

# Start Gemini agent with custom session name
hjk agent feat/auth gemini --name auth-session

# Start agent in detached mode (run in background)
hjk agent feat/auth claude -d "Refactor the auth module"

# Run multiple agents in parallel on the same instance
hjk agent feat/auth claude -d "Implement the login endpoint"
hjk agent feat/auth claude -d "Write tests for authentication"
```

## Authentication

Before using an agent, you must configure authentication:

```bash
# For Claude (Anthropic)
hjk auth claude

# For Gemini (Google)
hjk auth gemini

# For Codex (OpenAI)
hjk auth codex
```

Authentication tokens are securely stored in your system keychain and automatically injected into the container environment when starting an agent session.

## Workflow

The typical workflow separates instance creation from agent sessions:

```bash
# Step 1: Create the instance
hjk run feat/auth

# Step 2: Start an agent session
hjk agent feat/auth claude "Your prompt here"

# Step 3: Detach from session (Ctrl+B, d)

# Step 4: Later, reattach to the session
hjk attach feat/auth
```

## Error Handling

If no instance exists for the branch, you'll see an error with a helpful hint:

```
Error: no instance found for branch "feat/auth"
hint: run 'hjk run feat/auth' to create one
```

## See Also

- [hjk run](run.md) - Create an instance
- [hjk exec](exec.md) - Execute commands or start shell sessions
- [hjk attach](attach.md) - Attach to an existing session
- [hjk auth](auth.md) - Configure agent authentication
- [hjk logs](logs.md) - View session output
