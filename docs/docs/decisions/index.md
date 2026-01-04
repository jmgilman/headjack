---
sidebar_position: 0
title: Architecture Decision Records
description: Significant architectural decisions made during Headjack development
---

# Architecture Decision Records

This section documents significant architectural decisions made during Headjack's development. Each ADR captures the context, decision, and consequences of a particular choice.

## What is an ADR?

An Architecture Decision Record (ADR) is a document that captures an important architectural decision made along with its context and consequences. ADRs help future contributors understand why certain decisions were made.

## Current Decisions

| ADR | Title | Status |
|-----|-------|--------|
| [ADR-001](./adr-001-macos-only) | Initial macOS-Only Platform Support | Superseded |
| [ADR-002](./adr-002-apple-containerization) | Apple Containerization Framework | Superseded |
| [ADR-003](./adr-003-go-language) | Go as Implementation Language | Accepted |
| [ADR-004](./adr-004-cli-agents) | CLI-Based Agents over API-Based | Accepted |
| [ADR-005](./adr-005-no-gpg-support) | Defer GPG Commit Signing Support | Accepted |
| [ADR-006](./adr-006-oci-customization) | OCI Images for Environment Customization | Accepted |

## Key Themes

These decisions reflect several key themes in Headjack's design:

- **Simplicity over generality**: OCI images only, CLI agents only
- **Leverage existing ecosystems**: Docker/Podman, Go CLI patterns, standard OCI tooling
- **Defer complexity**: GPG support deferred, Nix support left to users
- **Optimize for the common case**: Subscription-based agents, opinionated base images
