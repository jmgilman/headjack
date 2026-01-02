package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// codexAccountName is the storage key for Codex credentials.
const codexAccountName = "codex-oauth-creds"

// codexConfigDir is the path where Codex CLI stores its configuration.
var codexConfigDir = filepath.Join(os.Getenv("HOME"), ".codex")

// CodexProvider authenticates with OpenAI Codex CLI.
type CodexProvider struct{}

// NewCodexProvider creates a new Codex authentication provider.
func NewCodexProvider() *CodexProvider {
	return &CodexProvider{}
}

// Authenticate runs `codex login` interactively and stores the auth.json contents.
func (p *CodexProvider) Authenticate(ctx context.Context, storage Storage) error {
	// Check if codex CLI is available
	if _, err := exec.LookPath("codex"); err != nil {
		return errors.New("codex CLI not found in PATH: please install OpenAI Codex CLI first")
	}

	// Run codex login interactively
	if err := runCodexLogin(ctx); err != nil {
		return err
	}

	// Read and store the auth.json file
	authData, err := readCodexAuth()
	if err != nil {
		return err
	}

	if err := storage.Set(codexAccountName, string(authData)); err != nil {
		return fmt.Errorf("store credentials: %w", err)
	}

	return nil
}

// Get retrieves the stored Codex credentials.
func (p *CodexProvider) Get(storage Storage) (string, error) {
	return storage.Get(codexAccountName)
}

// runCodexLogin executes the codex login command interactively.
func runCodexLogin(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "codex", "login")

	// Start the command with a PTY for interactive login
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

	// Copy stdin to the pty (user input)
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin) //nolint:errcheck // Best-effort stdin forwarding
	}()

	// Copy pty output to stdout (display)
	_, _ = io.Copy(os.Stdout, ptmx) //nolint:errcheck // EOF expected

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("codex login exited with error: %w", err)
	}

	return nil
}

// readCodexAuth reads the auth.json file from the Codex config directory.
func readCodexAuth() ([]byte, error) {
	authPath := filepath.Join(codexConfigDir, "auth.json")
	data, err := os.ReadFile(authPath) //nolint:gosec // Path is constructed from HOME env var
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("codex auth.json not found: login may have failed")
		}
		return nil, fmt.Errorf("read auth.json: %w", err)
	}

	if len(data) == 0 {
		return nil, errors.New("codex auth.json is empty: login may have failed")
	}

	return data, nil
}
