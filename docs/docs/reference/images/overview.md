---
sidebar_position: 1
title: Overview
description: Headjack container image variants
---

# Container Images Overview

Headjack provides pre-built container images for running isolated CLI-based LLM coding agents. All images are based on Ubuntu 24.04 LTS and include development tools, agent CLIs, and language runtime managers.

## Registry

All images are published to the GitHub Container Registry:

```
ghcr.io/gilmanlab/headjack
```

## Image Naming Convention

Images follow a consistent naming pattern:

```
ghcr.io/gilmanlab/headjack:<variant>
ghcr.io/gilmanlab/headjack:<variant>-<version>
```

Examples:
- `ghcr.io/gilmanlab/headjack:base` - Latest base image
- `ghcr.io/gilmanlab/headjack:base-v1.0.0` - Base image version 1.0.0
- `ghcr.io/gilmanlab/headjack:dind` - Latest Docker-in-Docker image

## Image Variants

The images form an inheritance hierarchy. Each variant builds on the previous one:

```
base --> systemd --> dind
```

### Comparison Table

| Feature | base | systemd | dind |
|---------|------|---------|------|
| Ubuntu 24.04 LTS | Yes | Yes | Yes |
| Agent CLIs (Claude, Gemini, Codex) | Yes | Yes | Yes |
| Version managers (pyenv, nodenv, goenv, rustup) | Yes | Yes | Yes |
| Development tools (git, gh, vim, ripgrep, etc.) | Yes | Yes | Yes |
| Terminal multiplexer (tmux) | Yes | Yes | Yes |
| systemd init system | No | Yes | Yes |
| Docker CE | No | No | Yes |
| Docker Compose plugin | No | No | Yes |
| Docker Buildx plugin | No | No | Yes |
| Multi-architecture support (amd64, arm64) | Yes | Yes | Yes |

### Image Sizes

| Variant | Approximate Size |
|---------|-----------------|
| `base` | ~600 MB |
| `systemd` | ~620 MB |
| `dind` | ~1.0 GB |

## Choosing an Image

### Use `base` when:
- Running simple agent workflows that do not require background services
- Minimizing image size is a priority
- No systemd or Docker functionality is needed

### Use `systemd` when:
- Your workflow requires running background services managed by systemd
- You need a proper init system for signal handling and process management
- You are running services that expect systemd to be available

### Use `dind` when:
- Your workflow requires building or running Docker containers
- You need to test containerized applications
- Your agent needs to execute Docker commands (e.g., `docker build`, `docker compose`)

## Pulling Images

```bash
# Pull the base image
docker pull ghcr.io/gilmanlab/headjack:base

# Pull the systemd image
docker pull ghcr.io/gilmanlab/headjack:systemd

# Pull the Docker-in-Docker image
docker pull ghcr.io/gilmanlab/headjack:dind
```

## Security

All images are:
- **Signed** with Cosign using keyless signing (Sigstore)
- **Attested** with SBOM (Software Bill of Materials) in SPDX format
- **Scanned** for vulnerabilities using Trivy

To verify image signatures:

```bash
cosign verify ghcr.io/gilmanlab/headjack:base \
  --certificate-identity-regexp='https://github.com/gilmanlab/headjack/.*' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com'
```

## Dockerfiles

For complete image specifications, see the Dockerfiles in the repository:

- [Base Dockerfile](https://github.com/GilmanLab/headjack/blob/master/images/base/Dockerfile)
- [Systemd Dockerfile](https://github.com/GilmanLab/headjack/blob/master/images/systemd/Dockerfile)
- [Docker-in-Docker Dockerfile](https://github.com/GilmanLab/headjack/blob/master/images/dind/Dockerfile)
