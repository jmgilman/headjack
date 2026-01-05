---
sidebar_position: 2
title: Authenticate Agents
description: How to set up authentication for Claude, Gemini, and Codex agents
---

# How to Authenticate Agents

Set up authentication so Headjack containers can use your AI subscriptions or API keys. All credentials are stored securely in the system keychain and automatically injected into containers at runtime.

## Prerequisites

- Headjack installed
- For **subscription** authentication: The CLI for your chosen agent installed and a valid subscription
- For **API key** authentication: An API key from the provider

## Authentication Methods

Each agent supports two authentication methods:

| Method | Billing | Best For |
|--------|---------|----------|
| **Subscription** | Uses your existing subscription (Claude Pro/Max, ChatGPT Plus/Pro, Gemini) | Users with active subscriptions |
| **API Key** | Pay-per-use API billing | Users without subscriptions or who prefer usage-based billing |

## Claude

Run the authentication command:

```bash
hjk auth claude
```

Choose your authentication method when prompted:

### Subscription (Claude Pro/Max)

1. Select option 1
2. In a **separate terminal**, run `claude setup-token`
3. Complete the browser login flow
4. Copy the token (starts with `sk-ant-`)
5. Paste it when prompted

### API Key

1. Select option 2
2. Enter your Anthropic API key (starts with `sk-ant-api`)

## Gemini

Run the authentication command:

```bash
hjk auth gemini
```

Choose your authentication method when prompted:

### Subscription (Google AI)

If you have existing Gemini CLI credentials (`~/.gemini/`), they are automatically detected.

If not found:

1. Run `gemini` in a separate terminal
2. Complete the Google OAuth login
3. Run `hjk auth gemini` again

### API Key

1. Select option 2
2. Enter your Google AI API key (starts with `AIza`)

## Codex

Run the authentication command:

```bash
hjk auth codex
```

Choose your authentication method when prompted:

### Subscription (ChatGPT Plus/Pro/Team)

If you have existing Codex CLI credentials (`~/.codex/auth.json`), they are automatically detected.

If not found:

1. Run `codex login` in a separate terminal
2. Complete the OAuth flow in your browser
3. Run `hjk auth codex` again

### API Key

1. Select option 2
2. Enter your OpenAI API key (starts with `sk-`)

## Verification

After authentication, verify by running an agent:

```bash
hjk run feat/test
hjk agent feat/test claude   # or gemini, codex
```

The agent should authenticate without prompting for login.

## Switching Authentication Methods

To switch between subscription and API key:

```bash
hjk auth claude   # Select the other option
hjk rm feat/test && hjk run feat/test   # Recreate instance with new credentials
hjk agent feat/test claude
```

## Notes

- Subscription credentials use your subscription, not API billing
- API key credentials are billed per-use through the provider's API
- Credentials are stored in the system keychain (macOS Keychain, GNOME Keyring, etc.)
- Re-run the auth command if authentication fails or tokens expire

## Related

- [Troubleshoot Authentication](./troubleshoot-auth)
