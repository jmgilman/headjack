---
sidebar_position: 2
title: Authenticate Agents
description: How to set up authentication for Claude, Gemini, and Codex agents
---

# How to Authenticate Agents

Set up authentication so Headjack containers can use your AI subscriptions. All credentials are stored securely in macOS Keychain and automatically injected into containers at runtime.

## Prerequisites

- Headjack installed
- The CLI for your chosen agent installed:
  - Claude Code (`claude` command) with Claude Pro or Max subscription
  - Gemini CLI (`gemini` command) with Google AI Pro or Ultra subscription
  - OpenAI Codex CLI (`codex` command) with ChatGPT Plus, Pro, Team, or Enterprise subscription

## Claude

Run the authentication command:

```bash
hjk auth claude
```

A URL will be displayed. Open it in your browser, log in with your Anthropic account, and enter the displayed code back in the terminal when prompted.

## Gemini

Gemini requires authenticating with the Gemini CLI first:

```bash
gemini
```

Complete the Google OAuth login in your browser. Verify the credentials were created:

```bash
ls ~/.gemini/oauth_creds.json ~/.gemini/google_accounts.json
```

Then store the credentials in Headjack:

```bash
hjk auth gemini
```

## Codex

Run the authentication command:

```bash
hjk auth codex
```

A browser window opens to `localhost:1455` for ChatGPT OAuth. Log in and complete the flow, then return to the terminal.

## Verification

After authentication, verify by running an agent:

```bash
hjk run my-feature --agent claude   # or gemini, codex
```

The agent should authenticate without prompting for login.

## Notes

- Credentials use your subscription, not API billing
- Tokens are stored under the service `com.headjack.cli` in macOS Keychain
- Re-run the auth command if authentication fails or tokens expire
- For Gemini, re-run both `gemini` and `hjk auth gemini` to refresh credentials

## Related

- [Troubleshoot Authentication](./troubleshoot-auth)
