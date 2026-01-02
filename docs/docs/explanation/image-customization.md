---
sidebar_position: 7
title: Image Customization
description: OCI images approach vs alternatives like Nix
---

# Image Customization

Headjack runs agents in containers, and those containers need the right tools installed. How do you customize the environment when your project needs specific languages, frameworks, or system packages? Headjack answers this with standard OCI images, delegating all customization to Docker/Podman tooling you already know.

## The Customization Problem

Different projects have different requirements:

- A Python data science project needs specific Python versions, numpy, pandas
- A Rust project needs the Rust toolchain with specific targets
- A monorepo might need Node.js, Python, Go, and Docker all in one environment
- A legacy project might need specific system library versions

The agent running in the container needs these tools available. Without them, the agent can't build the project, run tests, or do useful work.

## Approaches Considered

### First-Class Nix Support

[Nix](https://nixos.org/) offers declarative, reproducible environments. You describe what you need, and Nix provides it:

```nix
# shell.nix
{ pkgs ? import <nixpkgs> {} }:
pkgs.mkShell {
  buildInputs = [
    pkgs.python311
    pkgs.nodejs_20
    pkgs.rustc
  ];
}
```

This is appealing: declare dependencies once, get identical environments everywhere.

However, investigation revealed significant challenges:

**macOS bind mount limitations**: Nix relies on specific filesystem semantics that macOS doesn't provide through bind mounts. The Nix store's hardlink-based deduplication fails.

**Permission issues**: Sharing a Nix store across containers requires careful permission management. Each container's user might have different UIDs, breaking shared store access.

**Two mental models**: Supporting Nix alongside OCI images means users must understand both systems. Documentation, troubleshooting, and support complexity doubles.

Nix is powerful but didn't fit well with Headjack's container-centric architecture.

### Package Lists in Configuration

Another approach: let users specify packages in a simple YAML format:

```yaml
# headjack.yaml
packages:
  - python3.11
  - nodejs
  - ripgrep
```

Headjack would translate this to a Dockerfile and build automatically.

Problems with this approach:

**Incomplete abstraction**: Package names differ between distributions. Is it `python3.11` or `python3`? `ripgrep` or `rg`?

**Limited expressiveness**: What about packages that need configuration? Custom repositories? Post-install setup scripts?

**Build complexity**: Headjack would need to manage Dockerfile generation, caching, error handling, and multi-stage builds.

This approach trades Docker's well-understood complexity for a new, less capable abstraction.

### Pure OCI Images

The third option: don't abstract at all. Use standard OCI images directly.

Users customize environments by:

1. Using the default base image (works for many cases)
2. Specifying an alternative image from a registry
3. Providing a Dockerfile that Headjack builds

This is what Headjack implements.

## The Base Image

Headjack provides a default base image (`ghcr.io/gilmanlab/headjack:base`) with opinionated tooling:

**Agent CLIs**
- Claude Code
- Gemini CLI
- OpenAI Codex

**Development Tools**
- git, git-lfs
- ripgrep, fd, fzf
- jq, yq
- tmux, vim
- GitHub CLI (gh)

**Language Version Managers**
- pyenv (Python)
- nodenv (Node.js)
- goenv (Go)
- rustup (Rust)

This base image handles the majority of use cases. The agent CLIs are pre-installed and ready. Common development tools are available. Language version managers let agents install specific versions as needed.

## Customizing via Image Override

For cases where the base image isn't enough, specify an alternative:

```bash
hjk run feature-branch --image myregistry/custom-image:latest
```

The custom image must include the agent CLI you want to use (claude, gemini, or codex) plus any project-specific dependencies.

### Building Custom Images

The typical pattern is extending the base image:

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

# Install project-specific system packages
RUN apt-get update && apt-get install -y \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

# Install specific Python version
RUN pyenv install 3.11.9 && pyenv global 3.11.9

# Install specific Node version
RUN nodenv install 20.12.0 && nodenv global 20.12.0

# Pre-install common packages
RUN pip install poetry
RUN npm install -g pnpm
```

Build and push to your registry:

```bash
docker build -t myregistry/myproject:latest .
docker push myregistry/myproject:latest
```

Then use in Headjack:

```bash
hjk run feature-branch --image myregistry/myproject:latest
```

### Dockerfile Mode

For quick iterations, point to a local Dockerfile:

```bash
hjk run feature-branch --image ./Dockerfile.dev
```

Headjack builds the image before launching the container. This is convenient during development but slower than using a pre-built image.

## Why This Approach

### Simplicity

Headjack's codebase stays focused on orchestration. All customization complexity lives in standard OCI tooling that's already well-documented and widely understood.

### Flexibility

Dockerfiles are maximally flexible. Anything you can express in a Dockerfile works with Headjack. There's no restricted subset or abstraction layer limiting what's possible.

### No Lock-in

Custom images work anywhere OCI images work. The same image you build for Headjack runs in Docker, Kubernetes, CI systems, or other tools.

### Ecosystem Leverage

The Docker ecosystem has:
- Millions of pre-built images
- Layer caching for fast rebuilds
- Multi-platform support (amd64, arm64)
- Registry infrastructure
- Security scanning tools

Headjack inherits all of this by using standard images.

## Trade-offs

### User Responsibility

Users must write and maintain Dockerfiles for customization. This requires Docker knowledge that not all developers have.

### No Declarative Packages

There's no simple "list packages in YAML" option. Even basic customization requires a Dockerfile.

### No Cross-Instance Sharing

Each instance uses a full container image. There's no Nix-style deduplication where multiple instances share package installations.

### Build Time

Custom images must be built before use. For complex images, this can take minutes. Pre-building and pushing to a registry mitigates this.

## Image Variants

The base image comes in variants for different use cases:

| Variant | Features | Use Case |
|---------|----------|----------|
| `base` | Agent CLIs, version managers | Most development work |
| `systemd` | Adds systemd support | Projects requiring services |
| `dind` | Adds Docker-in-Docker | Testing Docker workflows |

Each variant extends the previous, adding capabilities at the cost of image size.

## Best Practices

### Start with Base

Try the default base image first. It handles many cases without customization. Use version managers to install specific language versions.

### Pin Versions

In production Dockerfiles, pin specific versions:

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base@sha256:abc123...
```

This ensures reproducibility and protects against upstream changes.

### Use Multi-Stage Builds

For complex images, multi-stage builds reduce final image size:

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base AS builder
# Compilation steps...

FROM ghcr.io/gilmanlab/headjack:base
COPY --from=builder /app/binary /usr/local/bin/
```

### Pre-Build for Teams

For team use, pre-build and push images to a shared registry. Don't make each developer build locally.

## Related

- [ADR-006: OCI Images for Customization](../decisions/adr-006-oci-customization) - The formal decision record
- [Version Managers](./version-managers) - Why pyenv/nodenv/goenv instead of system packages
