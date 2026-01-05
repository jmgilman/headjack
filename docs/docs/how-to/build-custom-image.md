---
sidebar_position: 12
title: Build and Use Custom Images
description: Build a custom container image with project dependencies and use it with Headjack
---

# Build and Use Custom Images

Create a custom image with your project's dependencies pre-installed for faster container startup.

:::tip Prefer Devcontainers
If your repository has a `devcontainer.json`, Headjack uses it automatically. You only need a custom image when:
- Your repository doesn't have a devcontainer configuration
- You want to share a pre-built image across multiple repositories
- You need faster startup than devcontainer building provides
:::

## Build a custom image

### Prerequisites

- Docker or Podman installed
- Familiarity with Dockerfile syntax

### Create a Dockerfile

Start from the Headjack base image and add your dependencies:

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

# Switch to root to install system packages
USER root

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    redis-tools \
    && rm -rf /var/lib/apt/lists/*

# Switch back to developer user
USER developer
WORKDIR /home/developer

# Install Python version with pyenv
RUN eval "$(~/.pyenv/bin/pyenv init -)" && \
    pyenv install 3.12.0 && \
    pyenv global 3.12.0

# Install Node.js version with nodenv
RUN eval "$(~/.nodenv/bin/nodenv init -)" && \
    nodenv install 20.10.0 && \
    nodenv global 20.10.0

# Install Go version with goenv
RUN eval "$(~/.goenv/bin/goenv init -)" && \
    goenv install 1.22.0 && \
    goenv global 1.22.0

# Pre-install global tools
RUN npm install -g pnpm typescript
```

### Build the image

Build with Docker:

```bash
docker build -t my-custom-headjack:latest -f Dockerfile.headjack .
```

Or with Podman:

```bash
podman build -t my-custom-headjack:latest -f Dockerfile.headjack .
```

### Build for multiple architectures

For teams with both Intel and ARM Macs (using Docker buildx):

```bash
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -t my-registry.io/my-custom-headjack:latest \
    --push \
    -f Dockerfile.headjack .
```

### Publish to a registry

Push to your registry:

```bash
docker push my-registry.io/my-custom-headjack:latest
```

For GitHub Container Registry:

```bash
docker tag my-custom-headjack:latest ghcr.io/your-org/my-custom-headjack:latest
docker push ghcr.io/your-org/my-custom-headjack:latest
```

## Use a custom image

### Override for a single run

Use the `--image` flag:

```bash
hjk run feat/auth --image my-registry.io/my-custom-headjack:latest
```

Combine with `--agent`:

```bash
hjk run feat/auth --image my-registry.io/my-custom-headjack:latest --agent claude "Implement the feature"
```

### Set as permanent default

Update your configuration:

```bash
hjk config default.base_image my-registry.io/my-custom-headjack:latest
```

Or use an environment variable:

```bash
export HEADJACK_BASE_IMAGE=my-registry.io/my-custom-headjack:latest
hjk run feat/auth
```

## Example: Python project image

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

USER developer
WORKDIR /home/developer

# Install Python 3.11
RUN eval "$(~/.pyenv/bin/pyenv init -)" && \
    pyenv install 3.11.7 && \
    pyenv global 3.11.7

# Install poetry
RUN eval "$(~/.pyenv/bin/pyenv init -)" && \
    pip install poetry
```

## Example: Node.js project image

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

USER developer
WORKDIR /home/developer

# Install Node.js 20
RUN eval "$(~/.nodenv/bin/nodenv init -)" && \
    nodenv install 20.10.0 && \
    nodenv global 20.10.0

# Install pnpm and common tools
RUN npm install -g pnpm turbo
```

## See also

- [Container Images Overview](../reference/images/overview) - compare official image variants
- [Configuration Reference](../reference/configuration) - all configuration options
