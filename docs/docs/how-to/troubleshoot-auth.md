---
sidebar_position: 15
title: Troubleshoot Authentication
description: How to troubleshoot authentication issues with Claude, Gemini, and Codex
---

# How to Troubleshoot Authentication Issues

Diagnose and resolve common authentication problems for Claude, Gemini, and Codex agents.

## "CLI not found" Errors

### Symptom

```
claude CLI not found in PATH: please install Claude Code first
codex CLI not found in PATH: please install OpenAI Codex CLI first
```

### Solution

Install the missing CLI tool:

- **Claude**: Install Claude Code from [claude.ai/code](https://claude.ai/code)
- **Gemini**: Install Gemini CLI with `npm install -g @anthropic-ai/gemini-cli`
- **Codex**: Install OpenAI Codex CLI with `npm install -g @openai/codex`

Verify the CLI is in your PATH:

```bash
which claude
which gemini
which codex
```

## "Credentials not found" Errors

### Symptom (Gemini)

```
gemini credentials not found: please run 'gemini' and complete the OAuth login first
google_accounts.json not found: please run 'gemini' and complete the OAuth login first
```

### Solution

Gemini requires you to authenticate with the CLI first:

```bash
gemini
```

Complete the Google OAuth flow in the browser, then re-run:

```bash
hjk auth gemini
```

### Symptom (Codex)

```
codex auth.json not found: login may have failed
codex auth.json is empty: login may have failed
```

### Solution

The Codex login flow did not complete successfully. Run authentication again:

```bash
hjk auth codex
```

Ensure you complete the OAuth flow in the browser before closing it.

## "No token received" Error

### Symptom (Claude)

```
no token received from claude setup-token
```

### Solution

The Claude authentication flow did not complete successfully. This can happen if:

- You cancelled the login flow
- The browser login timed out
- Network issues interrupted the flow

Run authentication again:

```bash
hjk auth claude
```

Complete all steps in the browser and enter the code when prompted.

## Token Expired or Invalid

### Symptom

The agent fails to authenticate when running, even though you previously ran `hjk auth`.

### Solution

Re-authenticate to get a fresh token:

```bash
# For Claude
hjk auth claude

# For Gemini
gemini  # Complete OAuth flow first
hjk auth gemini

# For Codex
hjk auth codex
```

## Keychain Access Denied

### Symptom

Authentication fails with a keychain or permission error.

### Solution

1. Open **Keychain Access** (in Applications > Utilities)

2. Search for `com.headjack.cli`

3. Delete any existing Headjack entries

4. Re-run authentication:

   ```bash
   hjk auth claude   # or gemini/codex
   ```

5. When prompted by macOS, allow Headjack to access the keychain

## Viewing Stored Credentials

To check if credentials are stored in the keychain:

1. Open **Keychain Access**

2. Search for `com.headjack.cli`

3. Look for entries labeled:
   - `Headjack - claude-oidc-token` (Claude)
   - `Headjack - gemini-oauth-creds` (Gemini)
   - `Headjack - codex-oauth-creds` (Codex)

## Clearing All Authentication

To remove all stored credentials and start fresh:

1. Open **Keychain Access**

2. Search for `com.headjack.cli`

3. Delete all matching entries

4. Re-authenticate each agent as needed

## Related

- [Authenticate Agents](./authenticate) - set up authentication for Claude, Gemini, or Codex
