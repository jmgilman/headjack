---
sidebar_position: 4
title: Building a Custom Image
description: Build and use a custom container image with your project's dependencies pre-installed
---

# Building a Custom Image

In this tutorial, we will build a custom container image with project-specific dependencies pre-installed. By the end, you will understand how to create images that give your agents immediate access to all the tools they need.

This tutorial takes approximately 30-40 minutes to complete.

## Prerequisites

Before starting, ensure you have:

- Completed the [Getting Started](./getting-started) tutorial
- Docker, Podman, or Apple Container installed on your machine
- Basic familiarity with Dockerfile syntax
- A project with specific runtime requirements (Python version, Node.js packages, system tools, etc.)

## Why Build a Custom Image

The default Headjack image includes version managers (pyenv, nodenv, goenv) but no pre-installed language versions. When an agent needs Python 3.11, it must install it first, which takes time.

A custom image lets you:

- Pre-install specific language versions your project requires
- Include system packages (database clients, build tools)
- Add global tooling (linters, formatters, CLI tools)
- Reduce agent startup time significantly

## Step 1: Identify Your Dependencies

First, catalog what your project needs. For this tutorial, we will build an image for a Python web application that requires:

- Python 3.11
- Poetry for dependency management
- PostgreSQL client for database access
- Node.js 20 for frontend build tools

Review your project's requirements and make a similar list.

## Step 2: Create the Dockerfile

Create a new file called `Dockerfile.headjack` in your project root:

```bash
cd ~/projects/my-app
touch Dockerfile.headjack
```

Open the file and add the base image:

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base
```

The Headjack base image provides:

- Ubuntu base with common development tools
- A `developer` user (non-root)
- pyenv, nodenv, and goenv pre-configured
- Claude Code, Gemini CLI, and Codex CLI installed

## Step 3: Install System Packages

System packages require root access. Add them first:

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

# Switch to root for system package installation
USER root

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Switch back to developer user
USER developer
WORKDIR /home/developer
```

The `--no-install-recommends` flag keeps the image smaller. Always clean up apt lists to reduce layer size.

## Step 4: Install Python

Now add Python using pyenv. This runs as the developer user:

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

USER developer
WORKDIR /home/developer

# Install Python 3.11
RUN eval "$(~/.pyenv/bin/pyenv init -)" && \
    pyenv install 3.11.7 && \
    pyenv global 3.11.7
```

The `eval` command initializes pyenv in the shell. This is necessary because the Dockerfile runs each `RUN` command in a fresh shell.

## Step 5: Install Python Tools

Add Poetry and any other Python tooling:

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

USER developer
WORKDIR /home/developer

# Install Python 3.11
RUN eval "$(~/.pyenv/bin/pyenv init -)" && \
    pyenv install 3.11.7 && \
    pyenv global 3.11.7

# Install Poetry
RUN eval "$(~/.pyenv/bin/pyenv init -)" && \
    pip install --upgrade pip && \
    pip install poetry
```

## Step 6: Install Node.js

Add Node.js for frontend tooling:

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

USER developer
WORKDIR /home/developer

# Install Python 3.11
RUN eval "$(~/.pyenv/bin/pyenv init -)" && \
    pyenv install 3.11.7 && \
    pyenv global 3.11.7

# Install Poetry
RUN eval "$(~/.pyenv/bin/pyenv init -)" && \
    pip install --upgrade pip && \
    pip install poetry

# Install Node.js 20
RUN eval "$(~/.nodenv/bin/nodenv init -)" && \
    nodenv install 20.10.0 && \
    nodenv global 20.10.0

# Install global Node.js tools
RUN npm install -g pnpm
```

## Step 7: Build the Image

Build the image with Docker:

```bash
docker build -t my-app-headjack:latest -f Dockerfile.headjack .
```

Or with Podman:

```bash
podman build -t my-app-headjack:latest -f Dockerfile.headjack .
```

:::note
Build with the same container runtime that Headjack uses. Check your configuration with `hjk config` and look for `runtime.name`. Images built with one runtime (Docker, Podman, or Apple Container) are not automatically available to others unless pushed to a registry.
:::

The build takes several minutes as it compiles Python and Node.js. You will see output for each step:

```
[+] Building 245.3s (12/12) FINISHED
 => [1/6] FROM ghcr.io/gilmanlab/headjack:base
 => [2/6] RUN apt-get update && apt-get install -y ...
 => [3/6] RUN eval "$(~/.pyenv/bin/pyenv init -)" && pyenv install 3.11.7 ...
 => [4/6] RUN eval "$(~/.pyenv/bin/pyenv init -)" && pip install poetry
 => [5/6] RUN eval "$(~/.nodenv/bin/nodenv init -)" && nodenv install 20.10.0 ...
 => [6/6] RUN npm install -g pnpm
```

## Step 8: Test the Image

Verify the image works by running a container:

```bash
docker run --rm -it my-app-headjack:latest bash
```

Inside the container, verify your tools:

```bash
python --version
# Python 3.11.7

poetry --version
# Poetry (version 1.x.x)

node --version
# v20.10.0

psql --version
# psql (PostgreSQL) 15.x
```

Type `exit` to leave the container.

## Step 9: Use the Image with Headjack

Now use your custom image with Headjack. Specify it with the `--base` flag:

```bash
hjk run feat/new-feature --base my-app-headjack:latest --agent claude "Add user authentication using PostgreSQL sessions"
```

The agent starts immediately with all dependencies available. No waiting for Python or Node.js installation.

## Step 10: Set as Default

To avoid specifying `--base` every time, set your image as the default:

```bash
hjk config default.base_image my-app-headjack:latest
```

Or use an environment variable:

```bash
export HEADJACK_BASE_IMAGE=my-app-headjack:latest
```

Add this to your shell profile (`.bashrc`, `.zshrc`) to make it permanent.

Now all `hjk run` commands use your custom image automatically:

```bash
hjk run feat/new-feature --agent claude "Add user authentication"
```

## Step 11: Share with Your Team

For team projects, publish the image to a registry.

**Push to Docker Hub:**

```bash
docker tag my-app-headjack:latest yourusername/my-app-headjack:latest
docker push yourusername/my-app-headjack:latest
```

**Push to GitHub Container Registry:**

```bash
docker tag my-app-headjack:latest ghcr.io/your-org/my-app-headjack:latest
docker push ghcr.io/your-org/my-app-headjack:latest
```

Team members can then use the shared image:

```bash
hjk config default.base_image ghcr.io/your-org/my-app-headjack:latest
```

## Step 12: Build for Multiple Architectures

If your team uses both Intel and Apple Silicon Macs, build a multi-architecture image:

```bash
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -t ghcr.io/your-org/my-app-headjack:latest \
    --push \
    -f Dockerfile.headjack .
```

This creates an image that works on both architectures. Docker, Podman, and Apple Container automatically pull the correct variant.

## Complete Dockerfile

Here is the complete Dockerfile from this tutorial:

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

# Install system packages as root
USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Switch to developer user for language installation
USER developer
WORKDIR /home/developer

# Install Python 3.11
RUN eval "$(~/.pyenv/bin/pyenv init -)" && \
    pyenv install 3.11.7 && \
    pyenv global 3.11.7

# Install Poetry
RUN eval "$(~/.pyenv/bin/pyenv init -)" && \
    pip install --upgrade pip && \
    pip install poetry

# Install Node.js 20
RUN eval "$(~/.nodenv/bin/nodenv init -)" && \
    nodenv install 20.10.0 && \
    nodenv global 20.10.0

# Install global Node.js tools
RUN npm install -g pnpm
```

## What We Learned

In this tutorial, we:

- Identified project dependencies for the custom image
- Created a Dockerfile extending the Headjack base image
- Installed system packages, Python, and Node.js
- Built and tested the image locally
- Configured Headjack to use the custom image by default
- Learned how to share images with a team

Custom images dramatically improve the agent experience. Instead of waiting for dependency installation, agents can start working immediately. For projects with complex dependency chains, this saves significant time per session.

## Next Steps

Now that you can build custom images, explore these resources:

**How-To Guides**
- [Build and Use Custom Images](../how-to/build-custom-image) - Additional image patterns and examples

**Reference**
- [Container Images Overview](../reference/images/overview) - Official image variants and their contents
- [Configuration Reference](../reference/configuration) - All configuration options including image settings

**Concepts**
- [Image Customization](../explanation/image-customization) - How Headjack uses containers and images
