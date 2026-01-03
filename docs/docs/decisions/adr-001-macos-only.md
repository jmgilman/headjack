---
sidebar_position: 1
title: "ADR-001: Initial macOS-Only Platform Support"
description: Initial decision to support macOS exclusively for the first release
---

# ADR 001: Initial macOS-Only Platform Support

## Status

Superseded

## Context

Headjack is a CLI tool for spawning isolated LLM coding agents. As a new project, we need to decide which platforms to support.

Platform support has significant implications:
- Development and testing burden scales with each platform
- Isolation technologies vary significantly across operating systems
- CLI tooling and system integration differ per platform

Relevant factors for this decision:

1. **Project scope**: Headjack is primarily a personal project. The author uses macOS exclusively for development.

2. **Target audience**: macOS has strong adoption among software developers, particularly in certain ecosystems (web, mobile, startups).

3. **Resource constraints**: Supporting multiple platforms requires additional development effort, testing infrastructure, and maintenance burden.

4. **Platform-specific challenges**:
   - **Linux**: Viable but not a current priority. Could be added later with reasonable effort.
   - **Windows**: Significant challenges around containerization, filesystem semantics, and shell integration. WSL adds complexity without solving core issues.

## Decision

Headjack would be limited to macOS for the initial release.

- Linux support was not ruled out but was not a priority
- Windows support was explicitly out of scope with no intention to add it

## Consequences

### Positive

- Reduced development and testing burden
- Can leverage macOS-native technologies without abstraction layers
- Faster iteration toward a working product
- Clear scope prevents feature creep

### Negative

- Limits potential user base to macOS users
- Contributors on other platforms cannot easily participate
- Some organizations with Linux-only policies cannot adopt

### Neutral

## Superseding Decision

Headjack now supports both macOS and Linux, and Docker is the default runtime. This ADR remains for historical context, but the platform scope has expanded.
