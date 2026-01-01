package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// claudeAccountName is the storage key for Claude credentials.
const claudeAccountName = "claude-oidc-token"

// ansiEscape matches ANSI escape sequences.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[()][AB012]`)

// ClaudeProvider authenticates with Claude Code CLI.
type ClaudeProvider struct{}

// NewClaudeProvider creates a new Claude authentication provider.
func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{}
}

// Authenticate runs `claude setup-token` interactively and stores the OAuth token.
func (p *ClaudeProvider) Authenticate(ctx context.Context, storage Storage) error {
	// Check if claude CLI is available
	if _, err := exec.LookPath("claude"); err != nil {
		return errors.New("claude CLI not found in PATH: please install Claude Code first")
	}

	cmd := exec.CommandContext(ctx, "claude", "setup-token")

	// Start the command with a PTY so Ink gets the TTY it needs
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("start pty: %w", err)
	}
	defer ptmx.Close()

	// Handle PTY size changes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			_ = pty.InheritSize(os.Stdin, ptmx) //nolint:errcheck // Best-effort resize
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize
	defer func() { signal.Stop(ch); close(ch) }()

	// Set stdin in raw mode so we pass through all keystrokes
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("set raw mode: %w", err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	// Buffer to capture output for token extraction
	var outputBuf bytes.Buffer

	// Copy stdin to the pty (user input)
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin) //nolint:errcheck // Best-effort stdin forwarding
	}()

	// Copy pty output to both stdout (display) and our buffer (capture)
	_, _ = io.Copy(io.MultiWriter(os.Stdout, &outputBuf), ptmx) //nolint:errcheck // EOF expected

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude exited with error: %w", err)
	}

	// Extract token from captured output
	token := extractToken(outputBuf.String())
	if token == "" {
		return errors.New("no token received from claude setup-token")
	}

	// Store the token
	if err := storage.Set(claudeAccountName, token); err != nil {
		return fmt.Errorf("store token: %w", err)
	}

	return nil
}

// Get retrieves the stored Claude OAuth token.
func (p *ClaudeProvider) Get(storage Storage) (string, error) {
	return storage.Get(claudeAccountName)
}

// extractToken searches the output for a Claude OAuth token.
func extractToken(output string) string {
	// Strip ANSI escape codes
	clean := ansiEscape.ReplaceAllString(output, "")

	// Normalize line endings (PTY may use \r\n or just \r)
	clean = strings.ReplaceAll(clean, "\r\n", "\n")
	clean = strings.ReplaceAll(clean, "\r", "\n")

	lines := strings.Split(clean, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isClaudeToken(trimmed) {
			return trimmed
		}
	}
	return ""
}

// isClaudeToken checks if a string looks like a Claude OAuth token.
// Claude tokens have the format: sk-ant-oat01-...
func isClaudeToken(s string) bool {
	return strings.HasPrefix(s, "sk-ant-")
}
