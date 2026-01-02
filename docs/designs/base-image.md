# Design: Container Images

## Overview

Headjack ships three container image variants, each building on the previous to add functionality. This layered approach allows users to choose the minimal image for their needs.

## Image Variants

```
headjack:base  →  headjack:systemd  →  headjack:dind
   (minimal)        (+ init system)      (+ Docker)
```

| Variant | Tag | Size (approx) | Use Case |
|---------|-----|---------------|----------|
| **Base** | `:base` | ~600 MB | Most development work; no Docker or systemd needed |
| **systemd** | `:systemd` | ~610 MB | Multi-service environments needing init system |
| **DinD** | `:dind` | ~1 GB | Workflows requiring Docker-in-Docker |

### Default

The default image is `:base` (minimal). Users who need systemd or Docker must explicitly specify the variant:

```bash
# Use systemd variant
hjk run feat/auth --base ghcr.io/jmgilman/headjack:systemd

# Use Docker-in-Docker variant
hjk run feat/auth --base ghcr.io/jmgilman/headjack:dind
```

Or configure in `~/.config/headjack/config.yaml`:

```yaml
default:
  base_image: ghcr.io/jmgilman/headjack:dind
```

---

## Image Details

| Property | Value |
|----------|-------|
| Registry | `ghcr.io/{owner}/headjack` |
| Base OS | Ubuntu 24.04 LTS |
| Architectures | `linux/amd64`, `linux/arm64` |

---

## Runtime Configuration Labels

Headjack images use OCI labels to declare how they should be run. These labels are read by Headjack at runtime to configure the container appropriately.

| Label | Purpose | Default (if absent) |
|-------|---------|---------------------|
| `io.headjack.init` | Command to run as PID 1 | `sleep infinity` |
| `io.headjack.podman.flags` | Space-separated `key=value` pairs for Podman runtime | (none) |
| `io.headjack.apple.flags` | Space-separated `key=value` pairs for Apple Containerization | (none) |

Images can specify both `podman.flags` and `apple.flags` to support both runtimes. Each runtime only reads its own label.

### Label Usage by Variant

| Variant | `io.headjack.init` | `io.headjack.podman.flags` |
|---------|-------------------|---------------------------|
| `:base` | (not set, uses default) | (not set) |
| `:systemd` | `/lib/systemd/systemd` | `systemd=always` |
| `:dind` | (inherited from systemd) | (inherited from systemd) |

### How It Works

1. When creating a container, Headjack fetches image metadata from the registry
2. It extracts the `io.headjack.*` labels
3. The `init` value becomes the container's main process (keeping it alive)
4. The runtime-specific flags label (`podman.flags` or `apple.flags`) is parsed and merged with config flags (see below)
5. The merged flags are passed to the container runtime

This allows images with systemd to be configured automatically with the correct flags, while the base image uses a simple `sleep infinity` to keep the container running.

### Flag Format

Both `podman.flags` and `apple.flags` labels use a simple `key=value` format:

| Label Value | Resulting Flag |
|-------------|----------------|
| `systemd=always` | `--systemd=always` |
| `privileged=true` | `--privileged` |
| `privileged=false` | (omitted) |
| `volume=/a:/b volume=/c:/d` | `--volume=/a:/b --volume=/c:/d` |
| `privileged` | `--privileged` (bare key = true) |

### Config Flags

You can also specify runtime flags in your config file. Config flags take precedence over image label flags:

```yaml
# ~/.config/headjack/config.yaml
runtime:
  flags:
    memory: "4g"           # --memory=4g
    systemd: "always"      # --systemd=always
    privileged: true       # --privileged
    volume:                # Repeated flags become arrays
      - "/host/path:/container/path"
      - "/another:/mount"
```

**Merge behavior:** When both image labels and config specify the same flag, the config value wins. This allows users to override image defaults without modifying the image.

### Limitations

**Flag format:** All flags are rendered in `--key=value` format. Short-form flags (e.g., `-m 2g`) and flags requiring separate arguments (e.g., `--memory 2g`) are not supported. Use long-form equals syntax instead.

**Boolean false in labels:** Setting `key=false` in image labels will omit the flag entirely. This works correctly when omission is equivalent to disabling the feature, but cannot express explicit `--key=false` for flags that require it.

### Custom Images

When creating custom images, you can set these labels to control runtime behavior:

```dockerfile
FROM ghcr.io/jmgilman/headjack:base

# Custom init process
LABEL io.headjack.init="/usr/local/bin/my-init"

# Custom Podman flags (e.g., for GPU support)
LABEL io.headjack.podman.flags="device=nvidia.com/gpu=all"
```

---

## Variant Contents

### Base Image (`:base`)

The minimal image containing everything needed for CLI-based coding agents.

**Operating System:**
- Ubuntu 24.04 LTS
- Locales: `en_US.UTF-8`
- Timezone: `UTC`

**Agent CLIs:**

| Agent | Package |
|-------|---------|
| Claude Code | `@anthropic-ai/claude-code` |
| Gemini CLI | `@google/gemini-cli` |
| Codex CLI | `@openai/codex` |

**Common Tooling:**

- **Version Control**: git, git-lfs, gh (GitHub CLI)
- **Network & Data**: curl, wget, jq, yq
- **File & Text Processing**: ripgrep, fd-find, fzf, tree, less
- **System Utilities**: htop, tmux, vim, openssh-client, zip, unzip, make, build-essential
- **Terminal Multiplexer**: zellij

**Language Version Managers:**

| Language | Manager | Shell Integration |
|----------|---------|-------------------|
| Python | pyenv | Added to `.bashrc` |
| Node.js | nodenv | Added to `.bashrc` |
| Go | goenv | Added to `.bashrc` |
| Rust | rustup | Added to `.bashrc` |

**Non-root User:**
- Username: `developer` (UID 1000)
- Home: `/home/developer`

**Default Command:** `/bin/bash`

### systemd Image (`:systemd`)

Extends `:base` with systemd init system support.

**Adds:**
- systemd and systemd-sysv packages
- systemd configured for container use (unnecessary services removed)
- Cgroup volume mount

**Default Command:** `/lib/systemd/systemd`

**Use when:**
- Your application requires proper process supervision
- You need to run multiple services in the container
- Services require systemd unit files

### DinD Image (`:dind`)

Extends `:systemd` with Docker-in-Docker support.

**Adds:**
- Docker CE (docker-ce, docker-ce-cli, containerd.io)
- Docker Compose plugin
- Docker Buildx plugin
- iptables-legacy workaround (required for Apple Containerization Framework, see ADR-002)
- `developer` user added to `docker` group

**Default Command:** `/lib/systemd/systemd` (inherited)

**Use when:**
- Your workflow requires building Docker images
- You need to run Docker containers inside the agent container
- Testing CI/CD pipelines that use Docker

---

## Version Managers

Rather than pre-installing specific language versions, images include version managers:

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

### Option 1: Use a Different Variant

Specify the variant via `--base` or configuration:

```bash
hjk run feat/auth --base ghcr.io/jmgilman/headjack:systemd
```

### Option 2: Extend an Existing Variant

Create a Dockerfile that inherits from a Headjack image:

```dockerfile
FROM ghcr.io/jmgilman/headjack:base

# Add custom tooling
RUN apt-get update && apt-get install -y \
    postgresql-client \
    redis-tools

# Pre-install specific language versions
USER developer
RUN pyenv install 3.11.0 && pyenv global 3.11.0
```

Then reference it:

```bash
hjk run feat/auth --base ./Dockerfile
```

---

## Environment Variables

| Variable | Value | Purpose |
|----------|-------|---------|
| `EDITOR` | `vim` | Default editor for CLI tools |
| `LANG` | `en_US.UTF-8` | Locale setting |
| `PATH` | Extended | Includes version manager shims |

Agent-specific environment variables (e.g., `CLAUDE_CODE_OAUTH_TOKEN`) are injected at runtime by Headjack.

---

## Image Build

Images are built from Dockerfiles in the Headjack repository:

```
images/
├── base/
│   └── Dockerfile      # Base image
├── systemd/
│   └── Dockerfile      # Extends base
└── dind/
    └── Dockerfile      # Extends systemd
```

### Build Process

1. CI builds all three variants on each release (in sequence due to dependencies)
2. Images are pushed to `ghcr.io/{owner}/headjack:{variant}`
3. Each image is signed with Cosign and has an SBOM attached

### Local Build

```bash
just build-images      # Build all variants
just build-base        # Build base only
just lint-dockerfiles  # Lint all Dockerfiles
```

### Versioning

- `headjack:base` - Latest base variant
- `headjack:systemd` - Latest systemd variant
- `headjack:dind` - Latest dind variant
- `headjack:base-v1.0.0` - Pinned version

Users can pin to specific versions:

```yaml
# config.yaml
default:
  base_image: ghcr.io/jmgilman/headjack:base-v1.0.0
```

---

## Size Breakdown

| Variant | Components | Size |
|---------|------------|------|
| `:base` | Ubuntu + tools + agents + version managers | ~600 MB |
| `:systemd` | Base + systemd | ~610 MB |
| `:dind` | systemd + Docker CE | ~1 GB |
