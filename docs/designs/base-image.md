# Design: Base Image

## Overview

This document specifies the default base image shipped with Headjack. The image provides a complete development environment suitable for running CLI-based coding agents.

## Image Details

| Property | Value |
|----------|-------|
| Registry | `ghcr.io/headjack/base` |
| Base OS | Ubuntu 24.04 LTS |
| Architecture | `linux/arm64` (Apple Silicon) |

## Pre-installed Components

### Operating System Configuration

- **systemd**: Enabled (required for multi-service environments)
- **iptables-legacy**: Configured as default (required for Docker-in-Docker)
- **Locales**: `en_US.UTF-8`
- **Timezone**: `UTC`

### Docker

Docker CE is pre-installed and configured:

- Docker daemon starts automatically via systemd
- iptables-legacy workaround applied (see ADR 002)
- User added to `docker` group for rootless access

### Agent CLIs

All three supported agent CLIs are pre-installed:

| Agent | Installation Method |
|-------|---------------------|
| **Claude Code** | `npm install -g @anthropic-ai/claude-code` |
| **Gemini CLI** | `npm install -g @anthropic-ai/gemini-cli` |
| **Codex CLI** | `npm install -g @openai/codex` |

Note: Installation methods may vary as these tools evolve. The image build process should use the official installation method for each.

### Common Tooling

**Version Control & Collaboration:**
- `git`
- `git-lfs`
- `gh` (GitHub CLI)

**Network & Data:**
- `curl`
- `wget`
- `jq`
- `yq`

**File & Text Processing:**
- `ripgrep` (`rg`)
- `fd-find` (`fd`)
- `fzf`
- `tree`
- `less`

**System Utilities:**
- `htop`
- `vim`
- `openssh-client`
- `zip`, `unzip`
- `make`
- `build-essential`

**Terminal Multiplexer:**
- `zellij` - Session management for CLI agents

### Language Version Managers

Rather than pre-installing specific language versions, the image includes version managers that allow users to install the versions they need:

| Language | Version Manager | Shell Integration |
|----------|-----------------|-------------------|
| **Python** | `pyenv` | Added to `.bashrc` |
| **Node.js** | `nodenv` | Added to `.bashrc` |
| **Go** | `goenv` | Added to `.bashrc` |
| **Rust** | `rustup` | Added to `.bashrc` |

Users can install specific versions inside their instances:

```bash
# Python
pyenv install 3.12.0
pyenv global 3.12.0

# Node.js
nodenv install 22.0.0
nodenv global 22.0.0

# Go
goenv install 1.22.0
goenv global 1.22.0

# Rust
rustup default stable
```

---

## Image Customization

Users who need additional tooling have two options (see ADR 006):

### Option 1: Use a Different Image

Specify an alternative image via the `--base` flag or configuration:

```bash
hjk new feat/auth --base my-registry.io/custom-image:latest
```

### Option 2: Extend the Base Image

Create a Dockerfile that inherits from the Headjack base:

```dockerfile
FROM ghcr.io/headjack/base:latest

# Add custom tooling
RUN apt-get update && apt-get install -y \
    postgresql-client \
    redis-tools

# Pre-install specific language versions
RUN pyenv install 3.11.0 && pyenv global 3.11.0
RUN nodenv install 20.0.0 && nodenv global 20.0.0
```

Then reference it:

```bash
hjk new feat/auth --base ./Dockerfile
```

---

## Environment Variables

The following environment variables are pre-configured:

| Variable | Value | Purpose |
|----------|-------|---------|
| `EDITOR` | `vim` | Default editor for CLI tools |
| `LANG` | `en_US.UTF-8` | Locale setting |
| `PATH` | Extended | Includes version manager shims |

Agent-specific environment variables (e.g., `CLAUDE_CODE_OAUTH_TOKEN`) are injected at runtime by Headjack, not baked into the image.

---

## Image Build

The base image is built using a Dockerfile in the Headjack repository:

```
/images/base/Dockerfile
```

### Build Process

1. CI builds the image on each release
2. Image is pushed to `ghcr.io/headjack/base:<version>` and `ghcr.io/headjack/base:latest`
3. SHA256 digest is recorded in release notes for verification

### Versioning

- Images are tagged with Headjack release versions (e.g., `v0.1.0`)
- `latest` always points to the most recent stable release
- Users can pin to specific versions in configuration:

```yaml
# config.yaml
default:
  base_image: ghcr.io/headjack/base:v0.1.0
```

---

## Size Considerations

Estimated image size breakdown:

| Component | Approximate Size |
|-----------|------------------|
| Ubuntu 24.04 base | ~75 MB |
| Docker CE | ~400 MB |
| Common tooling | ~150 MB |
| Version managers | ~50 MB |
| Agent CLIs + Node.js runtime | ~300 MB |
| **Total (compressed)** | **~1 GB** |

The image prioritizes functionality over size. Users requiring smaller images can build their own.

---

## Future Considerations

Out of scope for v1, but noted for future reference:

- **Multi-architecture support**: Add `linux/amd64` for non-Apple Silicon Macs
- **Slim variants**: Offer a minimal image without version managers
- **Pre-warmed language versions**: Variant images with common language versions pre-installed
