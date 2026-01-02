---
sidebar_position: 4
title: "ADR-004: CLI-Based Agents"
description: Decision to focus on CLI-based agents with subscription pricing
---

# ADR 004: CLI-Based Agents over API-Based Automation

## Status

Accepted

## Context

Headjack orchestrates LLM coding agents. There are multiple approaches to building and running such agents:

### Options Considered

**API-based agents (direct API calls)**
- Full control over prompts, tools, and orchestration
- Token-based pricing (pay per input/output token)
- Requires building agent infrastructure (tool execution, context management, etc.)
- Costs scale directly with usage and can become significant

**Custom agent frameworks (LangChain, AutoGPT, etc.)**
- Pre-built abstractions for common patterns
- Still token-based pricing underneath
- Additional complexity and dependencies
- Framework lock-in

**IDE-integrated agents (Cursor, Copilot, etc.)**
- Tight editor integration
- Not designed for headless/autonomous operation
- Difficult to orchestrate programmatically

**CLI-based agents (Claude Code, etc.)**
- Subscription-based pricing model
- Designed for terminal/headless operation
- Built-in tooling, prompts, and capabilities
- Can be orchestrated via standard process management

## Decision

Focus on **CLI-based agents** with subscription pricing as the primary use case for Headjack.

The initial targets are the three front-runners in this space:
- **Claude Code** (Anthropic)
- **Gemini CLI** (Google)
- **Codex CLI** (OpenAI/ChatGPT)

All three support subscription-based usage.

## Consequences

### Positive

- **Cost efficiency**: Subscription packages offer significant savings over token-based pricing. This is the primary driver—API costs for autonomous agents can quickly become prohibitive.
- **Built-in capabilities**: CLI agents come with pre-built tools, prompts, and interaction patterns
- **Simpler integration**: Spawn a process, provide environment, capture output—no SDK integration required
- **Aligned incentives**: Subscription users want to maximize usage; Headjack enables running multiple concurrent agents

### Negative

- **Vendor dependency**: Tied to specific CLI agent implementations and their evolution
- **Less control**: Cannot customize agent internals, prompts, or tool definitions as deeply as API-based approaches
- **Authentication complexity**: Must handle CLI-specific auth flows (solved via `setup-token` for Claude Code)

### Neutral

- Architecture remains process-based; adding support for additional CLI agents is straightforward
- If subscription economics change, API-based agents could be added as an alternative mode
