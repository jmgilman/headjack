---
sidebar_position: 5
title: OCI Labels
description: OCI label reference for custom images
---

# OCI Labels Reference

Headjack uses OCI image labels to configure container runtime behavior. When building custom images, you can use these labels to control how Headjack runs your containers.

## Overview

Labels are key-value pairs embedded in container images. Headjack reads these labels at runtime to determine how to configure the container.

Labels are set in Dockerfiles using the `LABEL` instruction:

```dockerfile
LABEL io.headjack.init="/lib/systemd/systemd"
```

## Available Labels

### io.headjack.init

Specifies the command to run as PID 1 inside the container.

| Property | Value |
|----------|-------|
| Key | `io.headjack.init` |
| Value type | String (command path) |
| Default | `sleep infinity` |

#### Description

By default, Headjack runs `sleep infinity` as PID 1 to keep the container alive while sessions run in the background. This label overrides that default with a custom init command.

#### Example

```dockerfile
# Use systemd as init
LABEL io.headjack.init="/lib/systemd/systemd"

# Use a custom init script
LABEL io.headjack.init="/usr/local/bin/my-init.sh"
```

#### Usage in Official Images

| Image | Value |
|-------|-------|
| `base` | Not set (uses default `sleep infinity`) |
| `systemd` | `/lib/systemd/systemd` |
| `dind` | Inherited from `systemd` |

---

### io.headjack.podman.flags

Specifies additional flags to pass to Podman when running the container.

| Property | Value |
|----------|-------|
| Key | `io.headjack.podman.flags` |
| Value type | String (space-separated key=value pairs) |
| Default | None |

#### Description

This label allows images to specify Podman-specific runtime flags that are required for correct operation. Headjack parses the value and applies the flags when creating the container.

#### Format

The value is a space-separated list of key=value pairs:

```
key1=value1 key2=value2
```

#### Supported Flags

| Flag | Description |
|------|-------------|
| `systemd=always` | Enable systemd container mode |
| `systemd=true` | Enable systemd container mode if systemd is detected |

#### Example

```dockerfile
# Enable systemd mode
LABEL io.headjack.podman.flags="systemd=always"

# Multiple flags
LABEL io.headjack.podman.flags="systemd=always privileged=true"
```

#### Usage in Official Images

| Image | Value |
|-------|-------|
| `base` | Not set |
| `systemd` | `systemd=always` |
| `dind` | Inherited from `systemd` |

---

### io.headjack.docker.flags

Specifies additional flags to pass to Docker when running the container.

| Property | Value |
|----------|-------|
| Key | `io.headjack.docker.flags` |
| Value type | String (space-separated key=value pairs) |
| Default | None |

#### Description

This label allows images to specify Docker-specific runtime flags that are required for correct operation. Headjack parses the value and applies the flags when creating the container. The format is the same as `io.headjack.podman.flags`.

#### Example

```dockerfile
# Enable privileged mode
LABEL io.headjack.docker.flags="privileged=true"
```

#### Usage in Official Images

| Image | Value |
|-------|-------|
| `base` | Not set |
| `systemd` | `privileged=true cgroupns=host volume=/sys/fs/cgroup:/sys/fs/cgroup:rw` |
| `dind` | Inherited from `systemd` |

## Building Custom Images

When building custom images that extend the official Headjack images, labels are not automatically inherited. You must explicitly set any labels you need.

### Extending the Base Image

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

# Add custom software
RUN apt-get update && apt-get install -y postgresql

# No labels needed - base image uses default init
```

### Extending the Systemd Image

```dockerfile
FROM ghcr.io/gilmanlab/headjack:systemd

# Add custom systemd service
COPY myservice.service /etc/systemd/system/
RUN systemctl enable myservice

# Re-declare labels (not inherited)
LABEL io.headjack.init="/lib/systemd/systemd"
LABEL io.headjack.podman.flags="systemd=always"
LABEL io.headjack.docker.flags="privileged=true cgroupns=host volume=/sys/fs/cgroup:/sys/fs/cgroup:rw"
```

### Creating a Custom Init Image

```dockerfile
FROM ghcr.io/gilmanlab/headjack:base

# Add custom init script
COPY init.sh /usr/local/bin/init.sh
RUN chmod +x /usr/local/bin/init.sh

# Configure Headjack to use custom init
LABEL io.headjack.init="/usr/local/bin/init.sh"
```

## Label Inspection

You can inspect image labels using Docker or Podman:

```bash
# Using Docker
docker inspect ghcr.io/gilmanlab/headjack:systemd --format='{{json .Config.Labels}}' | jq

# Using Podman
podman inspect ghcr.io/gilmanlab/headjack:systemd --format='{{json .Config.Labels}}' | jq
```

Example output:

```json
{
  "io.headjack.init": "/lib/systemd/systemd",
  "io.headjack.podman.flags": "systemd=always",
  "io.headjack.docker.flags": "privileged=true cgroupns=host volume=/sys/fs/cgroup:/sys/fs/cgroup:rw"
}
```

## See Also

- [Overview](overview.md) - Image variant comparison
- [Base Dockerfile](https://github.com/GilmanLab/headjack/blob/master/images/base/Dockerfile)
- [Systemd Dockerfile](https://github.com/GilmanLab/headjack/blob/master/images/systemd/Dockerfile)
- [Docker-in-Docker Dockerfile](https://github.com/GilmanLab/headjack/blob/master/images/dind/Dockerfile)
