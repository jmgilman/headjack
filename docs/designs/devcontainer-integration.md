# Devcontainer Integration Design

## Overview

This document explores approaches for integrating devcontainer support into Headjack, allowing users to point Headjack at a repository containing a `.devcontainer/devcontainer.json` and have it brought up as a managed instance with session support.

## Background

### Current Headjack Architecture

Headjack uses a layered architecture for container management:

1. **Runtime Interface** (`internal/container/container.go`): Abstracts container operations (Run, Exec, Stop, etc.) across Docker, Podman, and Apple Containerization
2. **Instance Manager** (`internal/instance/manager.go`): Orchestrates instance lifecycle, creating worktrees, containers, and managing sessions
3. **Image Labels**: Runtime configuration is extracted from OCI image labels (`io.headjack.init`, `io.headjack.podman.flags`, etc.)
4. **Configuration Merging**: Flags from config files take precedence over image labels

### Dev Container Ecosystem

The [Development Containers Specification](https://containers.dev/) is an open standard backed by Microsoft and GitHub. Key components:

- **devcontainer.json**: JSON configuration file defining the dev environment
- **Dev Container Features**: Modular, reusable configuration units
- **Dev Container CLI**: Reference implementation (`@devcontainers/cli` npm package)

## Integration Approaches

### Option 1: Wrap the Official Dev Container CLI (Recommended)

**Approach**: Use the official `@devcontainers/cli` as a subprocess, leveraging its full feature support.

**How it works**:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Headjack                                  │
├─────────────────────────────────────────────────────────────────┤
│  Instance Manager                                                │
│    │                                                            │
│    ├── Vanilla Image Path (current)                             │
│    │     └── container.Runtime.Run()                            │
│    │                                                            │
│    └── Devcontainer Path (new)                                  │
│          └── DevcontainerAdapter                                │
│                │                                                │
│                ├── devcontainer up --workspace-folder <path>    │
│                │     → Returns JSON: {containerId, remoteUser}  │
│                │                                                │
│                ├── devcontainer exec <cmd>                      │
│                │     → Executes with remoteUser, remoteEnv      │
│                │                                                │
│                └── docker/podman stop/rm (lifecycle)            │
└─────────────────────────────────────────────────────────────────┘
```

**CLI Commands Used**:

| Headjack Operation | Dev Container CLI Command |
|-------------------|--------------------------|
| Create instance | `devcontainer up --workspace-folder <path> --docker-path <runtime>` |
| Execute command | `devcontainer exec --workspace-folder <path> <cmd>` |
| Read config | `devcontainer read-configuration --workspace-folder <path>` |
| Stop container | `docker stop <containerId>` (use returned ID) |
| Remove container | `docker rm <containerId>` |

**Key CLI Options**:

```bash
devcontainer up \
  --workspace-folder /path/to/repo \
  --docker-path podman \              # Use podman instead of docker
  --docker-compose-path podman-compose \
  --config /path/to/devcontainer.json \
  --mount "type=bind,source=/host,target=/container"
```

**JSON Output from `devcontainer up`**:

```json
{
  "outcome": "success",
  "containerId": "abc123...",
  "remoteUser": "vscode",
  "remoteWorkspaceFolder": "/workspaces/myproject"
}
```

**Advantages**:
- Full spec compliance (features, lifecycle hooks, compose support)
- Automatic updates with new spec versions
- Minimal maintenance burden
- Handles complex build scenarios (multi-stage, features, caching)
- Automatic metadata embedding in image labels

**Disadvantages**:
- Node.js dependency (already present in Headjack images)
- Additional subprocess overhead
- Less control over container creation details

**Implementation Sketch**:

```go
// internal/devcontainer/adapter.go

type Adapter struct {
    binaryPath string      // Path to devcontainer CLI
    dockerPath string      // Path to docker/podman binary
    exec       exec.Runner // Command executor
}

type UpResult struct {
    Outcome               string `json:"outcome"`
    ContainerID           string `json:"containerId"`
    RemoteUser            string `json:"remoteUser"`
    RemoteWorkspaceFolder string `json:"remoteWorkspaceFolder"`
}

func (a *Adapter) Up(ctx context.Context, workspaceFolder string) (*UpResult, error) {
    args := []string{
        "up",
        "--workspace-folder", workspaceFolder,
        "--docker-path", a.dockerPath,
    }

    out, err := a.exec.Run(ctx, a.binaryPath, args...)
    if err != nil {
        return nil, fmt.Errorf("devcontainer up: %w", err)
    }

    var result UpResult
    if err := json.Unmarshal(out, &result); err != nil {
        return nil, fmt.Errorf("parse devcontainer output: %w", err)
    }

    return &result, nil
}

func (a *Adapter) Exec(ctx context.Context, workspaceFolder string, cmd []string) error {
    args := []string{
        "exec",
        "--workspace-folder", workspaceFolder,
    }
    args = append(args, cmd...)

    return a.exec.RunInteractive(ctx, a.binaryPath, args...)
}
```

---

### Option 2: Parse devcontainer.json Directly

**Approach**: Implement a subset of the devcontainer spec natively in Go.

**How it works**:

```
┌─────────────────────────────────────────────────────────────────┐
│  devcontainer.json Parser                                        │
│    │                                                            │
│    ├── Parse JSON → DevcontainerConfig struct                   │
│    │                                                            │
│    └── Translate to container.RunConfig                         │
│          ├── Image → RunConfig.Image                            │
│          ├── Mounts → RunConfig.Mounts                          │
│          ├── containerEnv → RunConfig.Env                       │
│          ├── Features → (pre-build image with features)         │
│          └── Flags (privileged, capAdd) → RunConfig.Flags       │
└─────────────────────────────────────────────────────────────────┘
```

**Supported Properties** (minimal viable set):

```go
type DevcontainerConfig struct {
    Name          string            `json:"name"`
    Image         string            `json:"image"`
    Build         *BuildConfig      `json:"build"`
    ContainerEnv  map[string]string `json:"containerEnv"`
    RemoteEnv     map[string]string `json:"remoteEnv"`
    RemoteUser    string            `json:"remoteUser"`
    Mounts        []string          `json:"mounts"`
    ForwardPorts  []any             `json:"forwardPorts"`
    Privileged    bool              `json:"privileged"`
    CapAdd        []string          `json:"capAdd"`
    SecurityOpt   []string          `json:"securityOpt"`

    // Lifecycle hooks
    PostCreateCommand any `json:"postCreateCommand"`
    PostStartCommand  any `json:"postStartCommand"`

    // Features would require significant implementation
    Features map[string]any `json:"features"`
}
```

**Advantages**:
- No external dependencies
- Faster startup (no subprocess)
- Full control over behavior
- Lighter runtime footprint

**Disadvantages**:
- Significant implementation effort
- Ongoing maintenance as spec evolves
- Features support is complex (require building feature images)
- Docker Compose support adds complexity
- Risk of spec divergence

**What Would NOT Be Supported**:
- Dev Container Features (would require implementing the feature installation process)
- Docker Compose multi-container setups
- Complex build scenarios (caching, multi-stage)
- `waitFor`, `userEnvProbe`, and other advanced properties

---

### Option 3: Hybrid Approach

**Approach**: Use the CLI for building/setup, but manage the container lifecycle directly.

**How it works**:

1. **Build Phase**: Use `devcontainer build` to create an image with all features applied
2. **Run Phase**: Use Headjack's native `container.Runtime.Run()` with the pre-built image
3. **Exec Phase**: Use Headjack's native `container.Runtime.Exec()`

```
┌─────────────────────────────────────────────────────────────────┐
│  Build Phase (once per devcontainer.json change)                │
│    └── devcontainer build --image-name hjk-<hash>               │
│          → Creates image with features applied                  │
│          → Embeds metadata in dev.containers.metadata label     │
├─────────────────────────────────────────────────────────────────┤
│  Run Phase (Headjack native)                                    │
│    └── container.Runtime.Run()                                  │
│          ├── Image: hjk-<hash>                                  │
│          ├── Read metadata from label                           │
│          └── Apply containerEnv, mounts, etc.                   │
├─────────────────────────────────────────────────────────────────┤
│  Exec Phase (Headjack native)                                   │
│    └── container.Runtime.Exec()                                 │
│          ├── Use remoteUser from metadata                       │
│          └── Apply remoteEnv                                    │
└─────────────────────────────────────────────────────────────────┘
```

**Advantages**:
- Full feature support via CLI build
- Consistent container lifecycle with vanilla images
- Leverages existing Headjack abstractions
- Pre-built images can be cached/reused

**Disadvantages**:
- Still requires CLI dependency
- Need to track image cache validity
- Lifecycle hooks require special handling

---

## Recommendation: Option 1 (CLI Wrapper)

**Rationale**:

1. **Spec Compliance**: The devcontainer spec is actively evolving. Wrapping the CLI ensures automatic support for new features.

2. **Feature Support**: Dev Container Features are the primary value proposition. Implementing feature installation natively would be a substantial project.

3. **Maintenance**: The Headjack team can focus on core functionality rather than keeping pace with devcontainer spec changes.

4. **Node.js Already Present**: Headjack images already include Node.js for agent CLIs (claude-code, gemini-cli).

5. **Industry Precedent**:
   - [DevPod](https://devpod.sh) natively implements the spec
   - [JetBrains IDEs](https://www.jetbrains.com/help/idea/dev-container-cli.html) use the CLI
   - [Coder](https://coder.com/docs/user-guides/devcontainers) uses the CLI

## Implementation Plan

### Phase 1: Core Integration

1. **Add Devcontainer Adapter** (`internal/devcontainer/adapter.go`)
   - Wrap CLI commands: `up`, `exec`, `read-configuration`
   - Parse JSON output
   - Handle `--docker-path` for runtime selection

2. **Extend Instance Manager**
   - Detect `.devcontainer/devcontainer.json` in repository
   - Create instances using adapter instead of direct runtime calls
   - Store devcontainer metadata in catalog

3. **Extend Catalog Entry**
   ```go
   type Entry struct {
       // ... existing fields ...
       Devcontainer *DevcontainerInfo `json:"devcontainer,omitempty"`
   }

   type DevcontainerInfo struct {
       ConfigPath            string `json:"configPath"`
       RemoteUser            string `json:"remoteUser"`
       RemoteWorkspaceFolder string `json:"remoteWorkspaceFolder"`
   }
   ```

4. **CLI Changes**
   - Add `--devcontainer` flag to `run` command
   - Auto-detect devcontainer.json if present

### Phase 2: Enhanced Features

1. **Configuration Merging**
   - Allow Headjack config to override devcontainer settings
   - Precedence: CLI flags > config file > devcontainer.json

2. **Lifecycle Hook Integration**
   - Run `postCreateCommand` after container creation
   - Run `postStartCommand` on each start

3. **Rebuild Support**
   - Detect devcontainer.json changes
   - Prompt user to rebuild when configuration changes

### Phase 3: Polish

1. **Image Caching**
   - Pre-build devcontainer images
   - Cache management commands

2. **Compose Support**
   - Handle `dockerComposeFile` configurations
   - Multi-container instance management

## Open Questions

1. **Worktree Handling**: Should devcontainer instances use Headjack's worktree model, or respect the devcontainer's `workspaceMount`?

2. **Runtime Flags Merging**: How should Headjack's `runtime.flags` config interact with devcontainer's `runArgs`?

3. **Agent Compatibility**: Are there devcontainer configurations that would break agent CLI operation?

4. **CLI Installation**: Should Headjack install the devcontainer CLI automatically, or require users to install it?

## References

- [Dev Container Specification](https://containers.dev/)
- [Dev Container CLI Repository](https://github.com/devcontainers/cli)
- [CLI Reference Implementation Docs](https://containers.dev/implementors/reference/)
- [devcontainer.json Reference](https://containers.dev/implementors/json_reference/)
- [Dev Container Features](https://containers.dev/implementors/features/)
- [DevPod devcontainer.json Support](https://devpod.sh/docs/developing-in-workspaces/devcontainer-json)
- [JetBrains Dev Container CLI Integration](https://www.jetbrains.com/help/idea/dev-container-cli.html)
