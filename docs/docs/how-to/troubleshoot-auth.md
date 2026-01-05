---
sidebar_position: 15
title: Troubleshoot Authentication
description: How to troubleshoot authentication issues with Claude, Gemini, and Codex
---

# How to Troubleshoot Authentication Issues

Diagnose and resolve common authentication problems for Claude, Gemini, and Codex agents.

## "auth not configured" Error

### Symptom

```
claude auth not configured: run 'hjk auth claude' first
gemini auth not configured: run 'hjk auth gemini' first
codex auth not configured: run 'hjk auth codex' first
```

### Solution

Run the authentication command for the agent:

```bash
hjk auth claude   # or gemini, codex
```

Choose either subscription or API key authentication when prompted.

## "Credentials not found" Errors (Subscription)

### Symptom (Gemini)

```
Gemini credentials not found.

To authenticate with your Gemini subscription:
  1. Run: gemini
  2. Complete the Google OAuth login
  3. Run: hjk auth gemini
```

### Solution

Gemini subscription authentication requires existing CLI credentials:

```bash
gemini  # Complete the Google OAuth flow
hjk auth gemini  # Select option 1
```

### Symptom (Codex)

```
Codex credentials not found.

To authenticate with your ChatGPT subscription:
  1. Run: codex login
  2. Complete the OAuth flow in your browser
  3. Run: hjk auth codex
```

### Solution

Codex subscription authentication requires existing CLI credentials:

```bash
codex login  # Complete the OAuth flow
hjk auth codex  # Select option 1
```

## Invalid Token/Key Format

### Symptom

```
invalid Claude OAuth token: must start with 'sk-ant-'
invalid Anthropic API key: must start with 'sk-ant-api'
invalid Google AI API key: must start with 'AIza'
invalid OpenAI API key: must start with 'sk-'
```

### Solution

Ensure you're entering the correct credential type:

| Agent | Subscription Token | API Key |
|-------|-------------------|---------|
| Claude | Starts with `sk-ant-` | Starts with `sk-ant-api` |
| Gemini | JSON from `~/.gemini/` | Starts with `AIza` |
| Codex | JSON from `~/.codex/auth.json` | Starts with `sk-` |

If you selected the wrong option, run `hjk auth` again and choose the other option.

## Token Expired or Invalid

### Symptom

The agent fails to authenticate when running, even though you previously ran `hjk auth`.

### Solution

Re-authenticate to get fresh credentials:

```bash
# For Claude (subscription)
# Run 'claude setup-token' first, then:
hjk auth claude

# For Gemini (subscription)
gemini  # Complete OAuth flow first
hjk auth gemini

# For Codex (subscription)
codex login  # Complete OAuth flow first
hjk auth codex

# For any agent (API key)
hjk auth <agent>  # Select option 2 and enter your API key
```

After re-authenticating, remove and recreate your instance to apply the new credentials:

```bash
hjk rm my-feature
hjk run my-feature
```

## Keychain Access Issues

### Symptom

Authentication fails with a keychain or permission error.

### Solution (macOS)

1. Open **Keychain Access** (in Applications > Utilities)
2. Search for `com.headjack.cli`
3. Delete any existing Headjack entries
4. Re-run authentication:

   ```bash
   hjk auth claude   # or gemini/codex
   ```

5. When prompted by macOS, allow Headjack to access the keychain

### Solution (Linux)

If using the encrypted file backend, ensure the password is set:

```bash
export HEADJACK_KEYRING_PASSWORD=your-password
hjk auth claude
```

Or switch to a different backend:

```bash
export HEADJACK_KEYRING_BACKEND=secret-service  # For GNOME Keyring
hjk auth claude
```

## Viewing Stored Credentials

### macOS

1. Open **Keychain Access**
2. Search for `com.headjack.cli`
3. Look for entries labeled:
   - `claude-credential`
   - `gemini-credential`
   - `codex-credential`

### Linux

Credentials are stored in your configured keyring backend. The location depends on the backend:

- **secret-service**: GNOME Keyring or KDE Wallet
- **keyctl**: Linux kernel keyring
- **file**: `~/.config/headjack/keyring/`

## Clearing All Authentication

To remove all stored credentials and start fresh:

### macOS

1. Open **Keychain Access**
2. Search for `com.headjack.cli`
3. Delete all matching entries
4. Re-authenticate each agent as needed

### Linux

```bash
# If using file backend
rm -rf ~/.config/headjack/keyring/

# Then re-authenticate
hjk auth claude
hjk auth gemini
hjk auth codex
```

## Switching Between Subscription and API Key

To switch authentication methods:

```bash
hjk auth claude  # Select the other option when prompted
hjk rm my-feature && hjk run my-feature  # Recreate instance with new credentials
```

## Related

- [Authenticate Agents](./authenticate) - Set up authentication for Claude, Gemini, or Codex
- [Authentication Explanation](../explanation/authentication) - How credential storage works
