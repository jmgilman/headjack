---
sidebar_position: 9
title: hjk auth
description: Configure agent authentication
---

# hjk auth

Configure authentication for agent CLIs.

## Synopsis

```bash
hjk auth <subcommand>
```

## Description

Configures agent authentication and stores credentials securely in the system keychain. These credentials are automatically injected into containers when running agents with `hjk agent`.

Each agent supports two authentication methods:

| Method | Description | Billing |
|--------|-------------|---------|
| **Subscription** | OAuth tokens from CLI tools | Uses your existing subscription (Claude Pro/Max, ChatGPT Plus/Pro, Gemini subscription) |
| **API Key** | Direct API keys | Pay-per-use API billing |

## Subcommands

### hjk auth claude

Configure Claude Code authentication for use in Headjack containers.

```bash
hjk auth claude
```

Prompts you to choose between:

1. **Subscription**: Uses your Claude Pro/Max subscription via OAuth token
2. **API Key**: Uses an Anthropic API key for pay-per-use billing

**Subscription flow**:

You must manually obtain the OAuth token:

1. Run `claude setup-token` in a separate terminal
2. Complete the browser login flow
3. Copy the token (starts with `sk-ant-`)
4. Paste it when prompted by `hjk auth claude`

**API Key flow**:

Enter your Anthropic API key directly (starts with `sk-ant-api`).

### hjk auth gemini

Configure Gemini CLI authentication for use in Headjack containers.

```bash
hjk auth gemini
```

Prompts you to choose between:

1. **Subscription**: Uses your Google AI subscription via OAuth credentials
2. **API Key**: Uses a Google AI API key for pay-per-use billing

**Subscription flow**:

Automatically reads existing credentials from `~/.gemini/` if available. If not found:

1. Run `gemini` on your host machine
2. Complete the Google OAuth login
3. Run `hjk auth gemini` again

**API Key flow**:

Enter your Google AI API key directly (starts with `AIza`).

### hjk auth codex

Configure OpenAI Codex CLI authentication for use in Headjack containers.

```bash
hjk auth codex
```

Prompts you to choose between:

1. **Subscription**: Uses your ChatGPT Plus/Pro/Team subscription via OAuth
2. **API Key**: Uses an OpenAI API key for pay-per-use billing

**Subscription flow**:

Automatically reads existing credentials from `~/.codex/auth.json` if available. If not found:

1. Run `codex login` on your host machine
2. Complete the OAuth flow in your browser
3. Run `hjk auth codex` again

**API Key flow**:

Enter your OpenAI API key directly (starts with `sk-`).

## Examples

```bash
# Set up Claude Code with subscription
hjk auth claude
# Select option 1, then paste your OAuth token

# Set up Claude Code with API key
hjk auth claude
# Select option 2, then enter your Anthropic API key

# Set up Gemini CLI (after running 'gemini' first)
hjk auth gemini

# Set up Codex CLI (after running 'codex login' first)
hjk auth codex
```

## Security

All credentials are stored in the system's secure credential store:

| Platform | Backend |
|----------|---------|
| macOS | Keychain |
| Linux (desktop) | GNOME Keyring / KDE Wallet |
| Linux (headless) | Kernel keyring or encrypted file |
| Windows | Credential Manager |

Security properties:

- Encryption at rest
- Access control via OS security policies
- Credentials never written to disk in plaintext

## Environment Variables

When injected into containers, credentials are set via environment variables:

| Agent | Subscription Env Var | API Key Env Var |
|-------|---------------------|-----------------|
| Claude | `CLAUDE_CODE_OAUTH_TOKEN` | `ANTHROPIC_API_KEY` |
| Gemini | `GEMINI_OAUTH_CREDS` | `GEMINI_API_KEY` |
| Codex | `CODEX_AUTH_JSON` | `OPENAI_API_KEY` |

## See Also

- [hjk agent](agent.md) - Start agent sessions using stored credentials
- [hjk run](run.md) - Create instances for running agents
- [Authentication](../../explanation/authentication.md) - How credential storage works
