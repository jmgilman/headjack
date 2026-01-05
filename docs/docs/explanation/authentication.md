---
sidebar_position: 6
title: Authentication
description: How credential storage and token injection work
---

# Authentication

Running CLI agents inside containers creates an authentication challenge: how do credentials get from your host machine into the isolated container environment? Headjack solves this through a combination of secure credential storage in the system keychain and just-in-time injection when sessions start.

## The Authentication Challenge

Each CLI agent supports multiple authentication methods:

| Agent | Subscription Auth | API Key Auth |
|-------|------------------|--------------|
| Claude Code | OAuth token (`sk-ant-*`) | Anthropic API key (`sk-ant-api*`) |
| Gemini CLI | Google OAuth credentials | Google AI API key (`AIza*`) |
| Codex | OpenAI OAuth (ChatGPT Plus/Pro) | OpenAI API key (`sk-*`) |

**Subscription authentication** uses your existing CLI subscription (Claude Pro/Max, ChatGPT Plus/Pro, Gemini subscription) via OAuth tokens.

**API key authentication** uses pay-per-use API keys that you purchase separately from the subscription.

## The Keychain Solution

Headjack stores agent credentials in the system's secure credential store as JSON:

```
+-----------------------------------------------------------+
|                   System Keychain                         |
|  +-----------------------------------------------------+  |
|  | Service: com.headjack.cli                           |  |
|  +-----------------------------------------------------+  |
|  | claude-credential  |  {"type":"subscription",       |  |
|  |                    |   "value":"sk-ant-oat01-..."}  |  |
|  | gemini-credential  |  {"type":"apikey",             |  |
|  |                    |   "value":"AIza..."}           |  |
|  | codex-credential   |  {"type":"subscription",       |  |
|  |                    |   "value":"{...auth.json...}"}  |  |
|  +-----------------------------------------------------+  |
+-----------------------------------------------------------+
```

### Platform-Specific Backends

Headjack automatically selects the best available backend for your platform:

| Platform | Backend | Description |
|----------|---------|-------------|
| macOS | `keychain` | Native macOS Keychain |
| Linux (desktop) | `secret-service` | GNOME Keyring or KDE Wallet via D-Bus |
| Linux (headless) | `keyctl` | Linux kernel keyring |
| Linux (fallback) | `file` | Encrypted file storage |
| Windows | `wincred` | Windows Credential Manager |

You can override the backend with the `HEADJACK_KEYRING_BACKEND` environment variable:

```bash
export HEADJACK_KEYRING_BACKEND=file  # Force encrypted file backend
```

For the encrypted file backend, provide a password via environment variable or interactive prompt:

```bash
export HEADJACK_KEYRING_PASSWORD=your-password
```

### Security Properties

The keychain provides:

- **Encryption at rest**: Credentials are encrypted on disk
- **Access control**: Only Headjack can read its credentials
- **OS integration**: Protected by system security mechanisms
- **No plaintext files**: Credentials never written to disk in readable form

## Authentication Flow

The authentication flow has two phases: capture and injection.

### Phase 1: Credential Capture

Before using an agent, you must authenticate:

```bash
hjk auth claude   # Configure Claude credentials
hjk auth gemini   # Configure Gemini credentials
hjk auth codex    # Configure Codex credentials
```

Each command presents a choice between subscription and API key authentication:

```
$ hjk auth claude

Configure claude authentication

Authentication method:
  1. Subscription (uses CLAUDE_CODE_OAUTH_TOKEN)
  2. API Key (uses ANTHROPIC_API_KEY)
Enter choice (1-2):
```

The capture process differs by agent and authentication type:

**Claude Code (Subscription)**

Claude requires you to manually obtain an OAuth token:

```
$ hjk auth claude
...
Enter choice (1-2): 1

Claude subscription credentials must be entered manually.

To get your OAuth token:
  1. Run: claude setup-token
  2. Complete the browser login flow
  3. Copy the token (starts with sk-ant-)

Paste your credential:
```

You run `claude setup-token` in a separate terminal, complete the OAuth flow, then paste the token.

**Claude Code (API Key)**

For API key authentication, you enter your Anthropic API key directly:

```
$ hjk auth claude
...
Enter choice (1-2): 2

Enter your claude API key.

API key:
```

**Gemini CLI (Subscription)**

Gemini credentials are auto-detected from existing config files:

```
$ hjk auth gemini
...
Enter choice (1-2): 1

Found existing subscription credentials.

Credentials stored securely.
```

If not found, you'll see instructions:

```
Gemini credentials not found.

To authenticate with your Gemini subscription:
  1. Run: gemini
  2. Complete the Google OAuth login
  3. Run: hjk auth gemini

Paste your credential:
```

**Codex (Subscription)**

Codex credentials are similarly auto-detected from `~/.codex/auth.json`:

```
$ hjk auth codex
...
Enter choice (1-2): 1

Found existing subscription credentials.

Credentials stored securely.
```

If not found, you must run `codex login` first in a separate terminal.

### Phase 2: Credential Injection

When a session starts, Headjack injects credentials into the container:

```
                Session Start
                     |
                     v
        +-------------------------+
        |  Read from Keychain     |
        |  (type + value)         |
        +-------------------------+
                     |
                     v
        +-------------------------+
        |  Set env variable       |
        |  based on credential    |
        |  type                   |
        +-------------------------+
                     |
                     v
        +-------------------------+
        |  Container setup writes |
        |  files (subscription    |
        |  only, if needed)       |
        +-------------------------+
                     |
                     v
        +-------------------------+
        |  Agent CLI reads config |
        |  and authenticates      |
        +-------------------------+
```

The environment variable used depends on credential type:

| Agent | Subscription Env Var | API Key Env Var |
|-------|---------------------|-----------------|
| Claude | `CLAUDE_CODE_OAUTH_TOKEN` | `ANTHROPIC_API_KEY` |
| Gemini | `GEMINI_OAUTH_CREDS` | `GEMINI_API_KEY` |
| Codex | `CODEX_AUTH_JSON` | `OPENAI_API_KEY` |

**API Key Mode**

When using API key authentication, the credential is passed directly via environment variable. No file setup is needed inside the container.

**Subscription Mode**

Subscription authentication may require additional file setup in the container:

- **Claude**: Creates `~/.claude.json` to skip onboarding prompts
- **Gemini**: Splits the combined JSON into `~/.gemini/oauth_creds.json` and `~/.gemini/google_accounts.json`
- **Codex**: Writes `~/.codex/auth.json`

## Security Properties

This design provides several security properties:

### Credentials Never in Images

Container images never contain credentials. They're injected at runtime. This means:

- Images can be shared without credential exposure
- Pushing images to registries is safe
- Container filesystem snapshots don't contain credentials

### Minimal Exposure Window

Credentials are written to the container filesystem only when a session starts. They exist in memory during the session but aren't persisted in the container image.

### Keychain Protection

Platform keychains provide:

- Encryption with user-specific keys
- Access control (only Headjack can access its credentials)
- Integration with system authentication
- Protection by OS security mechanisms

### No Cross-Instance Leakage

Each session gets its own credential injection. Sessions in different instances don't share credential storage (inside the container).

## Limitations

### Per-Machine Authentication

Credentials are stored in the local machine's keychain. If you use Headjack on multiple machines, you must run `hjk auth` on each one.

### Single Account per Agent

Currently, Headjack stores one set of credentials per agent type. Multi-account support (using different credentials for different instances) is not yet implemented.

### Container Filesystem Persistence

Once credentials are written inside a container, they persist until the container is recreated. Use `hjk rm` followed by `hjk run` to recreate an instance if credentials change on the host.

### OAuth Token Expiry

OAuth tokens expire. When they do, you must re-run `hjk auth` to capture fresh tokens. The agents themselves may handle refresh automatically, but if the initial token is too old, authentication fails.

## Troubleshooting Auth Issues

### "Token not found" or "auth not configured"

The credential hasn't been captured:

```bash
hjk auth claude  # Configure Claude credentials
```

### "Authentication failed" inside container

The token may have expired:

```bash
hjk auth claude  # Re-capture fresh credential
hjk rm <instance> && hjk run <branch>  # Recreate instance with new credentials
```

### Claude onboarding prompt

Claude Code shows onboarding prompts if it doesn't find expected config. Headjack creates `~/.claude.json` automatically when using subscription authentication. If you see onboarding prompts, the setup command may have failed. Check container logs.

### Switching between subscription and API key

To switch authentication methods, simply run `hjk auth` again and select the other option:

```bash
hjk auth claude  # Select option 2 for API key
hjk rm <instance> && hjk run <branch>  # Recreate instance with new credential type
```

## Why Not SSH Agent Forwarding?

SSH agent forwarding is a common solution for credential access in containers. Headjack doesn't use it because:

1. **Different credential types**: Agent CLIs don't use SSH keys
2. **OAuth complexity**: OAuth tokens aren't compatible with SSH agent protocol
3. **VM boundary**: SSH agent sockets don't cross hypervisor/container boundaries easily

The environment variable + file-writing approach works reliably across container boundaries.

## Related

- [Session Lifecycle](./session-lifecycle) - When credentials are injected
