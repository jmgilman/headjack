# Headjack Base Image
#
# A complete development environment for CLI-based coding agents.
# See docs/designs/base-image.md for full specification.

ARG UBUNTU_VERSION=24.04
ARG UBUNTU_DIGEST=sha256:c35e29c9450151419d9448b0fd75374fec4fff364a27f176fb458d472dfc9e54
FROM ubuntu:${UBUNTU_VERSION}@${UBUNTU_DIGEST}

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG TARGETARCH
ARG YQ_VERSION=v4.50.1
ARG NODE_VERSION=22.21.1
ARG GH_CLI_VERSION=2.83.2
ARG GH_CLI_GPG_FINGERPRINT=2C6106201985B60E6C7AC87323F3D4EA75716059
ARG DOCKER_GPG_FINGERPRINT=9DC858229FC7DD38854AE2D88D81803C0EBFCD88
ARG PYENV_VERSION=v2.6.17
ARG NODENV_VERSION=v1.6.2
ARG NODE_BUILD_VERSION=v5.4.22
ARG GOENV_VERSION=2.2.34
ARG RUSTUP_VERSION=1.28.2
ARG CLAUDE_CODE_VERSION=2.0.76
ARG GEMINI_CLI_VERSION=0.22.5
ARG CODEX_CLI_VERSION=0.77.0
ARG ENABLE_DOCKER_GROUP=1
# Set ENABLE_DOCKER_GROUP=0 to avoid granting docker-group root-equivalent access.

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
    # systemd (required for multi-service environments)
    systemd \
    systemd-sysv \
    # iptables (required for Docker-in-Docker)
    iptables \
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
# iptables-legacy Workaround (ADR-002)
# Required for Docker-in-Docker in Apple Containerization Framework
# =============================================================================

RUN update-alternatives --set iptables /usr/sbin/iptables-legacy && \
    update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy

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
# =============================================================================

RUN keyring=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
    tmpdir=$(mktemp -d) && \
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg -o /tmp/githubcli-archive-keyring.gpg && \
    GNUPGHOME="${tmpdir}" gpg --batch --show-keys --with-colons /tmp/githubcli-archive-keyring.gpg | \
        awk -F: '/^fpr:/ {print $10; exit}' | grep -qx "${GH_CLI_GPG_FINGERPRINT}" && \
    gpg --dearmor -o "${keyring}" /tmp/githubcli-archive-keyring.gpg && \
    chmod go+r "${keyring}" && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=${keyring}] https://cli.github.com/packages stable main" > /etc/apt/sources.list.d/github-cli.list && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends gh="${GH_CLI_VERSION}" && \
    rm -rf /var/lib/apt/lists/* /tmp/githubcli-archive-keyring.gpg "${tmpdir}"

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
# Docker CE
# =============================================================================

# hadolint ignore=DL3008
RUN keyring=/usr/share/keyrings/docker-archive-keyring.gpg && \
    tmpdir=$(mktemp -d) && \
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /tmp/docker.gpg && \
    GNUPGHOME="${tmpdir}" gpg --batch --show-keys --with-colons /tmp/docker.gpg | \
        awk -F: '/^fpr:/ {print $10; exit}' | grep -qx "${DOCKER_GPG_FINGERPRINT}" && \
    gpg --dearmor -o "${keyring}" /tmp/docker.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=${keyring}] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    docker-ce \
    docker-ce-cli \
    containerd.io \
    docker-buildx-plugin \
    docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/* /tmp/docker.gpg "${tmpdir}"

# Enable Docker to start with systemd
RUN systemctl enable docker

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
    # Update npm to fix CVE-2025-64756 (glob command injection vulnerability)
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
    fi; \
    if [ "${ENABLE_DOCKER_GROUP}" = "1" ]; then \
        usermod -aG docker "$USERNAME"; \
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
# systemd Configuration
# =============================================================================

USER root

# Clean up systemd services that don't make sense in a container
RUN rm -f /lib/systemd/system/multi-user.target.wants/* \
    /etc/systemd/system/*.wants/* \
    /lib/systemd/system/local-fs.target.wants/* \
    /lib/systemd/system/sockets.target.wants/*udev* \
    /lib/systemd/system/sockets.target.wants/*initctl* \
    /lib/systemd/system/sysinit.target.wants/systemd-tmpfiles-setup* \
    /lib/systemd/system/systemd-update-utmp*

# Use systemd as init
STOPSIGNAL SIGRTMIN+3

# Volume for cgroups (required for systemd)
VOLUME ["/sys/fs/cgroup"]

# Default to the developer user
USER $USERNAME
WORKDIR /home/$USERNAME

# systemd as init system
CMD ["/lib/systemd/systemd"]
