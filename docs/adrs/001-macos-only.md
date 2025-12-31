# ADR 001: macOS-Only Platform Support

## Status

Accepted

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

Headjack will support **macOS only** for the initial release.

- Linux support is not ruled out but is not a priority
- Windows support is explicitly out of scope with no intention to add it

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

- Linux support remains a future option if demand materializes
- Decision can be revisited as project matures
