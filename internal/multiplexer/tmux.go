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

// tmux implements Multiplexer using the tmux terminal multiplexer.
type tmux struct {
	exec exec.Executor
}

// NewTmux creates a Multiplexer using tmux CLI.
func NewTmux(e exec.Executor) Multiplexer {
	return &tmux{exec: e}
}

func (t *tmux) CreateSession(ctx context.Context, opts *CreateSessionOpts) (*Session, error) {
	if opts == nil || opts.Name == "" {
		return nil, fmt.Errorf("%w: session name is required", ErrCreateFailed)
	}

	// Check if session already exists
	sessions, err := t.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("check existing sessions: %w", err)
	}
	for _, s := range sessions {
		if s.Name == opts.Name {
			return nil, ErrSessionExists
		}
	}

	// Build tmux new-session command
	// -d: detached mode (don't attach)
	// -s: session name
	args := []string{"new-session", "-d", "-s", opts.Name}

	// Set working directory if specified
	if opts.Cwd != "" {
		args = append(args, "-c", opts.Cwd)
	}

	// Add environment variables
	for _, env := range opts.Env {
		args = append(args, "-e", env)
	}

	// Add command if specified (must come last)
	if len(opts.Command) > 0 {
		args = append(args, opts.Command...)
	}

	// Create the session
	result, err := t.exec.Run(ctx, &exec.RunOptions{
		Name: "tmux",
		Args: args,
	})
	if err != nil {
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "duplicate session") {
			return nil, ErrSessionExists
		}
		return nil, fmt.Errorf("%w: %v", ErrCreateFailed, err)
	}

	// Set up log capture via pipe-pane if LogPath is specified
	if opts.LogPath != "" {
		// Shell-escape the path to handle spaces and special characters safely
		escapedPath := shellEscape(opts.LogPath)
		pipeArgs := []string{"pipe-pane", "-t", opts.Name, "cat >> " + escapedPath}
		// Log capture failure is non-fatal, session was still created
		//nolint:errcheck // best-effort log capture
		_, _ = t.exec.Run(ctx, &exec.RunOptions{
			Name: "tmux",
			Args: pipeArgs,
		})
	}

	return &Session{
		ID:   opts.Name,
		Name: opts.Name,
	}, nil
}

func (t *tmux) AttachSession(ctx context.Context, sessionName string) error {
	// tmux attach-session -t <session-name>
	args := []string{"attach-session", "-t", sessionName}

	stdinFd := int(os.Stdin.Fd())

	// Capture stderr while also streaming to os.Stderr for user visibility
	var stderrBuf bytes.Buffer
	stderrWriter := io.MultiWriter(os.Stderr, &stderrBuf)

	// Check if stdin is a terminal
	if !term.IsTerminal(stdinFd) {
		// Fall back to non-interactive mode
		_, err := t.exec.Run(ctx, &exec.RunOptions{
			Name:   "tmux",
			Args:   args,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: stderrWriter,
		})
		if err != nil {
			stderr := stderrBuf.String()
			if strings.Contains(stderr, "no session") || strings.Contains(stderr, "can't find session") {
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

	// Run tmux with stdio attached
	_, err = t.exec.Run(ctx, &exec.RunOptions{
		Name:   "tmux",
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: stderrWriter,
	})
	if err != nil {
		stderr := stderrBuf.String()
		if strings.Contains(stderr, "no session") || strings.Contains(stderr, "can't find session") {
			return ErrSessionNotFound
		}
		return fmt.Errorf("%w: %v", ErrAttachFailed, err)
	}

	return nil
}

func (t *tmux) ListSessions(ctx context.Context) ([]Session, error) {
	// tmux list-sessions -F "#{session_name}"
	result, err := t.exec.Run(ctx, &exec.RunOptions{
		Name: "tmux",
		Args: []string{"list-sessions", "-F", "#{session_name}"},
	})
	if err != nil {
		// Only treat known "no sessions" messages as empty list.
		// Other errors (even with exit code 1) should be surfaced.
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "no server running") ||
			strings.Contains(stderr, "no sessions") ||
			strings.Contains(stderr, "error connecting to") {
			return []Session{}, nil
		}
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	// Parse output - each line is a session name
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

		sessions = append(sessions, Session{
			ID:   line,
			Name: line,
		})
	}

	return sessions, nil
}

func (t *tmux) KillSession(ctx context.Context, sessionName string) error {
	// tmux kill-session -t <session-name>
	result, err := t.exec.Run(ctx, &exec.RunOptions{
		Name: "tmux",
		Args: []string{"kill-session", "-t", sessionName},
	})
	if err != nil {
		stderr := string(result.Stderr)
		if strings.Contains(stderr, "no session") || strings.Contains(stderr, "can't find session") {
			return ErrSessionNotFound
		}
		return fmt.Errorf("kill session: %w", err)
	}

	return nil
}

// shellEscape escapes a string for safe use in a shell command.
// It wraps the string in single quotes and escapes any embedded single quotes.
func shellEscape(s string) string {
	// Single quotes prevent all shell interpretation except for single quotes themselves.
	// To include a single quote, we end the quoted string, add an escaped single quote,
	// and start a new quoted string: 'foo'\''bar' -> foo'bar
	escaped := strings.ReplaceAll(s, "'", `'\''`)
	return "'" + escaped + "'"
}
