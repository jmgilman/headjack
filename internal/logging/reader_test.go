package logging

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestLog(t *testing.T, dir, instanceID, sessionID string, lines []string) string {
	t.Helper()
	pm := NewPathManager(dir)
	path, err := pm.EnsureSessionLog(instanceID, sessionID)
	require.NoError(t, err)

	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	err = os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)

	return path
}

func TestReader_ReadAll(t *testing.T) {
	dir := t.TempDir()
	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	createTestLog(t, dir, "inst1", "sess1", lines)

	pm := NewPathManager(dir)
	reader := NewReader(pm)

	result, err := reader.ReadAll("inst1", "sess1")
	require.NoError(t, err)
	assert.Equal(t, lines, result)
}

func TestReader_ReadAll_Empty(t *testing.T) {
	dir := t.TempDir()
	createTestLog(t, dir, "inst1", "sess1", []string{})

	pm := NewPathManager(dir)
	reader := NewReader(pm)

	result, err := reader.ReadAll("inst1", "sess1")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestReader_ReadAll_NotFound(t *testing.T) {
	dir := t.TempDir()
	pm := NewPathManager(dir)
	reader := NewReader(pm)

	_, err := reader.ReadAll("inst1", "nonexistent")
	assert.Error(t, err)
}

func TestReader_ReadLastN(t *testing.T) {
	dir := t.TempDir()
	lines := []string{"line1", "line2", "line3", "line4", "line5", "line6", "line7", "line8", "line9", "line10"}
	createTestLog(t, dir, "inst1", "sess1", lines)

	pm := NewPathManager(dir)
	reader := NewReader(pm)

	t.Run("last 3 lines", func(t *testing.T) {
		result, err := reader.ReadLastN("inst1", "sess1", 3)
		require.NoError(t, err)
		assert.Equal(t, []string{"line8", "line9", "line10"}, result)
	})

	t.Run("last 5 lines", func(t *testing.T) {
		result, err := reader.ReadLastN("inst1", "sess1", 5)
		require.NoError(t, err)
		assert.Equal(t, []string{"line6", "line7", "line8", "line9", "line10"}, result)
	})

	t.Run("request more than available", func(t *testing.T) {
		result, err := reader.ReadLastN("inst1", "sess1", 100)
		require.NoError(t, err)
		assert.Equal(t, lines, result)
	})

	t.Run("default when n <= 0", func(t *testing.T) {
		// With only 10 lines and default of 100, should return all
		result, err := reader.ReadLastN("inst1", "sess1", 0)
		require.NoError(t, err)
		assert.Equal(t, lines, result)
	})
}

func TestReader_ReadLastN_FewerThanN(t *testing.T) {
	dir := t.TempDir()
	lines := []string{"only", "three", "lines"}
	createTestLog(t, dir, "inst1", "sess1", lines)

	pm := NewPathManager(dir)
	reader := NewReader(pm)

	result, err := reader.ReadLastN("inst1", "sess1", 10)
	require.NoError(t, err)
	assert.Equal(t, lines, result)
}

func TestReader_ReadLastN_Empty(t *testing.T) {
	dir := t.TempDir()
	createTestLog(t, dir, "inst1", "sess1", []string{})

	pm := NewPathManager(dir)
	reader := NewReader(pm)

	result, err := reader.ReadLastN("inst1", "sess1", 10)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestReader_Follow(t *testing.T) {
	dir := t.TempDir()
	pm := NewPathManager(dir)

	// Create initial log file
	logPath, err := pm.EnsureSessionLog("inst1", "sess1")
	require.NoError(t, err)

	logFile, err := os.Create(logPath)
	require.NoError(t, err)

	reader := NewReader(pm)
	output := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start following in a goroutine
	done := make(chan error)
	go func() {
		done <- reader.Follow(ctx, "inst1", "sess1", output, 50*time.Millisecond)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Append some data
	logFile.WriteString("new line 1\n")
	logFile.WriteString("new line 2\n")
	logFile.Sync()

	// Wait for follow to finish (via context timeout)
	err = <-done
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Verify output contains the new lines
	assert.Contains(t, output.String(), "new line 1\n")
	assert.Contains(t, output.String(), "new line 2\n")

	logFile.Close()
}

func TestReader_Follow_CancelledContext(t *testing.T) {
	dir := t.TempDir()
	createTestLog(t, dir, "inst1", "sess1", []string{"initial"})

	pm := NewPathManager(dir)
	reader := NewReader(pm)
	output := &bytes.Buffer{}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error)
	go func() {
		done <- reader.Follow(ctx, "inst1", "sess1", output, 50*time.Millisecond)
	}()

	// Cancel after a short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	err := <-done
	assert.ErrorIs(t, err, context.Canceled)
}

func TestReader_FollowWithHistory(t *testing.T) {
	dir := t.TempDir()
	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	logPath := createTestLog(t, dir, "inst1", "sess1", lines)

	pm := NewPathManager(dir)
	reader := NewReader(pm)
	output := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	done := make(chan error)
	go func() {
		done <- reader.FollowWithHistory(ctx, "inst1", "sess1", output, 3, 50*time.Millisecond)
	}()

	// Give it time to read history
	time.Sleep(100 * time.Millisecond)

	// Append new content
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	f.WriteString("line6\n")
	f.Sync()
	f.Close()

	// Wait for follow to finish
	<-done

	result := output.String()
	// Should contain last 3 lines from history
	assert.Contains(t, result, "line3\n")
	assert.Contains(t, result, "line4\n")
	assert.Contains(t, result, "line5\n")
	// Should contain new line
	assert.Contains(t, result, "line6\n")
	// Should not contain earlier history
	assert.NotContains(t, result, "line1\n")
	assert.NotContains(t, result, "line2\n")
}

func TestReader_Follow_PartialLines(t *testing.T) {
	dir := t.TempDir()
	pm := NewPathManager(dir)

	// Create initial log file
	logPath, err := pm.EnsureSessionLog("inst1", "sess1")
	require.NoError(t, err)

	logFile, err := os.Create(logPath)
	require.NoError(t, err)

	reader := NewReader(pm)
	output := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start following in a goroutine
	done := make(chan error)
	go func() {
		done <- reader.Follow(ctx, "inst1", "sess1", output, 50*time.Millisecond)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Write a partial line (no trailing newline)
	logFile.WriteString("partial")
	logFile.Sync()

	// Wait a bit for the poll to pick it up
	time.Sleep(100 * time.Millisecond)

	// Complete the line
	logFile.WriteString(" complete\n")
	logFile.Sync()

	// Write another line
	logFile.WriteString("next line\n")
	logFile.Sync()

	// Wait for follow to finish (via context timeout)
	err = <-done
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Verify output contains ALL data including the partial line
	result := output.String()
	assert.Contains(t, result, "partial")
	assert.Contains(t, result, "complete")
	assert.Contains(t, result, "next line")

	logFile.Close()
}

func TestReadLastNLines_LargeFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "large.log")

	// Create a file with 1000 lines
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, strings.Repeat("x", 100)) // 100 char lines
	}
	content := strings.Join(lines, "\n") + "\n"
	err := os.WriteFile(logPath, []byte(content), 0644)
	require.NoError(t, err)

	result, err := readLastNLines(logPath, 10)
	require.NoError(t, err)
	assert.Len(t, result, 10)
	for _, line := range result {
		assert.Equal(t, strings.Repeat("x", 100), line)
	}
}
