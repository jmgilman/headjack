package multiplexer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/jmgilman/headjack/internal/exec"
)

// zellij implements Multiplexer using the Zellij terminal multiplexer.
type zellij struct {
	exec exec.Executor
}

// NewZellij creates a Multiplexer using Zellij CLI.
func NewZellij(e exec.Executor) Multiplexer {
	return &zellij{exec: e}
}

func (z *zellij) CreateSession(_ context.Context, _ *CreateSessionOpts) (*Session, error) {
	// Zellij does not support creating sessions in detached mode.
	// Unlike tmux (which supports `tmux new-session -d`), Zellij always
	// attempts to attach to the session it creates, requiring a TTY.
	// There is no reliable way to create a background/detached session.
	return nil, ErrDetachedModeNotSupported
}

func (z *zellij) AttachSession(ctx context.Context, sessionName string) error {
	// zellij attach <session-name>
	// Note: We don't use --create here since CreateSession handles creation
	args := []string{"attach", sessionName}

	stdinFd := int(os.Stdin.Fd())

	// Capture stderr while also streaming to os.Stderr for user visibility
	// This allows us to detect error messages like "session not found"
	var stderrBuf bytes.Buffer
	stderrWriter := io.MultiWriter(os.Stderr, &stderrBuf)

	// Check if stdin is a terminal
	if !term.IsTerminal(stdinFd) {
		// Fall back to non-interactive mode
		_, err := z.exec.Run(ctx, &exec.RunOptions{
			Name:   "zellij",
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: stderrWriter,
		})
		if err != nil {
			stderr := stderrBuf.String()
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
	defer func() { _ = term.Restore(stdinFd, oldState) }()

	// Handle window resize signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	// Run zellij with stdio attached
	_, err = z.exec.Run(ctx, &exec.RunOptions{
		Name:   "zellij",
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: stderrWriter,
	})
	if err != nil {
		stderr := stderrBuf.String()
		if strings.Contains(stderr, "not found") || strings.Contains(stderr, "No session") {
			return ErrSessionNotFound
		}
		return fmt.Errorf("%w: %v", ErrAttachFailed, err)
	}

	return nil
}

func (z *zellij) ListSessions(ctx context.Context) ([]Session, error) {
	// zellij list-sessions
	result, err := z.exec.Run(ctx, &exec.RunOptions{
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
	result, err := z.exec.Run(ctx, &exec.RunOptions{
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
