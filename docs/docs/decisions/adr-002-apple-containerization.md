---
sidebar_position: 2
title: "ADR-002: Apple Containerization Framework"
description: Decision to use Apple Containerization Framework for agent isolation
---

# ADR 002: Apple Containerization Framework for Agent Isolation

## Status

Accepted

## Context

Headjack spawns isolated LLM coding agents that need to operate with full autonomy. Each agent instance requires:

- A complete Linux environment (not single-process)
- Ability to run multiple services and systemd
- Docker-in-Docker capability for testing workflows
- Strong isolation from the host and other agents
- Fast startup times for good UX

We evaluated several isolation technologies for macOS.

### Options Considered

**Docker**
- Industry standard, excellent tooling and ecosystem
- Widely available on macOS via Docker Desktop
- Single-process optimization conflicts with multi-service agent needs
- Docker-in-Docker requires privileged mode, weakening isolation
- Namespace-based isolation, not hypervisor boundary
- Note: Docker is now supported as a runtime option for users who prefer it

**Lima**
- CNCF incubating project, mature and well-documented
- Uses Apple Virtualization.framework under the hood
- 10-60 second cold start times due to full VM boot + cloud-init
- Designed for persistent VMs, not ephemeral agent instances

**Apple Virtualization.framework directly (via `github.com/Code-Hex/vz`)**
- Native Go bindings available
- Maximum control over VM lifecycle
- Still incurs full Linux boot time (kernel + init system)
- Requires building all VM management infrastructure ourselves
- Speed claims over Lima are questionable—same underlying technology

**Apple Containerization Framework**
- New framework from Apple (macOS 26+)
- Each container runs in its own VM (hypervisor isolation)
- OCI-compatible (standard images, registries)
- ~1 second startup time
- Supports systemd, multiple services, nested virtualization
- Docker-in-Docker works (with iptables-legacy workaround)

## Decision

Use **Apple Containerization Framework** as the isolation technology for Headjack agents.

## Consequences

### Positive

- **Future-aligned**: Apple is investing in this as the containerization primitive for macOS. Performance and capabilities will improve over time.
- **Official Apple support**: First-party framework with ongoing development
- **Standard semantics**: OCI images, familiar container workflows, existing images work unchanged
- **True VM isolation**: Each agent runs in its own hypervisor-isolated VM, not shared namespaces
- **Native performance**: Built on Virtualization.framework, optimized for Apple Silicon
- **Full environment support**: systemd, multiple services, Docker-in-Docker all validated
- **Sub-second startup**: ~0.9s measured in testing

### Negative

- **Young project**: Pre-1.0, API may change, fewer community resources
- **No Go SDK**: Must shell out to `container` CLI and parse output
- **macOS 26+ required**: Limits to users on latest macOS
- **Memory overhead**: Each container is a dedicated VM (no memory ballooning)
- **Known limitations**: No container-to-container DNS, no IPv6 in nested Docker

### Neutral

- By adopting early, we participate in the framework's growth through usage and bug reports
- The iptables-legacy workaround for Docker-in-Docker is stable but adds base image complexity

## Addendum: Multi-Runtime Support

While Apple Containerization Framework remains the recommended runtime for its superior isolation properties, Headjack now supports multiple container runtimes to accommodate different user preferences and environments:

| Runtime | Configuration | Binary | Notes |
|---------|--------------|--------|-------|
| Docker | `runtime.name: docker` | `docker` | Default runtime. Cross-platform, widely available. |
| Apple | `runtime.name: apple` | `container` | Recommended for macOS 26+. VM-level isolation. |
| Podman | `runtime.name: podman` | `podman` | Daemonless alternative. |

Users can configure their preferred runtime via:

```bash
hjk config runtime.name docker
```

This flexibility allows teams to use familiar tooling while still benefiting from Headjack's instance and session management.
