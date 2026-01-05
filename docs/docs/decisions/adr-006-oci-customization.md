---
sidebar_position: 6
title: "ADR-006: OCI Images for Customization"
description: Decision to use OCI images for environment customization
---

# ADR 006: OCI Images for Environment Customization

## Status

Accepted

## Context

Headjack spawns agents in isolated containers. Users need the ability to customize the environment — installing tools, languages, and dependencies their projects require.

We evaluated several approaches to environment customization:

### Options Considered

**First-class Nix/devenv support**
- Declarative, reproducible environments
- Rich ecosystem of packages
- Potential for shared Nix store across instances

However, investigation revealed significant challenges:
- macOS bind mounts don't preserve the filesystem semantics Nix requires
- Shared Nix store across containers fails due to permission/ownership issues
- Additional complexity in the codebase for managing Nix configuration
- Two different mental models for users (OCI images + Nix)

**Package lists in configuration**
- Simple YAML list of apt packages to install
- Headjack generates derived Dockerfile automatically

This adds complexity:
- Must generate and manage Dockerfiles
- Must track cache state
- Error handling for failed package installs
- Another configuration surface to maintain

**Pure OCI image approach**
- Ship a default base image with opinionated tooling
- Users override with `--image <image>` or `--image <Dockerfile>`
- All customization delegated to standard OCI tooling

This approach:
- Leverages existing, well-understood technology
- Offloads customization entirely to the user
- Keeps Headjack's codebase simple
- Uses `container build` caching (already validated)

## Decision

Use **OCI images exclusively** for environment customization. No first-class support for Nix, devenv, or package lists.

The customization model is:

1. **Devcontainer (default)**: If a `devcontainer.json` exists, use it automatically
2. **Image override**: Users specify an alternative image via `--image <registry-image>`
3. **Dockerfile override**: Users specify a Dockerfile via `--image <path/to/Dockerfile>`
4. **Fallback**: If no devcontainer and no `--image`, use configured `default.base_image`

When a Dockerfile path is provided (detected by filename ending in `Dockerfile` or `Containerfile`), Headjack runs `container build` before launching the instance. Layer caching is handled by the container runtime.

## Consequences

### Positive

- **Simplicity**: Headjack's codebase remains focused on orchestration, not environment management
- **Flexibility**: Users have full control via standard Dockerfile primitives
- **No lock-in**: Standard OCI images work with any container tooling
- **Caching**: Delegated to `container build`, which handles layer caching correctly
- **Familiarity**: Dockerfiles are widely understood; no new concepts to learn

### Negative

- **User responsibility**: Users must write and maintain their own Dockerfiles for customization
- **No declarative packages**: Can't just list packages in a YAML file
- **No cross-instance sharing**: Each instance uses a full copy of the image (no Nix-style deduplication)

### Neutral

- Users who want Nix can install it in their custom Dockerfile — it's not blocked, just not first-class
- The base image can include common tools, reducing the need for customization in typical cases
