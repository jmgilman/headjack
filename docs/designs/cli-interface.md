# Design: CLI Interface

## Overview

This document specifies the command-line interface for Headjack. The CLI is the primary way users interact with the tool.

## Binary Name

The binary is named `headjack` with a short alias `hjk` for ergonomic use.

```
headjack <command> [args]
hjk <command> [args]
```

## Command Structure

### Instance Management

#### `new`

Create a new instance from the current repository.

```
hjk new <branch> [prompt] [flags]
```

**Arguments:**
- `branch` (required) — Branch name for the worktree. Headjack handles creation intelligently:
  - If the branch already has a managed worktree: error (use `resume`)
  - If the branch exists in the repo: create worktree from existing branch
  - If the branch does not exist: create worktree with a new branch from HEAD
- `prompt` (optional) — Initial prompt to pass to the agent (requires `--agent`)

**Flags:**
- `--agent <name>` — Start the specified agent instead of dropping into a shell
- `--base <image>` — Override the default base image

**Behavior:**
1. Creates a git worktree at the configured location
2. Spawns a new container with the worktree mounted
3. If `--agent` specified: starts the agent (with optional prompt)
4. Otherwise: starts the default shell
5. Attaches stdio to the user's terminal

**Examples:**
```bash
# Drop into shell in new instance
hjk new feat/auth

# Start Claude Code with a prompt
hjk new feat/auth --agent claude "Implement JWT authentication"

# Start agent without initial prompt
hjk new feat/auth --agent claude
```

---

#### `resume`

Attach to an existing instance.

```
hjk resume <branch> [prompt] [flags]
```

**Arguments:**
- `branch` (required) — Branch name of the existing instance
- `prompt` (optional) — Initial prompt to pass to the agent (requires `--agent`)

**Flags:**
- `--agent <name>` — Start the specified agent instead of dropping into a shell

**Behavior:**
1. Locates the existing worktree for the branch
2. If no container is running: starts a new container for the worktree
3. If `--agent` specified: starts the agent (with optional prompt)
4. Otherwise: starts the default shell
5. Attaches stdio to the user's terminal

Multiple `resume` calls to the same instance are valid—this allows running multiple shell sessions or agents concurrently within a single instance.

**Examples:**
```bash
# Resume into shell
hjk resume feat/auth

# Resume and start an agent
hjk resume feat/auth --agent claude "Continue implementing the login flow"
```

---

#### `list`

List instances.

```
hjk list [flags]
```

**Flags:**
- `-a, --all` — List instances across all repositories (default: current repo only)

**Behavior:**
- When run inside a git repository: lists instances for that repo
- With `--all`: lists all managed instances across all repos
- Shows branch name, container status, and creation time

**Output format:**
```
BRANCH              STATUS      CREATED
feat/auth           running     2h ago
fix/login-bug       stopped     1d ago
main                running     3d ago
```

---

#### `stop`

Stop a running instance's container.

```
hjk stop <branch>
```

**Behavior:**
- Stops the container associated with the instance
- Worktree is preserved
- Instance can be resumed later with `resume`

---

#### `rm`

Remove an instance entirely.

```
hjk rm <branch> [flags]
```

**Flags:**
- `-f, --force` — Skip confirmation prompt

**Behavior:**
- Stops the container if running
- Deletes the container
- Deletes the git worktree
- Removes the instance from the catalog

**Warning:** This deletes uncommitted work in the worktree.

---

#### `recreate`

Recreate an instance's container without losing worktree state.

```
hjk recreate <branch>
```

**Behavior:**
- Stops and deletes the existing container
- Creates a new container with the same worktree
- Useful when the container environment is corrupted or needs a fresh state

The worktree (and all git-tracked and untracked files) is preserved.

---

### Authentication

#### `auth`

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

### Configuration

#### `config`

View and modify configuration.

```
hjk config [flags]
hjk config <key>
hjk config <key> <value>
```

**Subcommands/modes:**
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

### Utility

#### `version`

Display version information.

```
hjk version
```

#### `help`

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
# Default agent to use when --agent is specified without a value
default:
  agent: claude
  base_image: ghcr.io/headjack/base:latest

# Agent-specific settings
agents:
  claude:
    # Additional environment variables to pass
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
├── catalog.json                   # Instance catalog
└── logs/                          # Optional: agent logs
```

### Catalog Format

The catalog tracks the mapping between worktrees and containers:

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
      "status": "running"
    }
  ]
}
```

---

## Behavioral Notes

### Branch Name as Identity

Each branch can have exactly one instance. This invariant simplifies management and maps naturally to git workflows:

- One instance per branch (strict)
- Branch name is the primary identifier for all commands
- Users wanting parallel experiments should create separate branches

### Container Lifecycle

- Containers are ephemeral; worktrees are persistent
- Stopping a container preserves all worktree state
- Recreating a container gives a fresh environment without losing work
- Multiple terminal sessions can attach to the same instance

### Error Cases

| Situation | `new` behavior | `resume` behavior |
|-----------|----------------|-------------------|
| Branch has existing instance | Error: use `resume` | Attach to instance |
| Branch has no instance | Create new instance | Error: use `new` |
| Container stopped, worktree exists | N/A | Start new container |
| Not in a git repository | Error | Error |

---

## Future Considerations

These are explicitly out of scope for v1 but noted for future reference:

- **`hjk exec <branch> <command>`** — Run a one-off command in an instance
- **`hjk logs <branch>`** — View logs from an instance
- **`hjk snapshot <branch>`** — Create a container snapshot for faster recreation
- **Instance naming** — Allow user-defined names in addition to branch-based identity
