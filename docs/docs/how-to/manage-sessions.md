---
sidebar_position: 5
title: Manage Sessions
description: Start, monitor, and manage agent and shell sessions in isolated containers
---

# Manage Sessions

This guide covers the complete session lifecycle: starting agent and shell sessions, attaching and detaching, and monitoring background sessions.

## Start an agent session

Run an LLM coding agent (Claude, Gemini, or Codex) with a prompt:

```bash
hjk run feat/auth --agent claude "Implement JWT authentication"
```

This creates a git worktree for the branch, a container with the worktree mounted, and starts the agent with your prompt. The terminal attaches automatically.

### Start without a prompt

Run an agent in interactive mode:

```bash
hjk run fix/header-bug --agent gemini
```

### Choose an agent

Specify the agent with `--agent`:

```bash
hjk run feat/api --agent claude "Add rate limiting"
hjk run feat/api --agent gemini "Add rate limiting"
hjk run feat/api --agent codex "Add rate limiting"
```

## Start a shell session

Run an interactive shell without an agent:

```bash
hjk run feat/auth
```

This creates the same isolated environment but drops you into a shell instead of starting an agent. Useful for manual debugging or running commands alongside agent sessions.

## Run in detached mode

Start any session in the background with `-d`:

```bash
hjk run feat/auth --agent claude -d "Refactor the auth module"
hjk run feat/auth -d  # detached shell
```

Use `hjk logs` or `hjk attach` to monitor or interact later.

## Run multiple sessions in parallel

Start multiple agents on different branches, each in its own isolated container:

```bash
hjk run feat/auth --agent claude -d "Implement JWT authentication"
hjk run feat/api --agent claude -d "Add rate limiting to the API"
hjk run fix/header-bug --agent gemini -d "Fix the header rendering bug"
```

Monitor all running instances:

```bash
hjk ps
```

### Multiple sessions on one branch

Run multiple agents within a single instance using `--name`:

```bash
hjk run feat/auth --agent claude -d --name auth-impl "Implement the auth module"
hjk run feat/auth --agent claude -d --name auth-tests "Write tests for the auth module"
hjk run feat/auth --name debug-shell  # add a shell session
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
hjk run feat/auth --agent claude --name jwt-implementation "Implement JWT"
```

### Custom container image

```bash
hjk run feat/auth --agent claude --image my-registry.io/custom-image:latest
```

:::note
Using `--image` bypasses devcontainer detection. If your repository has a `devcontainer.json`, it will be used automatically without needing `--image`.
:::

## Troubleshooting

**"no sessions exist"** - No sessions are running. Start one with `hjk run`.

**"no instance found for branch"** - The branch doesn't have an instance. Create one with `hjk run <branch>`.

## See also

- [Stop and Remove Instances](stop-cleanup.md) - clean up when finished
