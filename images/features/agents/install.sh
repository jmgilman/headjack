#!/bin/bash
# AI Coding Agents Feature - install.sh
# Installs Claude Code, Gemini CLI, and Codex CLI via npm
set -e

# Feature options (devcontainer converts option names to uppercase)
CLAUDE_CODE="${CLAUDECODE:-true}"
CLAUDE_CODE_VERSION="${CLAUDECODEVERSION:-latest}"
GEMINI_CLI="${GEMINICLI:-true}"
GEMINI_CLI_VERSION="${GEMINICLIVERSION:-latest}"
CODEX_CLI="${CODEXCLI:-true}"
CODEX_CLI_VERSION="${CODEXCLIVERSION:-latest}"

echo "Installing AI Coding Agent CLIs..."

# Check if npm is available
if ! command -v npm > /dev/null 2>&1; then
    echo "Error: npm is not available. Please ensure Node.js is installed."
    echo "Add the Node.js feature before this feature:"
    echo '  "ghcr.io/devcontainers/features/node:1": {}'
    exit 1
fi

echo "Using npm version: $(npm --version)"
echo "Using Node.js version: $(node --version)"

# Build list of packages to install
PACKAGES=""

if [ "${CLAUDE_CODE}" = "true" ]; then
    if [ "${CLAUDE_CODE_VERSION}" = "latest" ]; then
        PACKAGES="${PACKAGES} @anthropic-ai/claude-code"
    else
        PACKAGES="${PACKAGES} @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}"
    fi
    echo "Will install Claude Code: ${CLAUDE_CODE_VERSION}"
fi

if [ "${GEMINI_CLI}" = "true" ]; then
    if [ "${GEMINI_CLI_VERSION}" = "latest" ]; then
        PACKAGES="${PACKAGES} @google/gemini-cli"
    else
        PACKAGES="${PACKAGES} @google/gemini-cli@${GEMINI_CLI_VERSION}"
    fi
    echo "Will install Gemini CLI: ${GEMINI_CLI_VERSION}"
fi

if [ "${CODEX_CLI}" = "true" ]; then
    if [ "${CODEX_CLI_VERSION}" = "latest" ]; then
        PACKAGES="${PACKAGES} @openai/codex"
    else
        PACKAGES="${PACKAGES} @openai/codex@${CODEX_CLI_VERSION}"
    fi
    echo "Will install Codex CLI: ${CODEX_CLI_VERSION}"
fi

# Install packages if any are enabled
if [ -n "${PACKAGES}" ]; then
    echo "Installing:${PACKAGES}"
    # shellcheck disable=SC2086
    npm install -g ${PACKAGES}
    echo "Installation complete."
else
    echo "No agents selected for installation."
fi

# Verify installations
echo ""
echo "Verification:"
if [ "${CLAUDE_CODE}" = "true" ]; then
    if command -v claude > /dev/null 2>&1; then
        echo "  Claude Code: installed"
    else
        echo "  Claude Code: FAILED"
        exit 1
    fi
fi

if [ "${GEMINI_CLI}" = "true" ]; then
    if command -v gemini > /dev/null 2>&1; then
        echo "  Gemini CLI: installed"
    else
        echo "  Gemini CLI: FAILED"
        exit 1
    fi
fi

if [ "${CODEX_CLI}" = "true" ]; then
    if command -v codex > /dev/null 2>&1; then
        echo "  Codex CLI: installed"
    else
        echo "  Codex CLI: FAILED"
        exit 1
    fi
fi

echo ""
echo "AI Coding Agent CLIs installation finished."
