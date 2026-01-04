---
sidebar_position: 3
title: "ADR-003: Go as Implementation Language"
description: Decision to implement Headjack in Go
---

# ADR 003: Go as Implementation Language

## Status

Accepted

## Context

Headjack is a CLI tool that needs to:

- Manage container lifecycles via the `container` CLI
- Handle git operations (worktrees, checkouts)
- Parse JSON/structured output from subprocesses
- Provide a responsive user experience
- Distribute as a single binary

We need to choose an implementation language.

### Options Considered

**Python**
- Excellent for rapid prototyping
- Rich ecosystem for CLI tooling (Click, Typer)
- Distribution complexity (virtualenvs, dependencies, version management)
- Runtime required on target system

**TypeScript/Node.js**
- Familiar to many developers
- Good async model for subprocess management
- Large runtime dependency
- Distribution requires bundling or Node installation

**Rust**
- Excellent performance and safety guarantees
- Single binary distribution
- Steeper learning curve
- Slower iteration during prototyping phase

**Go**
- Type-safe with fast compilation
- Single binary distribution (no runtime dependencies)
- Strong standard library for CLI needs (flags, JSON, exec)
- Mature CLI frameworks (Cobra, urfave/cli)
- Good subprocess and concurrency primitives
- Widely used for CLI tools (Docker, Kubernetes, gh, etc.)

## Decision

Use **Go** as the implementation language for Headjack.

## Consequences

### Positive

- **Single binary distribution**: Users download one file, no runtime installation
- **Type safety**: Catch errors at compile time, refactor with confidence
- **CLI ecosystem**: Proven libraries (Cobra) and patterns from successful projects
- **Fast compilation**: Quick iteration during development
- **Subprocess handling**: `os/exec` and goroutines well-suited for managing container processes
- **Cross-compilation**: Easy to build for different architectures if needed later

### Negative

- **Verbosity**: More boilerplate than scripting languages
- **Error handling**: Explicit error checking adds code volume
- **No container CLI SDK**: Must shell out to Docker/Podman CLI (not a Go-specific limitation)

### Neutral

- Go is a pragmatic choice, not novel—aligns with industry patterns for CLI tooling
