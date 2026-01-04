---
sidebar_position: 1
title: Overview
description: Headjack container image
---

# Container Image Overview

Headjack provides a pre-built container image for running isolated CLI-based LLM coding agents. The image is based on Ubuntu 24.04 LTS and includes development tools, agent CLIs, and language runtime managers.

## Registry

The image is published to the GitHub Container Registry:

```
ghcr.io/gilmanlab/headjack
```

## Image Naming Convention

Images follow a consistent naming pattern:

```
ghcr.io/gilmanlab/headjack:base
ghcr.io/gilmanlab/headjack:base-<version>
```

Examples:
- `ghcr.io/gilmanlab/headjack:base` - Latest base image
- `ghcr.io/gilmanlab/headjack:base-v1.0.0` - Base image version 1.0.0

## Features

The base image includes:

| Feature | Included |
|---------|----------|
| Ubuntu 24.04 LTS | Yes |
| Agent CLIs (Claude, Gemini, Codex) | Yes |
| Version managers (pyenv, nodenv, goenv, rustup) | Yes |
| Development tools (git, gh, vim, ripgrep, etc.) | Yes |
| Terminal multiplexer (tmux) | Yes |
| Multi-architecture support (amd64, arm64) | Yes |

## Pulling the Image

```bash
docker pull ghcr.io/gilmanlab/headjack:base
```

## Security

The image is:
- **Signed** with Cosign using keyless signing (Sigstore)
- **Attested** with SBOM (Software Bill of Materials) in SPDX format
- **Scanned** for vulnerabilities using Trivy

To verify image signatures:

```bash
cosign verify ghcr.io/gilmanlab/headjack:base \
  --certificate-identity-regexp='https://github.com/gilmanlab/headjack/.*' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com'
```

## Dockerfile

For the complete image specification, see the Dockerfile in the repository:

- [Base Dockerfile](https://github.com/GilmanLab/headjack/blob/master/images/base/Dockerfile)
