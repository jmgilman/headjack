#!/bin/bash
# Test script for claude-only scenario
set -e

echo "Testing Claude-only configuration..."

# Verify Claude Code is installed
if ! command -v claude > /dev/null 2>&1; then
    echo "FAIL: claude command not found"
    exit 1
fi
echo "PASS: claude command found"

# Verify Gemini CLI is NOT installed
if command -v gemini > /dev/null 2>&1; then
    echo "FAIL: gemini command should not be installed"
    exit 1
fi
echo "PASS: gemini command correctly not installed"

# Verify Codex CLI is NOT installed
if command -v codex > /dev/null 2>&1; then
    echo "FAIL: codex command should not be installed"
    exit 1
fi
echo "PASS: codex command correctly not installed"

echo "All tests passed!"
