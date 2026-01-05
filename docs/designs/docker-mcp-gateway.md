# Docker MCP Gateway Integration Design

## Overview

This document describes the design for integrating Docker MCP Gateway support into Headjack, allowing MCP (Model Context Protocol) tools to persist across container instances. The key challenge is enabling containerized agents to connect to an MCP Gateway running on the host machine.

## Background

### What is MCP?

The [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) is an open standard that enables AI applications to connect to external tools and data sources. MCP servers expose tools (functions), prompts, and resources that AI agents can use.

### What is Docker MCP Gateway?

[Docker MCP Gateway](https://github.com/docker/mcp-gateway) is Docker's open-source solution for orchestrating MCP servers. It acts as a centralized proxy between clients and MCP servers, providing:

- **Centralized management**: Single configuration point for all MCP servers
- **Security**: Containers run with restricted privileges, isolated networking
- **Credential management**: Secure injection of API keys and secrets
- **Tool discovery**: Automatic discovery of tools from running servers
- **Logging and tracing**: Full visibility into AI tool activity

### Current Headjack Architecture

Headjack spawns isolated containers for LLM coding agents. Each container:
- Has its own filesystem (git worktree)
- Runs in VM-level isolation
- Has limited network access by default
- Receives environment variables for API keys (Claude, Gemini, etc.)

The problem: MCP servers configured on the host are not accessible from within containers. This means:
1. Users must reconfigure MCP tools in each instance
2. No tool persistence across sessions
3. Duplicated configuration and credentials

## Design Goals

1. **Preserve MCP tools across instances**: Agents in containers should access the same MCP tools as the host
2. **Minimal configuration**: Users shouldn't need to manually configure networking
3. **Security**: Don't expose unnecessary host services to containers
4. **Opt-in behavior**: Not all users want MCP Gateway integration

## Technical Analysis

### Docker MCP Gateway Transport Options

The MCP Gateway supports multiple transports:

| Transport | Description | Multi-Client | Use Case |
|-----------|-------------|--------------|----------|
| `stdio` | Synchronous communication | ❌ Single | Default, simple scripts |
| `sse` | Server-Sent Events over HTTP | ✅ Multiple | HTTP streaming |
| `streaming` | Configurable port HTTP | ✅ Multiple | Production, containers |

For container integration, **streaming transport with a configurable port** is required:

```bash
docker mcp gateway run --port 8811 --transport streaming
```

### Claude Code MCP Configuration

Claude Code (the primary agent in Headjack) supports remote MCP servers via:

**CLI:**
```bash
claude mcp add --transport sse <name> <url>
```

**Configuration file (`.mcp.json`):**
```json
{
  "mcpServers": {
    "docker-gateway": {
      "type": "sse",
      "url": "http://host.docker.internal:8811/sse"
    }
  }
}
```

### Container-to-Host Connectivity

Containers can reach host services through several mechanisms:

| Method | Platform | Pros | Cons |
|--------|----------|------|------|
| `host.docker.internal` | macOS/Windows | Built-in, DNS resolution | Requires `--add-host` on Linux |
| `--add-host host.docker.internal:host-gateway` | Linux | Works on all Docker 20.10+ | Extra flag required |
| `--network=host` | All | Direct host access | Loses network isolation |
| Bridge gateway IP (`172.17.0.1`) | All | Always works | Hardcoded IP, varies |

**Recommended approach**: Use `host.docker.internal` with automatic `--add-host` flag on Linux.

## Proposed Design

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  Host Machine                                                        │
│                                                                      │
│  ┌──────────────────────┐     ┌──────────────────────────────────┐  │
│  │  Docker MCP Gateway  │     │  MCP Servers (containers)        │  │
│  │  :8811               │◄────┤  - filesystem-server             │  │
│  │                      │     │  - github-server                 │  │
│  │  Transports:         │     │  - postgres-server               │  │
│  │  - streaming         │     │  - etc.                          │  │
│  └──────────┬───────────┘     └──────────────────────────────────┘  │
│             │                                                        │
│             │ host.docker.internal:8811                             │
│             │                                                        │
│  ┌──────────┴───────────────────────────────────────────────────┐   │
│  │  Headjack Container (hjk-repo-branch)                         │   │
│  │                                                               │   │
│  │  ┌─────────────────────────────────────────────────────────┐ │   │
│  │  │  Claude Code Agent                                       │ │   │
│  │  │                                                          │ │   │
│  │  │  ~/.mcp.json:                                           │ │   │
│  │  │  {                                                       │ │   │
│  │  │    "mcpServers": {                                      │ │   │
│  │  │      "docker-gateway": {                                │ │   │
│  │  │        "type": "sse",                                   │ │   │
│  │  │        "url": "http://host.docker.internal:8811/sse"   │ │   │
│  │  │      }                                                  │ │   │
│  │  │    }                                                    │ │   │
│  │  │  }                                                      │ │   │
│  │  └─────────────────────────────────────────────────────────┘ │   │
│  └───────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### Configuration Options

Add new configuration for MCP Gateway integration:

```yaml
# ~/.config/headjack/config.yaml
mcp:
  gateway:
    enabled: true                    # Enable MCP Gateway integration
    url: "http://host.docker.internal:8811/sse"  # Gateway URL from container perspective
    host_port: 8811                  # Port gateway listens on (for auto-start)
```

### Runtime Flag Injection

When MCP Gateway is enabled, Headjack automatically adds the required runtime flags:

```go
// internal/mcp/gateway.go

func (g *Gateway) RuntimeFlags() []string {
    // On Linux, Docker requires explicit host mapping
    if runtime.GOOS == "linux" {
        return []string{"--add-host=host.docker.internal:host-gateway"}
    }
    // macOS/Windows have built-in support
    return nil
}
```

### MCP Configuration Injection

When creating a Claude session, inject the MCP configuration:

```go
// internal/instance/manager.go

func (m *Manager) setupMCPConfig(ctx context.Context, entry *catalog.Entry) error {
    if !m.config.MCP.Gateway.Enabled {
        return nil
    }

    mcpConfig := map[string]any{
        "mcpServers": map[string]any{
            "docker-gateway": map[string]any{
                "type": "sse",
                "url":  m.config.MCP.Gateway.URL,
            },
        },
    }

    configJSON, _ := json.MarshalIndent(mcpConfig, "", "  ")

    // Write to container's ~/.mcp.json
    return m.writeContainerFile(ctx, entry, "~/.mcp.json", configJSON)
}
```

### Gateway Lifecycle Management (Optional)

Headjack can optionally manage the MCP Gateway lifecycle:

```go
// internal/mcp/gateway.go

type Gateway struct {
    config   *config.MCPConfig
    runtime  container.Runtime
    executor exec.Executor
}

// EnsureRunning starts the MCP Gateway if not already running
func (g *Gateway) EnsureRunning(ctx context.Context) error {
    // Check if gateway is already running
    if g.isRunning(ctx) {
        return nil
    }

    // Start gateway with streaming transport
    args := []string{
        "mcp", "gateway", "run",
        "--port", strconv.Itoa(g.config.HostPort),
        "--transport", "streaming",
    }

    return g.executor.StartBackground(ctx, "docker", args...)
}

func (g *Gateway) isRunning(ctx context.Context) bool {
    // Try to connect to gateway URL
    resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", g.config.HostPort))
    return err == nil && resp.StatusCode == 200
}
```

### Devcontainer Integration

For devcontainer mode, the host mapping can be added to `runArgs`:

```json
// .devcontainer/devcontainer.json
{
  "runArgs": [
    "--add-host=host.docker.internal:host-gateway"
  ]
}
```

Or via Headjack's devcontainer feature injection (future enhancement).

## Implementation Plan

### Phase 1: Core Integration

1. **Add MCP configuration** (`internal/config/config.go`)
   - Add `MCP.Gateway.Enabled`, `URL`, `HostPort` fields
   - Default to disabled

2. **Implement runtime flag injection** (`internal/mcp/gateway.go`)
   - Detect platform (Linux vs macOS/Windows)
   - Return appropriate `--add-host` flags

3. **Wire flags into container creation** (`internal/instance/manager.go`)
   - Merge MCP flags with user-configured flags
   - Apply to both vanilla and devcontainer modes

4. **Add MCP config injection for Claude** (`internal/instance/manager.go`)
   - Create `~/.mcp.json` in container during session setup
   - Only for Claude agent type

### Phase 2: Gateway Management

1. **Implement gateway health check** (`internal/mcp/gateway.go`)
   - HTTP health endpoint check
   - Timeout handling

2. **Add gateway auto-start** (`internal/mcp/gateway.go`)
   - Start gateway if enabled but not running
   - Background process management

3. **CLI commands** (`internal/cmd/mcp.go`)
   - `hjk mcp status` - Show gateway status
   - `hjk mcp start` - Start gateway
   - `hjk mcp stop` - Stop gateway

### Phase 3: Enhanced Features

1. **Per-instance MCP config**
   - Allow instances to have additional MCP servers
   - Merge with gateway config

2. **Devcontainer feature**
   - Create devcontainer feature for MCP Gateway connectivity
   - Auto-inject into user devcontainers

3. **Tool filtering**
   - Allow configuration of which tools to expose per instance
   - Security boundary between instances

## User Experience

### Basic Usage

```bash
# Enable MCP Gateway integration (one-time setup)
hjk config set mcp.gateway.enabled true

# Create instance - automatically connects to MCP Gateway
hjk run

# Inside container, Claude has access to all gateway tools
claude "Use the github tool to list my repositories"
```

### Verification

```bash
# Check MCP status in instance
hjk exec <instance> -- claude mcp list

# Expected output:
# docker-gateway: http://host.docker.internal:8811/sse (SSE) - ✓ Connected
#   Tools: 47 available
```

### Manual Gateway Start

```bash
# Start gateway manually (if not using Docker Desktop)
docker mcp gateway run --port 8811 --transport streaming

# Or via headjack
hjk mcp start
```

## Security Considerations

1. **Host exposure**: Only the MCP Gateway port is exposed, not arbitrary host services
2. **Tool access**: Gateway handles authentication and authorization
3. **Network isolation**: Containers still can't access other host services
4. **Credential management**: Secrets stay in gateway, not exposed to containers

## Alternatives Considered

### Alternative 1: Stdio Proxy

Run an MCP proxy inside each container that connects to host via stdio.

**Rejected because:**
- Adds complexity to container images
- One proxy per container = resource overhead
- Doesn't solve the multi-client problem

### Alternative 2: Host Network Mode

Use `--network=host` for all containers.

**Rejected because:**
- Loses network isolation (security risk)
- Containers can access all host ports
- Not compatible with some devcontainer features

### Alternative 3: Sidecar Container

Run MCP Gateway as a sidecar in each instance's network namespace.

**Rejected because:**
- One gateway per instance = no tool sharing
- Resource overhead
- Configuration duplication

## References

- [Docker MCP Gateway](https://github.com/docker/mcp-gateway)
- [Docker MCP Docs](https://docs.docker.com/ai/mcp-catalog-and-toolkit/mcp-gateway/)
- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
- [Claude Code MCP Configuration](https://docs.claude.com/en/docs/claude-code/mcp)
- [Docker Networking: host.docker.internal](https://docs.docker.com/desktop/features/networking/)
