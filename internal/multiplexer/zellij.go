package multiplexer

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/jmgilman/headjack/internal/exec"
)

// zellij implements Multiplexer using the Zellij terminal multiplexer.
type zellij struct {
	exec exec.Executor
}

// NewZellij creates a Multiplexer using Zellij CLI.
func NewZellij(e exec.Executor) *zellij {
	return &zellij{exec: e}
}

func (z *zellij) CreateSession(ctx context.Context, opts CreateSessionOpts) (*Session, error) {
	// Check if session already exists
	sessions, err := z.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("check existing sessions: %w", err)
	}

	for _, s := range sessions {
		if s.Name == opts.Name {
			return nil, ErrSessionExists
		}
	}

	// Build the zellij command for background execution
	// We use shell to start zellij in the background (detached mode)
	zellijCmd := fmt.Sprintf("zellij --session %s", shellEscape(opts.Name))

	if opts.Cwd != "" {
		zellijCmd += fmt.Sprintf(" --cwd %s", shellEscape(opts.Cwd))
	}

	// Start zellij in background using shell
	// The session will persist after the shell command returns
	shellCmd := fmt.Sprintf("%s &", zellijCmd)

	result, err := z.exec.Run(ctx, exec.RunOptions{
		Name: "sh",
		Args: []string{"-c", shellCmd},
	})
	if err != nil {
		stderr := string(result.Stderr)
		return nil, fmt.Errorf("%w: %s", ErrCreateFailed, stderr)
	}

	// Brief wait for session to initialize
	time.Sleep(100 * time.Millisecond)

	// Verify session was created
	sessions, err = z.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("verify session created: %w", err)
	}

	for _, s := range sessions {
		if s.Name == opts.Name {
			return &Session{
				ID:        s.Name,
				Name:      s.Name,
				CreatedAt: time.Now(),
			}, nil
		}
	}

	return nil, fmt.Errorf("%w: session not found after creation", ErrCreateFailed)
}

func (z *zellij) AttachSession(ctx context.Context, sessionName string) error {
	// zellij attach <session-name>
	// Note: We don't use --create here since CreateSession handles creation
	args := []string{"attach", sessionName}

	stdinFd := int(os.Stdin.Fd())

	// Check if stdin is a terminal
	if !term.IsTerminal(stdinFd) {
		// Fall back to non-interactive mode
		result, err := z.exec.Run(ctx, exec.RunOptions{
			Name:   "zellij",
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		})
		if err != nil {
			stderr := string(result.Stderr)
			if strings.Contains(stderr, "not found") || strings.Contains(stderr, "No session") {
				return ErrSessionNotFound
			}
			return fmt.Errorf("%w: %v", ErrAttachFailed, err)
		}
		return nil
	}

	// Put terminal in raw mode for proper TTY handling
	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return fmt.Errorf("set terminal raw mode: %w", err)
	}
	defer term.Restore(stdinFd, oldState)

	// Handle window resize signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	// Run zellij with stdio attached
	result, err := z.exec.Run(ctx, exec.RunOptions{
		Name:   "zellij",
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		stderr := ""
		if result != nil {
			stderr = string(result.Stderr)
		}
		if strings.Contains(stderr, "not found") || strings.Contains(stderr, "No session") {
			return ErrSessionNotFound
		}
		return fmt.Errorf("%w: %v", ErrAttachFailed, err)
	}

	return nil
}

func (z *zellij) ListSessions(ctx context.Context) ([]Session, error) {
	// zellij list-sessions
	result, err := z.exec.Run(ctx, exec.RunOptions{
		Name: "zellij",
		Args: []string{"list-sessions"},
	})
	if err != nil {
		// If zellij exits with error but has no sessions, that's ok
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "No active") || result.ExitCode == 0 {
			return []Session{}, nil
		}
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	// Parse output - each line is a session name
	// Format: "session-name [Created ...] (current)" or just "session-name"
	output := strings.TrimSpace(string(result.Stdout))
	if output == "" {
		return []Session{}, nil
	}

	lines := strings.Split(output, "\n")
	sessions := make([]Session, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract session name (first word before any brackets or parentheses)
		name := line
		if idx := strings.IndexAny(line, " \t[("); idx > 0 {
			name = line[:idx]
		}

		sessions = append(sessions, Session{
			ID:   name,
			Name: name,
			// CreatedAt is not reliably available from list output
		})
	}

	return sessions, nil
}

func (z *zellij) KillSession(ctx context.Context, sessionName string) error {
	// zellij kill-session <session-name>
	result, err := z.exec.Run(ctx, exec.RunOptions{
		Name: "zellij",
		Args: []string{"kill-session", sessionName},
	})
	if err != nil {
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "not found") || strings.Contains(stderr, "No session") {
			return ErrSessionNotFound
		}
		return fmt.Errorf("kill session: %w", err)
	}

	return nil
}

// shellEscape escapes a string for safe use in shell commands.
func shellEscape(s string) string {
	// Use single quotes and escape any single quotes within
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
