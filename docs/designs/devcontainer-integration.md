# Devcontainer Integration Design

## Overview

This document describes the design for integrating devcontainer support into Headjack, allowing users to point Headjack at a repository containing a `.devcontainer/devcontainer.json` and have it brought up as a managed instance with session support.

## Background

### Current Headjack Architecture

Headjack uses a layered architecture for container management:

1. **Runtime Interface** (`internal/container/container.go`): Abstracts container operations (Run, Exec, Stop, etc.) across Docker, Podman, and Apple Containerization
2. **Instance Manager** (`internal/instance/manager.go`): Orchestrates instance lifecycle, creating worktrees, containers, and managing sessions
3. **Image Labels**: Runtime configuration is extracted from OCI image labels (`io.headjack.init`, `io.headjack.podman.flags`, etc.)
4. **Configuration Merging**: Flags from config files take precedence over image labels

The Instance Manager depends on a `containerRuntime` interface:

```go
type containerRuntime interface {
    Run(ctx context.Context, cfg *container.RunConfig) (*container.Container, error)
    Exec(ctx context.Context, id string, cfg container.ExecConfig) error
    Stop(ctx context.Context, id string) error
    Start(ctx context.Context, id string) error
    Remove(ctx context.Context, id string) error
    Get(ctx context.Context, id string) (*container.Container, error)
    List(ctx context.Context, filter container.ListFilter) ([]container.Container, error)
    ExecCommand() []string
}
```

### Dev Container Ecosystem

The [Development Containers Specification](https://containers.dev/) is an open standard backed by Microsoft and GitHub. Key components:

- **devcontainer.json**: JSON configuration file defining the dev environment
- **Dev Container Features**: Modular, reusable configuration units
- **Dev Container CLI**: Reference implementation (`@devcontainers/cli` npm package)

The Dev Container CLI provides commands that handle the complexity of the spec:

| Command | Purpose |
|---------|---------|
| `devcontainer up` | Create and start container with full config applied |
| `devcontainer exec` | Execute commands with remoteUser, remoteEnv applied |
| `devcontainer build` | Pre-build images with features |
| `devcontainer read-configuration` | Parse effective configuration |

## Design

### Core Principle: Devcontainer as a Runtime Decorator

The devcontainer CLI is not a container runtime—it's a layer that sits on top of Docker or Podman. The design reflects this by implementing `DevcontainerRuntime` as a **decorator** that wraps an underlying runtime:

```
┌─────────────────────────────────────────────────────────────────┐
│  DevcontainerRuntime (implements containerRuntime)              │
│    │                                                            │
│    ├── Run()   → devcontainer up --docker-path <underlying>     │
│    ├── Exec()  → devcontainer exec                              │
│    │                                                            │
│    └── Delegates to underlying runtime:                         │
│          ├── Stop()   → underlying.Stop()                       │
│          ├── Start()  → underlying.Start()                      │
│          ├── Remove() → underlying.Remove()                     │
│          ├── Get()    → underlying.Get()                        │
│          └── List()   → underlying.List()                       │
└─────────────────────────────────────────────────────────────────┘
```

This approach:
- **Reuses the existing `containerRuntime` interface** — no changes to Instance Manager
- **No parallel code paths** — just a different runtime implementation
- **Leverages CLI for creation/exec** — where the spec complexity lives
- **Delegates lifecycle to underlying runtime** — containers are just containers once created

### Runtime Support

The devcontainer CLI supports Docker and Podman via the `--docker-path` flag:

| Runtime | Supported | Notes |
|---------|-----------|-------|
| Docker | ✅ | Native support |
| Podman | ✅ | Via `--docker-path podman` |
| Apple Containerization | ❌ | Not Docker-compatible |

Attempting to use devcontainer mode with Apple Containerization will result in an error.

### RunConfig Extension

Add a `WorkspaceFolder` field to `RunConfig` for devcontainer mode:

```go
type RunConfig struct {
    Name            string   // Container name (required)
    Image           string   // OCI image reference (required for vanilla mode)
    Mounts          []Mount  // Volume mounts
    Env             []string // Environment variables
    Init            string   // Init command
    Flags           []string // Runtime-specific flags
    WorkspaceFolder string   // For devcontainer: path to folder with devcontainer.json
}
```

- Vanilla runtimes (Docker, Podman, Apple) ignore `WorkspaceFolder`
- `DevcontainerRuntime` uses `WorkspaceFolder` and ignores `Image`

### DevcontainerRuntime Implementation

```go
// internal/devcontainer/runtime.go

package devcontainer

// Runtime wraps an underlying container runtime with devcontainer CLI support.
type Runtime struct {
    underlying container.Runtime  // Docker or Podman runtime
    cliPath    string             // Path to devcontainer CLI binary
    dockerPath string             // Path to docker/podman binary (for --docker-path)
    exec       exec.Runner        // Command executor
}

// NewRuntime creates a DevcontainerRuntime wrapping the given underlying runtime.
func NewRuntime(underlying container.Runtime, cliPath, dockerPath string) *Runtime {
    return &Runtime{
        underlying: underlying,
        cliPath:    cliPath,
        dockerPath: dockerPath,
    }
}

// Run creates a container using devcontainer up.
func (r *Runtime) Run(ctx context.Context, cfg *container.RunConfig) (*container.Container, error) {
    args := []string{
        "up",
        "--workspace-folder", cfg.WorkspaceFolder,
        "--docker-path", r.dockerPath,
    }

    out, err := r.exec.Run(ctx, r.cliPath, args...)
    if err != nil {
        return nil, fmt.Errorf("devcontainer up: %w", err)
    }

    var result upResult
    if err := json.Unmarshal(out, &result); err != nil {
        return nil, fmt.Errorf("parse devcontainer output: %w", err)
    }

    if result.Outcome != "success" {
        return nil, fmt.Errorf("devcontainer up failed: %s", result.Outcome)
    }

    // Return container info - Get() will fetch full details
    return &container.Container{
        ID:     result.ContainerID,
        Name:   cfg.Name,
        Status: container.StatusRunning,
    }, nil
}

// Exec executes a command using devcontainer exec.
func (r *Runtime) Exec(ctx context.Context, id string, cfg container.ExecConfig) error {
    args := []string{
        "exec",
        "--workspace-folder", cfg.Workdir, // devcontainer exec uses workspace folder
    }
    args = append(args, cfg.Command...)

    return r.exec.RunInteractive(ctx, r.cliPath, args...)
}

// Stop delegates to the underlying runtime.
func (r *Runtime) Stop(ctx context.Context, id string) error {
    return r.underlying.Stop(ctx, id)
}

// Start delegates to the underlying runtime.
func (r *Runtime) Start(ctx context.Context, id string) error {
    return r.underlying.Start(ctx, id)
}

// Remove delegates to the underlying runtime.
func (r *Runtime) Remove(ctx context.Context, id string) error {
    return r.underlying.Remove(ctx, id)
}

// Get delegates to the underlying runtime.
func (r *Runtime) Get(ctx context.Context, id string) (*container.Container, error) {
    return r.underlying.Get(ctx, id)
}

// List delegates to the underlying runtime.
func (r *Runtime) List(ctx context.Context, filter container.ListFilter) ([]container.Container, error) {
    return r.underlying.List(ctx, filter)
}

// ExecCommand returns the underlying runtime's exec command.
func (r *Runtime) ExecCommand() []string {
    return r.underlying.ExecCommand()
}

// upResult represents the JSON output from devcontainer up.
type upResult struct {
    Outcome               string `json:"outcome"`
    ContainerID           string `json:"containerId"`
    RemoteUser            string `json:"remoteUser"`
    RemoteWorkspaceFolder string `json:"remoteWorkspaceFolder"`
}
```

### User Experience

**Auto-detection with explicit override:**

```bash
# Auto-detects devcontainer.json if present in repo
hjk run

# Explicit image skips devcontainer, uses vanilla mode
hjk run --image ghcr.io/gilmanlab/headjack:base
```

**Precedence:**

1. `--image` flag passed → vanilla mode with specified image
2. `.devcontainer/devcontainer.json` exists → devcontainer mode
3. No devcontainer.json → vanilla mode with default base image

This matches user expectations from VS Code, GitHub Codespaces, and DevPod.

### Runtime Selection Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  CLI Layer (cmd/run.go)                                         │
│    │                                                            │
│    ├── --image flag passed?                                     │
│    │     └── Yes: Use vanilla runtime (Docker/Podman/Apple)     │
│    │                                                            │
│    └── devcontainer.json exists?                                │
│          ├── Yes + Docker/Podman: Use DevcontainerRuntime       │
│          ├── Yes + Apple: Error (not supported)                 │
│          └── No: Use vanilla runtime with default image         │
│                                                                 │
│  Instance Manager receives containerRuntime interface           │
│  (doesn't know or care which implementation)                    │
└─────────────────────────────────────────────────────────────────┘
```

### Catalog Metadata

Store devcontainer-specific metadata for exec operations:

```go
type Entry struct {
    // ... existing fields ...
    RemoteUser    string `json:"remoteUser,omitempty"`    // From devcontainer up output
    RemoteWorkdir string `json:"remoteWorkdir,omitempty"` // From devcontainer up output
}
```

## Implementation Plan

### Phase 1: Core Integration

1. **Extend RunConfig** (`internal/container/container.go`)
   - Add `WorkspaceFolder string` field

2. **Implement DevcontainerRuntime** (`internal/devcontainer/runtime.go`)
   - Implement `containerRuntime` interface
   - Wrap underlying Docker/Podman runtime
   - Call `devcontainer up` for Run()
   - Call `devcontainer exec` for Exec()
   - Delegate lifecycle methods to underlying runtime

3. **Add devcontainer detection** (`internal/devcontainer/detect.go`)
   - Check for `.devcontainer/devcontainer.json`
   - Support alternate locations per spec

4. **Wire up CLI** (`internal/cmd/run.go`)
   - Detect devcontainer.json when no `--image` flag
   - Construct DevcontainerRuntime wrapping base runtime
   - Pass to Instance Manager

5. **Extend Catalog Entry**
   - Add `RemoteUser` and `RemoteWorkdir` fields
   - Populate from `devcontainer up` JSON output

### Phase 2: Polish

1. **Error handling**
   - Clear error when using devcontainer + Apple runtime
   - Parse devcontainer CLI errors into actionable messages

2. **Exec improvements**
   - Use stored `RemoteUser` and `RemoteWorkdir` for exec commands
   - Handle `remoteEnv` from devcontainer.json

3. **Rebuild support**
   - Detect devcontainer.json changes
   - Trigger rebuild when configuration changes

## Lifecycle Hooks

The `devcontainer up` command handles lifecycle hooks automatically:

| Hook | Behavior |
|------|----------|
| `onCreateCommand` | Run by `devcontainer up` on first creation |
| `postCreateCommand` | Run by `devcontainer up` after container created |
| `postStartCommand` | Run by `devcontainer up` on each start |
| `postAttachCommand` | Not automatically run (tool-specific) |

**Note:** When using `Runtime.Start()` directly (e.g., `hjk start`), `postStartCommand` will not run since we delegate to the underlying runtime. This is an acceptable trade-off—users needing full hook fidelity can use `hjk rm` + `hjk run` to recreate.

## References

- [Dev Container Specification](https://containers.dev/)
- [Dev Container CLI Repository](https://github.com/devcontainers/cli)
- [CLI Reference Implementation Docs](https://containers.dev/implementors/reference/)
- [devcontainer.json Reference](https://containers.dev/implementors/json_reference/)
- [Dev Container Features](https://containers.dev/implementors/features/)
