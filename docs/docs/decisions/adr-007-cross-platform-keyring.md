---
sidebar_position: 7
title: "ADR-007: Cross-Platform Keyring"
description: Decision to use 99designs/keyring for cross-platform credential storage
---

# ADR 007: Cross-Platform Keyring

## Status

Accepted

## Context

Headjack stores agent credentials (Claude, Gemini, Codex OAuth tokens) securely and injects them into containers at session start. The original implementation used `github.com/keybase/go-keychain`, which only supports macOS Keychain.

As Headjack adds Linux support (via OCI runtimes like Docker and Podman), credential storage becomes a blocker. Linux has multiple options for secret storage:

1. **Secret Service API** (D-Bus): Used by GNOME Keyring and KDE Wallet. Works well on desktop Linux but requires a running D-Bus session.

2. **Linux kernel keyring** (`keyctl`): Built into the kernel, works on headless servers without D-Bus. Session-scoped by default.

3. **Encrypted file**: Universal fallback that works everywhere but requires password management.

4. **`pass`** (password-store): Uses GPG encryption. Requires GPG key setup.

We need a solution that:
- Maintains seamless macOS Keychain support
- Works on desktop Linux (GNOME/KDE)
- Works on headless Linux servers
- Minimizes code complexity

## Decision

Adopt `github.com/99designs/keyring` as the unified keyring library. This library:

- Supports 7 backends: macOS Keychain, Windows Credential Manager, Secret Service (D-Bus), KWallet, keyctl, pass, and encrypted file
- Is battle-tested (powers AWS Vault, 1.3k GitHub stars)
- Provides automatic backend selection per platform
- Includes encrypted file fallback for environments without native keyring support

### Backend Selection

| Platform | Priority Order |
|----------|----------------|
| macOS | `keychain` (native) |
| Linux | `secret-service` → `keyctl` → `file` |
| Windows | `wincred` |

Users can override via `HEADJACK_KEYRING_BACKEND` environment variable.

### Password Handling

For the encrypted file backend:
1. Check `HEADJACK_KEYRING_PASSWORD` environment variable
2. Fall back to interactive terminal prompt if available
3. Return clear error if no password source available

## Consequences

### Positive

- Enables Linux support without custom implementations for each keyring system
- Single library handles all platform complexity
- `keyctl` backend enables headless Linux server support
- Encrypted file fallback works in any environment (CI, containers, etc.)
- Windows support comes "for free" if ever needed

### Negative

- Larger dependency footprint (the library includes multiple backend implementations)
- File backend requires password management (env var or interactive prompt)
- `keyctl` credentials are session-scoped by default; may not persist across reboots

### Neutral

- API change: `New()` now returns `(Keychain, error)` instead of just `Keychain`
- Existing macOS users experience no change in behavior
