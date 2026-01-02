---
sidebar_position: 6
title: Authentication
description: How Keychain storage and token injection work
---

# Authentication

Running CLI agents inside containers creates an authentication challenge: how do credentials get from your host machine into the isolated container environment? Headjack solves this through a combination of secure credential storage in macOS Keychain and just-in-time injection when sessions start.

## The Authentication Challenge

Each CLI agent has its own authentication mechanism:

| Agent | Auth Type | Credential |
|-------|-----------|------------|
| Claude Code | OAuth 2.0 | OIDC token (`sk-ant-*`) |
| Gemini CLI | Google OAuth | OAuth credentials + account info |
| Codex | OpenAI OAuth | Auth JSON file |

These credentials are typically stored in config files in the user's home directory:

```
~/.claude.json              # Claude Code
~/.gemini/oauth_creds.json  # Gemini CLI
~/.codex/auth.json          # Codex
```

Simply mounting these files into containers would work but creates problems:

- **Persistence**: Containers are ephemeral; credentials would be lost on container recreation
- **Security**: Credentials on disk are readable by any process; keychain provides better protection
- **Isolation**: Multiple containers might need different credentials (future multi-account support)

## The Keychain Solution

Headjack stores agent credentials in macOS Keychain, the system's secure credential store:

```
+-----------------------------------------------------------+
|                   macOS Keychain                          |
|  +-----------------------------------------------------+  |
|  | Service: com.headjack                               |  |
|  +-----------------------------------------------------+  |
|  | claude-oidc-token  |  sk-ant-oat01-xxxx...          |  |
|  | gemini-oauth-creds |  {"oauth_creds":...}           |  |
|  | codex-oauth-creds  |  {"api_key":...}               |  |
|  +-----------------------------------------------------+  |
+-----------------------------------------------------------+
```

Keychain provides:

- **Encryption at rest**: Credentials are encrypted on disk
- **Access control**: Only Headjack can read its credentials
- **OS integration**: Locked when screen locks, protected by system security
- **No plaintext files**: Credentials never written to disk in readable form

## Authentication Flow

The authentication flow has two phases: capture and injection.

### Phase 1: Credential Capture

Before using an agent, you must authenticate:

```bash
hjk auth claude   # Capture Claude credentials
hjk auth gemini   # Capture Gemini credentials
hjk auth codex    # Capture Codex credentials
```

Each agent has a unique capture process:

**Claude Code**

Claude uses `claude setup-token` which runs an interactive OAuth flow:

```go
// From claude.go
cmd := exec.CommandContext(ctx, "claude", "setup-token")
// Run with PTY for interactive OAuth
ptmx, err := pty.Start(cmd)
// Extract token from output
token := extractToken(outputBuf.String())
// Store in keychain
storage.Set(claudeAccountName, token)
```

You authenticate in your browser, then paste the token into the terminal.

**Gemini CLI**

Gemini credentials are captured from existing config files:

```go
// From gemini.go
oauthData, err := os.ReadFile("~/.gemini/oauth_creds.json")
accountsData, err := os.ReadFile("~/.gemini/google_accounts.json")
// Combine and store
config := &GeminiConfig{OAuthCreds: oauthData, GoogleAccounts: accountsData}
storage.Set(geminiAccountName, string(configJSON))
```

You must have already run `gemini` on your host and completed OAuth.

**Codex**

Codex uses `codex login` interactively:

```go
// From codex.go
cmd := exec.CommandContext(ctx, "codex", "login")
// Run with PTY for interactive login
ptmx, err := pty.Start(cmd)
// After completion, read the auth file
authData, err := os.ReadFile("~/.codex/auth.json")
// Store in keychain
storage.Set(codexAccountName, string(authData))
```

### Phase 2: Credential Injection

When a session starts, Headjack injects credentials into the container:

```
                Session Start
                     |
                     v
        +-------------------------+
        |  Read from Keychain     |
        +-------------------------+
                     |
                     v
        +-------------------------+
        |  Pass as env variable   |
        +-------------------------+
                     |
                     v
        +-------------------------+
        |  Container setup writes |
        |  to expected locations  |
        +-------------------------+
                     |
                     v
        +-------------------------+
        |  Agent CLI reads config |
        |  and authenticates      |
        +-------------------------+
```

The implementation differs by agent:

**Claude Code**

Claude is passed the token via environment variable:

```go
// Environment variable set at session start
env = append(env, "CLAUDE_CODE_OAUTH_TOKEN=" + token)
```

Claude Code reads this variable directly.

**Gemini CLI**

Gemini needs config files, so Headjack writes them at instance start:

```go
setupCmd := `mkdir -p ~/.gemini && \
echo "$GEMINI_OAUTH_CREDS" | jq -r '.oauth_creds' > ~/.gemini/oauth_creds.json && \
echo "$GEMINI_OAUTH_CREDS" | jq -r '.google_accounts' > ~/.gemini/google_accounts.json && \
echo '{"security":{"auth":{"selectedType":"oauth-personal"}}}' > ~/.gemini/settings.json`
```

The combined credentials are passed as `GEMINI_OAUTH_CREDS`, then split into the expected files.

**Codex**

Codex similarly needs a config file:

```go
setupCmd := `mkdir -p ~/.codex && echo "$CODEX_AUTH_JSON" > ~/.codex/auth.json`
```

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

macOS Keychain provides:

- Encryption with user-specific keys
- Access control lists (ACL)
- Integration with Touch ID / system authentication
- Automatic locking when system sleeps

### No Cross-Instance Leakage

Each session gets its own credential injection. Sessions in different instances don't share credential storage (inside the container).

## Limitations

### Per-Machine Authentication

Credentials are stored in the local machine's Keychain. If you use Headjack on multiple machines, you must `hjk auth` on each one.

### Single Account per Agent

Currently, Headjack stores one set of credentials per agent type. Multi-account support (using different credentials for different instances) is not yet implemented.

### Container Filesystem Persistence

Once credentials are written inside a container, they persist until the container is recreated. A `hjk recreate` is needed to rotate credentials if they change on the host.

### OAuth Token Expiry

OAuth tokens expire. When they do, you must re-run `hjk auth` to capture fresh tokens. The agents themselves may handle refresh automatically, but if the initial token is too old, authentication fails.

## Troubleshooting Auth Issues

### "Token not found"

The credential hasn't been captured:

```bash
hjk auth claude  # Capture Claude credentials
```

### "Authentication failed" inside container

The token may have expired:

```bash
hjk auth claude  # Re-capture fresh token
hjk recreate <instance>  # Recreate container with new credentials
```

### Claude onboarding prompt

Claude Code shows onboarding prompts if it doesn't find expected config:

```go
// Headjack creates this to skip onboarding
setupCmd := `mkdir -p ~/.claude && echo '{"hasCompletedOnboarding":true}' > ~/.claude.json`
```

If you see onboarding prompts, the setup command may have failed. Check container logs.

## Why Not SSH Agent Forwarding?

SSH agent forwarding is a common solution for credential access in containers. Headjack doesn't use it because:

1. **Different credential types**: Agent CLIs don't use SSH keys
2. **OAuth complexity**: OAuth tokens aren't compatible with SSH agent protocol
3. **VM boundary**: SSH agent sockets don't cross the hypervisor boundary easily

The environment variable + file-writing approach works reliably across the VM boundary that Apple Containerization creates.

## Related

- [Session Lifecycle](./session-lifecycle) - When credentials are injected
