---
sidebar_position: 3
title: Environment Variables
description: Environment variables used by Headjack
---

# Environment Variables Reference

Headjack uses environment variables for configuration overrides and passes specific variables into container sessions.

## Configuration Overrides

These environment variables override values in the configuration file. They follow the pattern `HEADJACK_<SECTION>_<KEY>`.

| Variable | Type | Description | Overrides |
|----------|------|-------------|-----------|
| `HEADJACK_DEFAULT_AGENT` | string | Default agent for new instances | `default.agent` |
| `HEADJACK_BASE_IMAGE` | string | Default container image | `default.base_image` |
| `HEADJACK_MULTIPLEXER` | string | Terminal multiplexer | `default.multiplexer` |
| `HEADJACK_WORKTREE_DIR` | string | Worktree storage directory | `storage.worktrees` |

### Example Usage

```bash
# Use Claude as the default agent
export HEADJACK_DEFAULT_AGENT=claude

# Use a custom container image
export HEADJACK_BASE_IMAGE=myregistry.com/myimage:latest

# Override worktree directory
export HEADJACK_WORKTREE_DIR=/data/headjack/worktrees
```

## Host Environment Variables

These environment variables affect Headjack behavior on the host system.

| Variable | Type | Description |
|----------|------|-------------|
| `EDITOR` | string | Text editor for `hjk config --edit`. If not set, the edit command will fail. |

## Agent-Specific Environment Variables

Each agent can have environment variables configured in the configuration file under `agents.<agent>.env`. These variables are injected into the container when running agent sessions.

### Claude

Default environment variables for Claude sessions:

| Variable | Default | Description |
|----------|---------|-------------|
| `CLAUDE_CODE_MAX_TURNS` | `100` | Maximum number of conversation turns for Claude Code. |

### Gemini

No default environment variables are configured for Gemini sessions.

### Codex

No default environment variables are configured for Codex sessions.

### Configuring Agent Environment Variables

Agent environment variables can be configured in the configuration file:

```yaml
agents:
  claude:
    env:
      CLAUDE_CODE_MAX_TURNS: "200"
      CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: "1"
  gemini:
    env:
      GEMINI_API_KEY: "your-api-key"
  codex:
    env:
      OPENAI_API_KEY: "your-api-key"
```

## Container Environment

When Headjack starts a container, it sets up the environment to include:

1. Agent-specific environment variables from configuration
2. Any credentials configured via `hjk auth` commands
3. Standard container environment variables

The exact environment passed to containers depends on the agent type and authentication configuration.
