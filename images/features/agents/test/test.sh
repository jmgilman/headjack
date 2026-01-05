#!/bin/bash
# Test script for default scenario (all agents installed)
set -e

echo "Testing AI Coding Agents feature..."

# Verify Claude Code
if ! command -v claude > /dev/null 2>&1; then
    echo "FAIL: claude command not found"
    exit 1
fi
echo "PASS: claude command found"

# Verify Gemini CLI
if ! command -v gemini > /dev/null 2>&1; then
    echo "FAIL: gemini command not found"
    exit 1
fi
echo "PASS: gemini command found"

# Verify Codex CLI
if ! command -v codex > /dev/null 2>&1; then
    echo "FAIL: codex command not found"
    exit 1
fi
echo "PASS: codex command found"

echo "All tests passed!"
