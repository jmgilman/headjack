package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// TeeWriter wraps an io.Writer to also write to a log file.
// It implements io.WriteCloser.
type TeeWriter struct {
	primary io.Writer
	logFile *os.File
	mu      sync.Mutex
}

// NewTeeWriter creates a TeeWriter that writes to both the primary writer
// and the specified log file path. The log file is created or truncated.
func NewTeeWriter(primary io.Writer, logPath string) (*TeeWriter, error) {
	//nolint:gosec // G304: logPath is constructed from trusted PathManager, not arbitrary user input
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	return &TeeWriter{
		primary: primary,
		logFile: logFile,
	}, nil
}

// NewTeeWriterAppend creates a TeeWriter that writes to both the primary writer
// and appends to the specified log file path.
func NewTeeWriterAppend(primary io.Writer, logPath string) (*TeeWriter, error) {
	//nolint:gosec // G302/G304: logPath is from trusted PathManager; 0644 needed for log rotation tools
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file for append: %w", err)
	}

	return &TeeWriter{
		primary: primary,
		logFile: logFile,
	}, nil
}

// Write writes data to both the primary writer and the log file.
// If the primary writer is nil, it only writes to the log file.
func (t *TeeWriter) Write(p []byte) (n int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Always write to log file first
	if t.logFile != nil {
		if _, err := t.logFile.Write(p); err != nil {
			return 0, fmt.Errorf("write to log file: %w", err)
		}
	}

	// Write to primary if available
	if t.primary != nil {
		return t.primary.Write(p)
	}

	return len(p), nil
}

// Close closes the log file. The primary writer is not closed.
func (t *TeeWriter) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.logFile != nil {
		if err := t.logFile.Close(); err != nil {
			return fmt.Errorf("close log file: %w", err)
		}
		t.logFile = nil
	}
	return nil
}

// Sync flushes the log file to disk.
func (t *TeeWriter) Sync() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.logFile != nil {
		return t.logFile.Sync()
	}
	return nil
}

// LogPath returns the path of the log file, or empty string if no log file.
func (t *TeeWriter) LogPath() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.logFile != nil {
		return t.logFile.Name()
	}
	return ""
}

// LogOnlyWriter creates a writer that only writes to the log file (no primary).
// Useful for detached sessions where there's no terminal output.
func LogOnlyWriter(logPath string) (*TeeWriter, error) {
	return NewTeeWriter(nil, logPath)
}

// LogOnlyWriterAppend creates a writer that appends to a log file (no primary).
func LogOnlyWriterAppend(logPath string) (*TeeWriter, error) {
	return NewTeeWriterAppend(nil, logPath)
}

// SessionWriters holds stdout and stderr tee writers for a session.
type SessionWriters struct {
	Stdout *TeeWriter
	Stderr *TeeWriter
}

// NewSessionWriters creates tee writers for both stdout and stderr
// that write to a single combined log file.
func NewSessionWriters(stdout, stderr io.Writer, logPath string) (*SessionWriters, error) {
	// Create a single log file that both stdout and stderr write to
	//nolint:gosec // G304: logPath is constructed from trusted PathManager, not arbitrary user input
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create session log file: %w", err)
	}

	return &SessionWriters{
		Stdout: &TeeWriter{primary: stdout, logFile: logFile},
		Stderr: &TeeWriter{primary: stderr, logFile: logFile},
	}, nil
}

// NewSessionWritersAppend creates tee writers that append to an existing log file.
func NewSessionWritersAppend(stdout, stderr io.Writer, logPath string) (*SessionWriters, error) {
	//nolint:gosec // G302/G304: logPath is from trusted PathManager; 0644 needed for log rotation tools
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open session log file for append: %w", err)
	}

	return &SessionWriters{
		Stdout: &TeeWriter{primary: stdout, logFile: logFile},
		Stderr: &TeeWriter{primary: stderr, logFile: logFile},
	}, nil
}

// Close closes both stdout and stderr writers.
// Only one close is needed since they share the same underlying log file.
func (s *SessionWriters) Close() error {
	// Only close stdout's log file since both share the same file
	if s.Stdout != nil {
		return s.Stdout.Close()
	}
	return nil
}

// Sync flushes both writers to disk.
func (s *SessionWriters) Sync() error {
	if s.Stdout != nil {
		return s.Stdout.Sync()
	}
	return nil
}
