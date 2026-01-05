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
| `HEADJACK_BASE_IMAGE` | string | Fallback container image when no devcontainer exists | `default.base_image` |
| `HEADJACK_MULTIPLEXER` | string | Terminal multiplexer | `default.multiplexer` |
| `HEADJACK_WORKTREE_DIR` | string | Worktree storage directory | `storage.worktrees` |

### Example Usage

```bash
# Use Claude as the default agent
export HEADJACK_DEFAULT_AGENT=claude

# Set a fallback container image (used when no devcontainer.json exists)
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

## Credential Environment Variables

When running agent sessions, Headjack injects credential environment variables based on your authentication configuration. The variable used depends on the credential type (subscription or API key).

### Claude

| Variable | Credential Type | Description |
|----------|-----------------|-------------|
| `CLAUDE_CODE_OAUTH_TOKEN` | Subscription | OAuth token from Claude Pro/Max subscription |
| `ANTHROPIC_API_KEY` | API Key | Anthropic API key for pay-per-use billing |

### Gemini

| Variable | Credential Type | Description |
|----------|-----------------|-------------|
| `GEMINI_OAUTH_CREDS` | Subscription | Combined OAuth credentials JSON from `~/.gemini/` |
| `GEMINI_API_KEY` | API Key | Google AI API key for pay-per-use billing |

### Codex

| Variable | Credential Type | Description |
|----------|-----------------|-------------|
| `CODEX_AUTH_JSON` | Subscription | OAuth credentials JSON from `~/.codex/auth.json` |
| `OPENAI_API_KEY` | API Key | OpenAI API key for pay-per-use billing |

These variables are set automatically when you run `hjk run --agent <agent>`. You configure which credential type to use via `hjk auth <agent>`.

## Keyring Environment Variables

These environment variables configure the cross-platform keyring backend used for credential storage.

| Variable | Type | Description |
|----------|------|-------------|
| `HEADJACK_KEYRING_BACKEND` | string | Override the keyring backend. Options: `keychain` (macOS), `secret-service` (Linux desktop), `keyctl` (Linux kernel), `file` (encrypted file) |
| `HEADJACK_KEYRING_PASSWORD` | string | Password for the encrypted file backend. Required when using `file` backend without interactive prompt. |

### Example Usage

```bash
# Force encrypted file backend on Linux
export HEADJACK_KEYRING_BACKEND=file
export HEADJACK_KEYRING_PASSWORD=my-secure-password

# Use GNOME Keyring on Linux
export HEADJACK_KEYRING_BACKEND=secret-service
```

## Container Environment

When Headjack starts a container, it sets up the environment to include:

1. Agent-specific environment variables from configuration
2. Credential environment variables based on authentication type (see above)
3. Standard container environment variables

The exact environment passed to containers depends on the agent type and authentication configuration.
