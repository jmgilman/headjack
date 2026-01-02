---
sidebar_position: 4
title: Storage
description: Data directories and catalog format
---

# Storage Reference

Headjack stores data in several locations on the filesystem. This reference documents the directory structure, file formats, and organization.

## Default Locations

| Purpose | Default Path | Configurable |
|---------|--------------|--------------|
| Configuration | `~/.config/headjack/config.yaml` | No |
| Worktrees | `~/.local/share/headjack/git/` | Yes (`storage.worktrees`) |
| Catalog | `~/.local/share/headjack/catalog.json` | Yes (`storage.catalog`) |
| Logs | `~/.local/share/headjack/logs/` | Yes (`storage.logs`) |

## Directory Structure

```
~/.config/headjack/
└── config.yaml              # Configuration file

~/.local/share/headjack/
├── catalog.json             # Instance catalog
├── git/                     # Worktree storage
│   └── <repo-id>/           # Per-repository directory
│       └── <branch>/        # Per-branch worktree
└── logs/                    # Session logs
    └── <instance-id>/       # Per-instance directory
        └── <session-id>.log # Per-session log file
```

## Worktree Organization

Worktrees are organized hierarchically by repository and branch.

### Path Format

```
<worktrees-dir>/<repo-id>/<sanitized-branch>
```

- `<worktrees-dir>`: Base worktree directory (default: `~/.local/share/headjack/git`)
- `<repo-id>`: Unique repository identifier in format `<repo-name>-<short-hash>` (e.g., `myproject-a1b2c3`)
- `<sanitized-branch>`: Branch name with special characters replaced

### Branch Name Sanitization

Branch names are sanitized for filesystem compatibility:

- Slashes (`/`) are replaced with dashes (`-`)
- Invalid characters are removed (only `a-zA-Z0-9-_` allowed)
- Leading and trailing dashes are trimmed

Examples:

| Original Branch | Sanitized Name |
|-----------------|----------------|
| `main` | `main` |
| `feature/login` | `feature-login` |
| `bugfix/issue-123` | `bugfix-issue-123` |
| `release/v1.0.0` | `release-v100` |

## Catalog Format

The catalog is a JSON file that tracks all Headjack instances.

### Schema

```json
{
  "version": 2,
  "entries": [
    {
      "id": "a1b2c3d4",
      "repo": "/path/to/repository",
      "repo_id": "myproject-a1b2c3",
      "branch": "feature/my-feature",
      "worktree": "/home/user/.local/share/headjack/git/myproject-a1b2c3/feature-my-feature",
      "container_id": "abc123def456",
      "created_at": "2024-01-15T10:30:00Z",
      "status": "running",
      "sessions": [
        {
          "id": "sess-1234",
          "name": "happy-panda",
          "type": "claude",
          "mux_session_id": "hjk-sess-1234",
          "created_at": "2024-01-15T10:30:00Z",
          "last_accessed": "2024-01-15T11:00:00Z"
        }
      ]
    }
  ]
}
```

### Entry Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique 8-character hex identifier for the instance |
| `repo` | string | Absolute path to the source repository |
| `repo_id` | string | Unique repository identifier (`<name>-<hash>`) |
| `branch` | string | Branch name (original, not sanitized) |
| `worktree` | string | Absolute path to the git worktree |
| `container_id` | string | Container ID (may be empty if not running) |
| `created_at` | string | ISO 8601 timestamp of instance creation |
| `status` | string | Instance status: `creating`, `running`, `stopped`, `error` |
| `sessions` | array | List of sessions within the instance |

### Session Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique session identifier |
| `name` | string | Human-readable name (e.g., "happy-panda") |
| `type` | string | Session type: `shell`, `claude`, `gemini`, `codex` |
| `mux_session_id` | string | Terminal multiplexer session identifier |
| `created_at` | string | ISO 8601 timestamp of session creation |
| `last_accessed` | string | ISO 8601 timestamp of last access (for MRU tracking) |

### Instance Status Values

| Status | Description |
|--------|-------------|
| `creating` | Instance is being created |
| `running` | Instance is running and accessible |
| `stopped` | Instance has been stopped |
| `error` | Instance encountered an error |

### Version History

| Version | Changes |
|---------|---------|
| 1 | Initial catalog format |
| 2 | Added `sessions` field to entries |

The catalog automatically migrates from older versions when loaded.

## Log Files

Session output is captured to log files for later review.

### Path Format

```
<logs-dir>/<instance-id>/<session-id>.log
```

- `<logs-dir>`: Base log directory (default: `~/.local/share/headjack/logs`)
- `<instance-id>`: Instance identifier from the catalog
- `<session-id>`: Session identifier

### Log File Format

Log files contain the raw output from the terminal multiplexer session, including ANSI escape codes for colors and formatting.

### Viewing Logs

Use the `hjk logs` command to view session logs:

```bash
# View logs for a session
hjk logs <branch> <session>

# Follow logs in real-time
hjk logs <branch> <session> -f

# View last 500 lines
hjk logs <branch> <session> -n 500

# View entire log from session start
hjk logs <branch> <session> --full
```

## File Locking

The catalog file uses file-level locking to prevent concurrent modification:

- Read operations acquire a shared lock
- Write operations acquire an exclusive lock
- Lock acquisition times out after 5 seconds

This ensures data integrity when multiple Headjack processes access the catalog simultaneously.

## Data Cleanup

When removing an instance with `hjk rm`:

1. The container is stopped and removed
2. The git worktree is removed
3. The catalog entry is deleted
4. Log files for the instance are removed

The worktree directory structure is preserved even after removing instances, but empty directories may remain.
