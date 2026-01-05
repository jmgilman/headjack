---
sidebar_position: 5
title: Manage Sessions
description: Start, monitor, and manage agent and shell sessions in isolated containers
---

# Manage Sessions

This guide covers the complete session lifecycle: starting agent and shell sessions, attaching and detaching, and monitoring background sessions.

## Create an instance first

Before starting any sessions, create an instance for your branch:

```bash
hjk run feat/auth
```

This creates a git worktree for the branch and a container with the worktree mounted. The instance is now ready for sessions.

## Start an agent session

Run an LLM coding agent (Claude, Gemini, or Codex) with a prompt:

```bash
hjk agent feat/auth claude --prompt "Implement JWT authentication"
```

The terminal attaches automatically to the agent session.

### Start without a prompt

Run an agent in interactive mode:

```bash
hjk agent fix/header-bug gemini
```

### Choose an agent

Specify the agent as the second argument:

```bash
hjk agent feat/api claude --prompt "Add rate limiting"
hjk agent feat/api gemini --prompt "Add rate limiting"
hjk agent feat/api codex --prompt "Add rate limiting"
```

## Start a shell session

Run an interactive shell using `hjk exec`:

```bash
hjk exec feat/auth
```

This opens a bash shell inside the container. Useful for manual debugging or running commands alongside agent sessions.

### Run a command

Execute a specific command:

```bash
hjk exec feat/auth npm test
hjk exec feat/auth npm run build
```

### Direct execution (no tmux)

For quick commands without session persistence, use `--no-mux`:

```bash
hjk exec feat/auth --no-mux ls -la
hjk exec feat/auth --no-mux pwd
```

This prints output directly to your terminal without creating a tmux session.

## Run in detached mode

Start any session in the background with `-d`:

```bash
hjk agent feat/auth claude -d --prompt "Refactor the auth module"
hjk exec feat/auth -d npm run build
```

Use `hjk logs` or `hjk attach` to monitor or interact later.

## Run multiple sessions in parallel

Start multiple agents on different branches, each in its own isolated container:

```bash
# Create instances first
hjk run feat/auth
hjk run feat/api
hjk run fix/header-bug

# Then start agents in detached mode
hjk agent feat/auth claude -d --prompt "Implement JWT authentication"
hjk agent feat/api claude -d --prompt "Add rate limiting to the API"
hjk agent fix/header-bug gemini -d --prompt "Fix the header rendering bug"
```

Monitor all running instances:

```bash
hjk ps
```

### Multiple sessions on one branch

Run multiple agents within a single instance using `--name`:

```bash
hjk agent feat/auth claude -d --name auth-impl --prompt "Implement the auth module"
hjk agent feat/auth claude -d --name auth-tests --prompt "Write tests for the auth module"
hjk exec feat/auth --name debug-shell  # add a shell session
```

All sessions share the same git worktree but run independently.

## Attach to sessions

### Attach to the most recent session

```bash
hjk attach
```

Headjack uses most-recently-used (MRU) selection.

### Attach by branch

```bash
hjk attach feat/auth
```

### Attach to a specific session

```bash
hjk attach feat/auth debug-shell
```

Find session names with `hjk ps <branch>`.

## Detach from sessions

When attached to a session, detach without terminating it:

1. Press `Ctrl+B`
2. Then press `d`

This returns you to your host terminal while the session continues running.

## Monitor with logs

View output from detached sessions without attaching.

### View recent output

```bash
hjk logs feat/auth happy-panda
```

Arguments are branch name and session name. Find session names with `hjk ps <branch>`.

### Follow logs in real-time

```bash
hjk logs feat/auth happy-panda -f
```

Press `Ctrl+C` to stop following.

### Show more history

```bash
hjk logs feat/auth happy-panda -n 500   # last 500 lines
hjk logs feat/auth happy-panda --full   # complete log
```

## Additional options

### Custom session name

```bash
hjk agent feat/auth claude --name jwt-implementation --prompt "Implement JWT"
hjk exec feat/auth --name build-session npm run build
```

### Custom container image

```bash
hjk run feat/auth --image my-registry.io/custom-image:latest
```

:::note
Using `--image` bypasses devcontainer detection. If your repository has a `devcontainer.json`, it will be used automatically without needing `--image`.
:::

## Troubleshooting

**"no sessions exist"** - No sessions are running. Start one with `hjk agent` or `hjk exec`.

**"no instance found for branch"** - The branch doesn't have an instance. Create one with `hjk run <branch>`.

## See also

- [Stop and Remove Instances](stop-cleanup.md) - clean up when finished
