// Package logging provides session output logging infrastructure for Headjack.
package logging

import (
	"fmt"
	"os"
	"path/filepath"
)

// PathManager handles log file path construction and directory management.
type PathManager struct {
	baseDir string
}

// NewPathManager creates a new PathManager with the given base directory.
// The base directory is typically ~/.local/share/headjack/logs.
func NewPathManager(baseDir string) *PathManager {
	return &PathManager{baseDir: baseDir}
}

// BaseDir returns the base log directory.
func (p *PathManager) BaseDir() string {
	return p.baseDir
}

// InstanceDir returns the log directory for a specific instance.
// Path format: <baseDir>/<instanceID>/
func (p *PathManager) InstanceDir(instanceID string) string {
	return filepath.Join(p.baseDir, instanceID)
}

// SessionLogPath returns the full path for a session's log file.
// Path format: <baseDir>/<instanceID>/<sessionID>.log
func (p *PathManager) SessionLogPath(instanceID, sessionID string) string {
	return filepath.Join(p.baseDir, instanceID, sessionID+".log")
}

// EnsureInstanceDir creates the instance log directory if it doesn't exist.
// Returns the instance directory path.
func (p *PathManager) EnsureInstanceDir(instanceID string) (string, error) {
	dir := p.InstanceDir(instanceID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create instance log directory: %w", err)
	}
	return dir, nil
}

// EnsureSessionLog ensures the parent directory exists for a session log file.
// Returns the full log file path.
func (p *PathManager) EnsureSessionLog(instanceID, sessionID string) (string, error) {
	if _, err := p.EnsureInstanceDir(instanceID); err != nil {
		return "", err
	}
	return p.SessionLogPath(instanceID, sessionID), nil
}

// LogExists checks if a log file exists for the given session.
func (p *PathManager) LogExists(instanceID, sessionID string) bool {
	path := p.SessionLogPath(instanceID, sessionID)
	_, err := os.Stat(path)
	return err == nil
}

// RemoveSessionLog removes a session's log file if it exists.
func (p *PathManager) RemoveSessionLog(instanceID, sessionID string) error {
	path := p.SessionLogPath(instanceID, sessionID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove session log: %w", err)
	}
	return nil
}

// RemoveInstanceLogs removes all log files for an instance.
func (p *PathManager) RemoveInstanceLogs(instanceID string) error {
	dir := p.InstanceDir(instanceID)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove instance logs: %w", err)
	}
	return nil
}

// ListSessionLogs returns a list of session IDs that have log files for the given instance.
func (p *PathManager) ListSessionLogs(instanceID string) ([]string, error) {
	dir := p.InstanceDir(instanceID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read instance log directory: %w", err)
	}

	var sessions []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if ext := filepath.Ext(name); ext == ".log" {
			sessions = append(sessions, name[:len(name)-len(ext)])
		}
	}
	return sessions, nil
}
