# Headjack Base Image
#
# A minimal development environment for CLI-based coding agents.
# This is the base variant with agent CLIs and language runtime managers.
# For systemd support, use the systemd variant.
# For Docker-in-Docker support, use the dind variant.
# See docs/designs/base-image.md for full specification.

ARG UBUNTU_VERSION=24.04
ARG UBUNTU_DIGEST=sha256:c35e29c9450151419d9448b0fd75374fec4fff364a27f176fb458d472dfc9e54
FROM ubuntu:${UBUNTU_VERSION}@${UBUNTU_DIGEST}

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG TARGETARCH
ARG YQ_VERSION=v4.50.1
ARG NODE_VERSION=22.21.1
ARG GH_CLI_VERSION=2.83.2
ARG PYENV_VERSION=v2.6.17
ARG NODENV_VERSION=v1.6.2
ARG NODE_BUILD_VERSION=v5.4.22
ARG GOENV_VERSION=2.2.34
ARG RUSTUP_VERSION=1.28.2
ARG ZELLIJ_VERSION=0.43.1
ARG CLAUDE_CODE_VERSION=2.0.76
ARG GEMINI_CLI_VERSION=0.22.5
ARG CODEX_CLI_VERSION=0.77.0

# Set locale and timezone
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8
ENV TZ=UTC

# Default editor
ENV EDITOR=vim

# =============================================================================
# Base System Setup
# =============================================================================

# hadolint ignore=DL3008
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    # Locale support
    locales \
    # Timezone
    tzdata \
    # SSL certificates
    ca-certificates \
    # Basic utilities needed for subsequent installations
    curl \
    wget \
    gnupg \
    lsb-release \
    software-properties-common \
    && rm -rf /var/lib/apt/lists/*

# Configure locale
RUN locale-gen en_US.UTF-8

# =============================================================================
# Common Tooling
# =============================================================================

# hadolint ignore=DL3008
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    # Version control
    git \
    git-lfs \
    # Network & data utilities
    jq \
    # File & text processing
    ripgrep \
    fd-find \
    fzf \
    tree \
    less \
    # System utilities
    htop \
    tmux \
    vim \
    openssh-client \
    zip \
    unzip \
    make \
    build-essential \
    # Additional build dependencies for version managers
    libssl-dev \
    zlib1g-dev \
    libbz2-dev \
    libreadline-dev \
    libsqlite3-dev \
    libncursesw5-dev \
    xz-utils \
    tk-dev \
    libxml2-dev \
    libxmlsec1-dev \
    libffi-dev \
    liblzma-dev \
    && rm -rf /var/lib/apt/lists/*

# =============================================================================
# GitHub CLI (gh)
# Install from GitHub releases to ensure we get the official pre-built binary
# with properly versioned dependencies (fixes CVE-2024-52308, CVE-2025-66564)
# =============================================================================

RUN : "${TARGETARCH:?TARGETARCH is required for gh install}" && \
    gh_archive="gh_${GH_CLI_VERSION}_linux_${TARGETARCH}.tar.gz" && \
    gh_dir="gh_${GH_CLI_VERSION}_linux_${TARGETARCH}" && \
    curl -fsSL "https://github.com/cli/cli/releases/download/v${GH_CLI_VERSION}/${gh_archive}" -o "/tmp/${gh_archive}" && \
    curl -fsSL "https://github.com/cli/cli/releases/download/v${GH_CLI_VERSION}/gh_${GH_CLI_VERSION}_checksums.txt" -o /tmp/gh_checksums.txt && \
    gh_sha256="$(sha256sum "/tmp/${gh_archive}" | awk '{print $1}')" && \
    grep "${gh_archive}" /tmp/gh_checksums.txt | grep -q "${gh_sha256}" && \
    tar -xzf "/tmp/${gh_archive}" -C /tmp && \
    install -m 0755 "/tmp/${gh_dir}/bin/gh" /usr/bin/gh && \
    rm -rf "/tmp/${gh_archive}" /tmp/gh_checksums.txt "/tmp/${gh_dir}"

# =============================================================================
# yq (YAML processor)
# =============================================================================

# TARGETARCH is automatically set by Docker buildx for multi-platform builds
RUN : "${TARGETARCH:?TARGETARCH is required for yq install}" && \
    yq_binary="yq_linux_${TARGETARCH}" && \
    curl -fsSL "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/${yq_binary}" -o "/tmp/${yq_binary}" && \
    curl -fsSL "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/checksums" -o /tmp/yq_checksums && \
    yq_sha256="$(sha256sum "/tmp/${yq_binary}" | awk '{print $1}')" && \
    grep -E "^${yq_binary}\\b" /tmp/yq_checksums | grep -q "${yq_sha256}" && \
    install -m 0755 "/tmp/${yq_binary}" /usr/local/bin/yq && \
    rm -f "/tmp/${yq_binary}" /tmp/yq_checksums

# =============================================================================
# Zellij (terminal multiplexer)
# =============================================================================

RUN : "${TARGETARCH:?TARGETARCH is required for Zellij install}" && \
    case "${TARGETARCH}" in \
        amd64) zellij_arch="x86_64" ;; \
        arm64) zellij_arch="aarch64" ;; \
        *) echo "Unsupported TARGETARCH=${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    zellij_archive="zellij-${zellij_arch}-unknown-linux-musl.tar.gz" && \
    zellij_checksums="zellij-${zellij_arch}-unknown-linux-musl.sha256sum" && \
    curl -fsSL "https://github.com/zellij-org/zellij/releases/download/v${ZELLIJ_VERSION}/${zellij_archive}" -o "/tmp/${zellij_archive}" && \
    curl -fsSL "https://github.com/zellij-org/zellij/releases/download/v${ZELLIJ_VERSION}/${zellij_checksums}" -o "/tmp/${zellij_checksums}" && \
    tar -xzf "/tmp/${zellij_archive}" -C /tmp && \
    expected_sha256="$(awk '{print $1}' "/tmp/${zellij_checksums}")" && \
    actual_sha256="$(sha256sum /tmp/zellij | awk '{print $1}')" && \
    [ "${expected_sha256}" = "${actual_sha256}" ] || { echo "Checksum mismatch"; exit 1; } && \
    install -m 0755 /tmp/zellij /usr/local/bin/zellij && \
    rm -f "/tmp/${zellij_archive}" "/tmp/${zellij_checksums}" /tmp/zellij

# =============================================================================
# Node.js (required for Agent CLIs)
# =============================================================================

WORKDIR /tmp
RUN : "${TARGETARCH:?TARGETARCH is required for Node.js install}" && \
    case "${TARGETARCH}" in \
        amd64) node_arch="x64" ;; \
        arm64) node_arch="arm64" ;; \
        *) echo "Unsupported TARGETARCH=${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    node_dist="node-v${NODE_VERSION}-linux-${node_arch}" && \
    curl -fsSLO "https://nodejs.org/dist/v${NODE_VERSION}/${node_dist}.tar.xz" && \
    curl -fsSLO "https://nodejs.org/dist/v${NODE_VERSION}/SHASUMS256.txt" && \
    grep " ${node_dist}.tar.xz$" SHASUMS256.txt | sha256sum -c - && \
    tar -xJf "${node_dist}.tar.xz" -C /usr/local --strip-components=1 && \
    rm -f "/tmp/${node_dist}.tar.xz" /tmp/SHASUMS256.txt && \
    npm install -g npm@latest
WORKDIR /

# =============================================================================
# Agent CLIs
# =============================================================================

RUN npm install -g \
    @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION} \
    @google/gemini-cli@${GEMINI_CLI_VERSION} \
    @openai/codex@${CODEX_CLI_VERSION}

# =============================================================================
# CVE-2025-64756 Fix: Replace vulnerable glob in all node_modules
# The glob package has a command injection vulnerability fixed in 10.5.0+
# This must run AFTER all npm packages are installed
# =============================================================================

RUN curl -fsSL "https://registry.npmjs.org/glob/-/glob-10.5.0.tgz" -o /tmp/glob-10.5.0.tgz && \
    find /usr/local/lib/node_modules -type d -name "glob" -path "*/node_modules/glob" | while read -r glob_dir; do \
        if [ -f "${glob_dir}/package.json" ]; then \
            version=$(grep -o '"version"[[:space:]]*:[[:space:]]*"[^"]*"' "${glob_dir}/package.json" | head -1 | grep -o '[0-9][0-9.]*'); \
            case "$version" in \
                10.[0-4].*|[0-9].*) \
                    echo "Patching glob at ${glob_dir} (version ${version})"; \
                    rm -rf "${glob_dir:?}"/*; \
                    tar -xzf /tmp/glob-10.5.0.tgz -C "${glob_dir}" --strip-components=1; \
                    ;; \
            esac; \
        fi; \
    done && \
    rm -f /tmp/glob-10.5.0.tgz

# =============================================================================
# Create non-root user
# =============================================================================

ARG USERNAME=developer
ARG USER_UID=1000
ARG USER_GID=1000

# Check if user/group with these IDs already exist, reuse or create as needed
RUN if getent passwd "$USER_UID" > /dev/null 2>&1; then \
        # User with this UID exists, get their name and rename if needed
        existing_user=$(getent passwd "$USER_UID" | cut -d: -f1); \
        if [ "$existing_user" != "$USERNAME" ]; then \
            usermod -l "$USERNAME" -d "/home/$USERNAME" -m "$existing_user"; \
        fi; \
    else \
        # Create group if needed
        getent group "$USER_GID" > /dev/null 2>&1 || groupadd --gid "$USER_GID" "$USERNAME"; \
        # Create user
        useradd --uid "$USER_UID" --gid "$USER_GID" -m "$USERNAME"; \
    fi

# =============================================================================
# Version Managers (installed as user)
# =============================================================================

USER $USERNAME
WORKDIR /home/$USERNAME

# pyenv
RUN git clone --depth 1 --branch "${PYENV_VERSION}" https://github.com/pyenv/pyenv.git ~/.pyenv

# nodenv
RUN git clone --depth 1 --branch "${NODENV_VERSION}" https://github.com/nodenv/nodenv.git ~/.nodenv && \
    git clone --depth 1 --branch "${NODE_BUILD_VERSION}" https://github.com/nodenv/node-build.git ~/.nodenv/plugins/node-build

# goenv
RUN git clone --depth 1 --branch "${GOENV_VERSION}" https://github.com/go-nv/goenv.git ~/.goenv

# rustup
RUN case "${TARGETARCH}" in \
        amd64) rustup_target="x86_64-unknown-linux-gnu" ;; \
        arm64) rustup_target="aarch64-unknown-linux-gnu" ;; \
        *) echo "Unsupported TARGETARCH=${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    curl -fsSL "https://static.rust-lang.org/rustup/archive/${RUSTUP_VERSION}/${rustup_target}/rustup-init" -o /tmp/rustup-init && \
    curl -fsSL "https://static.rust-lang.org/rustup/archive/${RUSTUP_VERSION}/${rustup_target}/rustup-init.sha256" -o /tmp/rustup-init.sha256 && \
    expected_sha256="$(cut -d ' ' -f1 /tmp/rustup-init.sha256)" && \
    echo "${expected_sha256}  /tmp/rustup-init" | sha256sum -c - && \
    chmod +x /tmp/rustup-init && \
    /tmp/rustup-init -y --no-modify-path && \
    rm -f /tmp/rustup-init /tmp/rustup-init.sha256

# =============================================================================
# Shell Configuration
# =============================================================================

# hadolint ignore=SC2016
RUN { \
    echo ''; \
    echo '# pyenv'; \
    echo 'export PYENV_ROOT="$HOME/.pyenv"'; \
    echo 'export PATH="$PYENV_ROOT/bin:$PATH"'; \
    echo 'eval "$(pyenv init -)"'; \
    echo ''; \
    echo '# nodenv'; \
    echo 'export PATH="$HOME/.nodenv/bin:$PATH"'; \
    echo 'eval "$(nodenv init -)"'; \
    echo ''; \
    echo '# goenv'; \
    echo 'export GOENV_ROOT="$HOME/.goenv"'; \
    echo 'export PATH="$GOENV_ROOT/bin:$PATH"'; \
    echo 'eval "$(goenv init -)"'; \
    echo ''; \
    echo '# rustup'; \
    echo 'source "$HOME/.cargo/env"'; \
    echo ''; \
    echo '# fd-find is installed as fdfind on Ubuntu'; \
    echo 'alias fd=fdfind'; \
    } >> ~/.bashrc

# =============================================================================
# Final Setup
# =============================================================================

WORKDIR /home/$USERNAME

# Default shell
CMD ["/bin/bash"]
