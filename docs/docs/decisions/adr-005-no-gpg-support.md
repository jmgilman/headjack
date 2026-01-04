---
sidebar_position: 5
title: "ADR-005: Defer GPG Commit Signing"
description: Decision to defer GPG commit signing support
---

# ADR 005: Defer GPG Commit Signing Support

## Status

Accepted

## Context

Developers commonly use GPG to sign git commits. When running agents in isolated container instances, the host's GPG keys and agent are not directly accessible.

We investigated two approaches to enable GPG signing from within containers:

### Option 1: GPG Agent Forwarding via TCP Bridge

GPG agent forwarding works by proxying the host's `gpg-agent` socket into the container. This requires mounting the socket into the container.

A TCP bridge approach using `socat` to bridge Unix socket → TCP on the host, then TCP → Unix socket in the container was validated empirically and works, including with hardware tokens (Yubikey).

**Complexity:**
- Requires `socat` installed on host
- Requires starting/stopping a TCP bridge process per instance
- Requires port allocation and management
- Requires `socat` and network setup inside each container
- Public keys must be exported and imported into container

### Option 2: Short-Lived GPG Subkeys

Create a temporary signing subkey for each container instance, export only the subkey, and import it into the container.

**Limitations:**
- Does not work with hardware tokens (private keys cannot be exported)
- GitHub/GitLab require re-uploading the public key to verify commits signed by new subkeys
- Key management complexity (creation, cleanup, revocation)

### User Impact Assessment

- Many developers do not sign commits
- Organizations that require signing often also require hardware tokens
- Agent-generated commits are semantically different from human commits—whether they should bear the human's cryptographic signature is debatable

## Decision

**Defer GPG signing support.** It will not be included in the initial release.

## Consequences

### Positive

- Reduced implementation complexity
- No additional host dependencies (`socat`)
- No TCP bridge process management
- Faster path to initial release
- Avoids making premature design decisions about the "right" approach

### Negative

- Users who require signed commits cannot use Headjack for those workflows
- Organizations with mandatory commit signing policies may not be able to adopt

### Neutral

- GPG support can be added in a future version if demand materializes
- The TCP bridge approach is documented and validated for future implementation
- Users can manually configure signing if they're willing to set up the bridge themselves
