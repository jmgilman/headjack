# Headjack systemd Image
#
# Extends the base image with systemd init system support.
# Use this for multi-service environments that need a proper init system.
# For Docker-in-Docker support, use the dind variant instead.
# See docs/designs/base-image.md for full specification.

# hadolint ignore=DL3006
FROM base

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Switch to root - systemd requires root privileges as PID 1
# hadolint ignore=DL3002
USER root

# =============================================================================
# Install systemd
# =============================================================================

# hadolint ignore=DL3008
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    systemd \
    systemd-sysv \
    && rm -rf /var/lib/apt/lists/*

# =============================================================================
# systemd Configuration
# =============================================================================

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

# systemd as init system
CMD ["/lib/systemd/systemd"]
