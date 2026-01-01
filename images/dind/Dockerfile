# Headjack DinD (Docker-in-Docker) Image
#
# Extends the systemd image with Docker CE for Docker-in-Docker support.
# Use this when your workflows require Docker commands inside the container.
# See docs/designs/base-image.md for full specification.

ARG BASE_IMAGE=ghcr.io/jmgilman/headjack:systemd
FROM ${BASE_IMAGE}

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG DOCKER_GPG_FINGERPRINT=9DC858229FC7DD38854AE2D88D81803C0EBFCD88
ARG USERNAME=developer

# =============================================================================
# iptables-legacy Workaround (ADR-002)
# Required for Docker-in-Docker in Apple Containerization Framework
# =============================================================================

# hadolint ignore=DL3008
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    iptables \
    && rm -rf /var/lib/apt/lists/* \
    && update-alternatives --set iptables /usr/sbin/iptables-legacy \
    && update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy

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

# =============================================================================
# Docker Configuration
# =============================================================================

# Add developer user to docker group and enable Docker service
RUN usermod -aG docker "$USERNAME" && \
    systemctl enable docker

# STOPSIGNAL, VOLUME, and CMD are inherited from systemd image
