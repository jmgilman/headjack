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

Runs the agent-specific authentication flow and stores credentials securely in the macOS Keychain. These credentials are automatically injected into containers when running agents with `hjk run --agent`.

## Subcommands

### hjk auth claude

Configure Claude Code authentication for use in Headjack containers.

```bash
hjk auth claude
```

This command runs the Claude setup-token flow which:

1. Displays a URL to open in your browser
2. Prompts you to log in with your Anthropic account
3. Presents a code to enter back in the terminal
4. Stores the resulting OAuth token securely in macOS Keychain

The stored token uses your Claude Pro/Max subscription rather than API billing.

### hjk auth gemini

Configure Gemini CLI authentication for use in Headjack containers.

```bash
hjk auth gemini
```

This command reads existing Gemini CLI credentials and stores them securely in the macOS Keychain. You must first authenticate with Gemini CLI by running `gemini` and completing the Google OAuth login flow.

The stored credentials use your Google AI Pro/Ultra subscription rather than API billing.

**Prerequisites**: Run `gemini` on your host machine first to complete the OAuth flow.

### hjk auth codex

Configure OpenAI Codex CLI authentication for use in Headjack containers.

```bash
hjk auth codex
```

This command runs the Codex login flow which:

1. Opens a browser to localhost:1455 for ChatGPT OAuth
2. Prompts you to log in with your ChatGPT account
3. Creates auth.json at `~/.codex/auth.json`
4. Stores the auth.json contents securely in macOS Keychain

The stored credentials use your ChatGPT Plus/Pro/Team/Enterprise subscription rather than API billing.

## Flags

### Inherited Flags

| Flag | Type | Description |
|------|------|-------------|
| `--multiplexer` | string | Terminal multiplexer to use (`tmux`, `zellij`) |

## Examples

```bash
# Set up Claude Code authentication
hjk auth claude

# Set up Gemini CLI authentication (after running 'gemini' first)
hjk auth gemini

# Set up Codex CLI authentication
hjk auth codex
```

## Security

All credentials are stored in the macOS Keychain, which provides:

- Encryption at rest
- Access control via macOS security policies
- Integration with Touch ID and Apple Watch unlock
- Automatic locking when the system sleeps

Credentials are never written to disk in plaintext.

## See Also

- [hjk run](run.md) - Use authenticated agents with `--agent` flag
