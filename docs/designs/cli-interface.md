# Design: CLI Interface

## Overview

This document specifies the command-line interface for Headjack. The CLI is the primary way users interact with the tool.

## Binary Name

The binary is named `headjack` with a short alias `hjk` for ergonomic use.

```
headjack <command> [args]
hjk <command> [args]
```

## Core Concepts

### Instances

An **instance** is the combination of:
- A git worktree (persistent)
- A container (ephemeral, can be stopped/recreated)

Each branch can have exactly one instance. The branch name is the primary identifier.

### Sessions

A **session** is a persistent, attachable/detachable process running within an instance. Sessions are implemented using [Zellij](https://zellij.dev/), a modern terminal multiplexer.

- An instance can have zero or more sessions
- Sessions can be shells, agent CLIs (Claude, Gemini, Codex), or other processes
- Users can attach/detach from sessions at will
- Multiple sessions can run concurrently within an instance

Session names are auto-generated (similar to Docker container names) but can be overridden with the `--name` flag.

---

## Command Structure

### `run`

Create a new session (and instance if needed), then attach.

```
hjk run <branch> [prompt] [flags]
```

**Arguments:**
- `branch` (required) — Branch name. If an instance doesn't exist, one is created. If the branch doesn't exist in git, it's created from HEAD.
- `prompt` (optional) — Initial prompt to pass to the agent (requires `--agent`)

**Flags:**
- `--agent <name>` — Start the specified agent instead of a shell
- `--name <session-name>` — Override auto-generated session name
- `--base <image>` — Override the default base image
- `-d, --detached` — Create session but don't attach (run in background)

**Behavior:**
1. If no instance exists for the branch:
   - Creates a git worktree at the configured location
   - Spawns a new container with the worktree mounted
2. Creates a new Zellij session within the instance
3. If `--agent` specified: starts the agent (with optional prompt)
4. Otherwise: starts the default shell
5. If `--detached`: returns immediately without attaching
6. Otherwise: attaches stdio to the user's terminal

All session output is captured to a log file regardless of attached/detached mode, enabling `hjk logs` to work.

**Examples:**
```bash
# New instance with shell session
hjk run feat/auth

# New instance with Claude agent
hjk run feat/auth --agent claude "Implement JWT authentication"

# Additional session in existing instance
hjk run feat/auth --agent gemini --name gemini-experiment

# Shell session with custom name
hjk run feat/auth --name debug-shell

# Detached sessions (run in background)
hjk run feat/auth --agent claude -d "Refactor the auth module"
hjk run feat/auth --agent claude -d "Write tests for auth module"
# Now two Claude sessions are running in parallel
```

---

### `attach`

Attach to an existing session.

```
hjk attach [branch] [session]
```

**Arguments:**
- `branch` (optional) — Branch name of the instance
- `session` (optional) — Session name to attach to

**Behavior:**

The command uses a "most recently accessed" (MRU) strategy:

| Arguments | Behavior |
|-----------|----------|
| None | Attach to the most recently accessed session across all instances |
| `branch` only | Attach to the most recently accessed session for that instance |
| `branch` and `session` | Attach to the specified session |

If no sessions exist for the resolved scope, the command errors with a message suggesting `hjk run`.

**Examples:**
```bash
# Attach to whatever you were last working on
hjk attach

# Attach to most recent session in feat/auth
hjk attach feat/auth

# Attach to specific session
hjk attach feat/auth claude-main
```

**Detaching:**

To detach from a session without terminating it, use the Zellij keybinding (default: `Ctrl+O, d`). This returns you to your host terminal while the session continues running.

---

### `ps`

List instances or sessions.

```
hjk ps [branch] [flags]
```

**Arguments:**
- `branch` (optional) — If provided, list sessions for this instance

**Flags:**
- `-a, --all` — List across all repositories (default: current repo only)

**Behavior:**
- No arguments: lists instances for the current repository
- With branch: lists sessions for that instance
- With `--all`: lists all instances across all repositories

**Output formats:**

Instance listing (`hjk ps`):
```
BRANCH              STATUS      SESSIONS    CREATED
feat/auth           running     2           2h ago
fix/login-bug       stopped     0           1d ago
main                running     1           3d ago
```

Session listing (`hjk ps feat/auth`):
```
SESSION             TYPE        STATUS      CREATED         ACCESSED
claude-main         claude      detached    2h ago          5m ago
debug-shell         shell       detached    1h ago          30m ago
```

---

### `logs`

View output from a session without attaching.

```
hjk logs <branch> <session> [flags]
```

**Arguments:**
- `branch` (required) — Branch name of the instance
- `session` (required) — Session name

**Flags:**
- `-f, --follow` — Follow log output in real-time (like `tail -f`)
- `-n, --lines <num>` — Number of lines to show (default: 100)
- `--full` — Show entire log from session start

**Behavior:**
- Reads from the session's log file without attaching to the Zellij session
- Useful for checking on detached agents without interrupting them
- Log files are stored at `~/.local/share/headjack/logs/<instance-id>/<session-id>.log`

**Examples:**
```bash
# View recent output
hjk logs feat/auth happy-panda

# Follow output in real-time
hjk logs feat/auth happy-panda -f

# Show last 500 lines
hjk logs feat/auth happy-panda -n 500

# Show entire log
hjk logs feat/auth happy-panda --full
```

---

### `kill`

Kill a specific session.

```
hjk kill <branch>/<session>
```

**Arguments:**
- `branch/session` (required) — The instance branch and session name, separated by `/`

**Behavior:**
- Terminates the Zellij session
- Removes the session from the catalog
- The instance and other sessions are unaffected

**Examples:**
```bash
hjk kill feat/auth/debug-shell
hjk kill main/claude-experiment
```

---

### `stop`

Stop an instance's container.

```
hjk stop <branch>
```

**Behavior:**
- Terminates all sessions in the instance
- Stops the container
- Worktree is preserved
- Instance can be resumed later with `run`

---

### `rm`

Remove an instance entirely.

```
hjk rm <branch> [flags]
```

**Flags:**
- `-f, --force` — Skip confirmation prompt

**Behavior:**
- Terminates all sessions
- Stops and deletes the container
- Deletes the git worktree
- Removes the instance from the catalog

**Warning:** This deletes uncommitted work in the worktree.

---

### `recreate`

Recreate an instance's container without losing worktree state.

```
hjk recreate <branch>
```

**Behavior:**
- Terminates all sessions
- Stops and deletes the existing container
- Creates a new container with the same worktree
- Useful when the container environment is corrupted or needs a fresh state

The worktree (and all git-tracked and untracked files) is preserved.

---

### `auth`

Configure authentication for an agent CLI.

```
hjk auth <agent>
```

**Arguments:**
- `agent` (required) — The agent to authenticate (`claude`, `gemini`, `codex`)

**Behavior:**
- Runs the agent-specific authentication flow
- Stores credentials securely in macOS Keychain
- For Claude Code: runs `claude setup-token`, stores the OAuth token

**Examples:**
```bash
hjk auth claude   # Set up Claude Code authentication
hjk auth gemini   # Set up Gemini CLI authentication
```

---

### `config`

View and modify configuration.

```
hjk config [flags]
hjk config <key>
hjk config <key> <value>
```

**Modes:**
- No arguments: display all configuration
- One argument: display value for key
- Two arguments: set value for key

**Flags:**
- `--edit` — Open config file in `$EDITOR`

**Examples:**
```bash
hjk config                        # Show all config
hjk config default.agent          # Show default agent
hjk config default.agent claude   # Set default agent
hjk config --edit                 # Open in editor
```

---

### `version`

Display version information.

```
hjk version
```

### `help`

Display help for any command.

```
hjk help [command]
hjk <command> --help
```

---

## Configuration

### Hierarchy

Configuration follows a tiered precedence model (highest to lowest):

1. **Command-line flags** — Explicit flags always win
2. **Environment variables** — `HEADJACK_*` prefix
3. **User config file** — `~/.config/headjack/config.yaml`

No project-level configuration. Headjack is a user tool; repositories remain oblivious to its existence.

### Config File Location

Following XDG Base Directory specification:

```
~/.config/headjack/config.yaml
```

### Config File Format

```yaml
# Default settings
default:
  agent: claude
  base_image: ghcr.io/headjack/base:latest

# Agent-specific settings
agents:
  claude:
    env:
      CLAUDE_CODE_MAX_TURNS: "100"
  gemini:
    env: {}
  codex:
    env: {}

# Storage locations (defaults shown)
storage:
  worktrees: ~/.local/share/headjack/git
  catalog: ~/.local/share/headjack/catalog.json
```

### Environment Variables

| Variable | Description | Config equivalent |
|----------|-------------|-------------------|
| `HEADJACK_DEFAULT_AGENT` | Default agent CLI | `default.agent` |
| `HEADJACK_BASE_IMAGE` | Default base image | `default.base_image` |
| `HEADJACK_WORKTREE_DIR` | Worktree storage location | `storage.worktrees` |

---

## Data Storage

### Directory Structure

```
~/.local/share/headjack/
├── git/
│   └── <repo-identifier>/
│       └── <branch-name>/        # Git worktrees
├── catalog.json                   # Instance & session catalog
└── logs/
    └── <instance-id>/
        └── <session-id>.log      # Session output logs
```

### Catalog Format

The catalog tracks instances and their sessions:

```json
{
  "instances": [
    {
      "id": "abc123",
      "repo": "/Users/josh/code/myproject",
      "repo_id": "myproject-a1b2c3",
      "branch": "feat/auth",
      "worktree": "~/.local/share/headjack/git/myproject-a1b2c3/feat/auth",
      "container_id": "container-xyz",
      "created_at": "2025-12-30T10:00:00Z",
      "status": "running",
      "sessions": [
        {
          "id": "sess-abc",
          "name": "claude-main",
          "type": "claude",
          "zellij_session": "hjk-abc123-sess-abc",
          "created_at": "2025-12-30T10:00:00Z",
          "last_accessed": "2025-12-30T14:30:00Z"
        },
        {
          "id": "sess-def",
          "name": "debug-shell",
          "type": "shell",
          "zellij_session": "hjk-abc123-sess-def",
          "created_at": "2025-12-30T11:00:00Z",
          "last_accessed": "2025-12-30T12:00:00Z"
        }
      ]
    }
  ]
}
```

### Session Naming

Session names are auto-generated using a word-based scheme similar to Docker container names (e.g., `happy-panda`, `clever-wolf`). Users can override this with the `--name` flag on `hjk run`.

---

## Behavioral Notes

### Branch Name as Identity

Each branch can have exactly one instance:
- One instance per branch (strict)
- Branch name is the primary identifier for instance commands
- Users wanting parallel experiments should create separate branches

### Session Lifecycle

- Sessions are managed by Zellij within the container
- Detaching preserves the session and its running process
- Sessions are terminated when:
  - Explicitly killed with `hjk kill`
  - The instance is stopped with `hjk stop`
  - The instance is removed with `hjk rm`
  - The container is recreated with `hjk recreate`

### Container Lifecycle

- Containers are ephemeral; worktrees are persistent
- Stopping a container terminates all sessions but preserves worktree state
- Recreating a container gives a fresh environment without losing work

### MRU (Most Recently Used) Tracking

The `last_accessed` timestamp for a session is updated when:
- A user attaches to the session via `hjk attach`
- A user creates and attaches to the session via `hjk run`

This enables the "attach to most recent" behavior for quick context-switching.

### Error Cases

| Situation | `run` behavior | `attach` behavior |
|-----------|----------------|-------------------|
| Branch has no instance | Create instance + session | Error: no sessions |
| Instance exists, no sessions | Create session | Error: no sessions |
| Instance exists, has sessions | Create new session | Attach to MRU session |
| Session name conflict | Error: name in use | N/A |
| Container stopped | Start container, create session | Start container, attach to session |
| Not in a git repository | Error | Error (unless using global MRU with `--all`) |

---

## Future Considerations

These are explicitly out of scope for v1 but noted for future reference:

- **`hjk exec <branch> <command>`** — Run a one-off command in an instance
- **`hjk snapshot <branch>`** — Create a container snapshot for faster recreation
- **Session sharing** — Allow multiple users to attach to the same session
- **Log rotation** — Automatic cleanup of old session logs
