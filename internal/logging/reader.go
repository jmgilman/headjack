package logging

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

// DefaultTailLines is the default number of lines to read when tailing.
const DefaultTailLines = 100

// Reader provides functionality to read session log files.
type Reader struct {
	pathMgr *PathManager
}

// NewReader creates a new Reader with the given PathManager.
func NewReader(pathMgr *PathManager) *Reader {
	return &Reader{pathMgr: pathMgr}
}

// ReadAll reads the entire log file for a session.
func (r *Reader) ReadAll(instanceID, sessionID string) ([]string, error) {
	path := r.pathMgr.SessionLogPath(instanceID, sessionID)
	return readAllLines(path)
}

// ReadLastN reads the last n lines from a session's log file.
// If n <= 0, uses DefaultTailLines.
func (r *Reader) ReadLastN(instanceID, sessionID string, n int) ([]string, error) {
	if n <= 0 {
		n = DefaultTailLines
	}

	path := r.pathMgr.SessionLogPath(instanceID, sessionID)
	return readLastNLines(path, n)
}

// Follow streams new log lines to the provided writer as they are appended.
// This is similar to `tail -f`. It blocks until the context is cancelled.
// The pollInterval determines how frequently to check for new content.
func (r *Reader) Follow(ctx context.Context, instanceID, sessionID string, out io.Writer, pollInterval time.Duration) error {
	path := r.pathMgr.SessionLogPath(instanceID, sessionID)

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer file.Close()

	// Seek to end of file
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("seek to end: %w", err)
	}

	reader := bufio.NewReader(file)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for {
				line, err := reader.ReadBytes('\n')
				// Always write any data we received, even with EOF
				if len(line) > 0 {
					if _, werr := out.Write(line); werr != nil {
						return fmt.Errorf("write output: %w", werr)
					}
				}
				if err != nil {
					if err == io.EOF {
						// No more data, wait for next poll
						break
					}
					return fmt.Errorf("read line: %w", err)
				}
			}
		}
	}
}

// FollowWithHistory reads the last n lines and then follows new output.
// This is similar to `tail -n N -f`.
func (r *Reader) FollowWithHistory(ctx context.Context, instanceID, sessionID string, out io.Writer, n int, pollInterval time.Duration) error {
	// First, output the last N lines
	lines, err := r.ReadLastN(instanceID, sessionID, n)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return fmt.Errorf("write history: %w", err)
		}
	}

	// Then start following
	return r.Follow(ctx, instanceID, sessionID, out, pollInterval)
}

// readAllLines reads all lines from a file.
func readAllLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan log file: %w", err)
	}

	return lines, nil
}

// readLastNLines reads the last n lines from a file.
// Uses a ring buffer approach for efficiency with large files.
func readLastNLines(path string, n int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	defer file.Close()

	// Use a ring buffer to keep track of last n lines
	ring := make([]string, n)
	idx := 0
	count := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ring[idx] = scanner.Text()
		idx = (idx + 1) % n
		count++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan log file: %w", err)
	}

	// Build result in correct order
	if count == 0 {
		return nil, nil
	}

	if count < n {
		// Haven't filled the buffer yet
		return ring[:count], nil
	}

	// Buffer is full, need to reorder
	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = ring[(idx+i)%n]
	}
	return result, nil
}
