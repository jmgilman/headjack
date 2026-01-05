# AI Coding Agents (agents)

Installs AI coding agent CLIs: Claude Code, Gemini CLI, and Codex CLI.

## Usage

Add this feature to your `devcontainer.json`:

```json
{
  "features": {
    "ghcr.io/gilmanlab/features/agents:1": {}
  }
}
```

**Note:** This feature requires Node.js/npm. Add the Node.js feature if not already present:

```json
{
  "features": {
    "ghcr.io/devcontainers/features/node:1": {},
    "ghcr.io/gilmanlab/features/agents:1": {}
  }
}
```

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `claudeCode` | boolean | `true` | Install Claude Code CLI |
| `claudeCodeVersion` | string | `latest` | Version of Claude Code |
| `geminiCli` | boolean | `true` | Install Gemini CLI |
| `geminiCliVersion` | string | `latest` | Version of Gemini CLI |
| `codexCli` | boolean | `true` | Install Codex CLI |
| `codexCliVersion` | string | `latest` | Version of Codex CLI |

## Examples

### Install all agents (default)

```json
{
  "features": {
    "ghcr.io/gilmanlab/features/agents:1": {}
  }
}
```

### Install only Claude Code

```json
{
  "features": {
    "ghcr.io/gilmanlab/features/agents:1": {
      "geminiCli": false,
      "codexCli": false
    }
  }
}
```

### Install specific versions

```json
{
  "features": {
    "ghcr.io/gilmanlab/features/agents:1": {
      "claudeCodeVersion": "2.0.76",
      "geminiCliVersion": "0.22.5",
      "codexCliVersion": "0.77.0"
    }
  }
}
```

## Installed CLIs

| CLI | Package | Binary |
|-----|---------|--------|
| Claude Code | `@anthropic-ai/claude-code` | `claude` |
| Gemini CLI | `@google/gemini-cli` | `gemini` |
| Codex CLI | `@openai/codex` | `codex` |
