package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeeWriter_Write(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	primary := &bytes.Buffer{}
	tw, err := NewTeeWriter(primary, logPath)
	require.NoError(t, err)
	defer tw.Close()

	// Write some data
	n, err := tw.Write([]byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, 11, n)

	// Verify primary received data
	assert.Equal(t, "hello world", primary.String())

	// Verify log file received data
	tw.Close()
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestTeeWriter_WriteMultiple(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	primary := &bytes.Buffer{}
	tw, err := NewTeeWriter(primary, logPath)
	require.NoError(t, err)

	// Write multiple times
	tw.Write([]byte("first\n"))
	tw.Write([]byte("second\n"))
	tw.Write([]byte("third\n"))

	tw.Close()

	// Verify both destinations
	assert.Equal(t, "first\nsecond\nthird\n", primary.String())

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "first\nsecond\nthird\n", string(data))
}

func TestTeeWriter_NilPrimary(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	tw, err := NewTeeWriter(nil, logPath)
	require.NoError(t, err)
	defer tw.Close()

	// Write should succeed even with nil primary
	n, err := tw.Write([]byte("log only"))
	require.NoError(t, err)
	assert.Equal(t, 8, n)

	tw.Close()
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "log only", string(data))
}

func TestTeeWriterAppend(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	// Create initial file with content
	err := os.WriteFile(logPath, []byte("existing\n"), 0644)
	require.NoError(t, err)

	// Open in append mode
	primary := &bytes.Buffer{}
	tw, err := NewTeeWriterAppend(primary, logPath)
	require.NoError(t, err)

	tw.Write([]byte("appended\n"))
	tw.Close()

	// Verify append worked
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "existing\nappended\n", string(data))
}

func TestTeeWriter_LogPath(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	tw, err := NewTeeWriter(&bytes.Buffer{}, logPath)
	require.NoError(t, err)

	assert.Equal(t, logPath, tw.LogPath())

	tw.Close()
	assert.Equal(t, "", tw.LogPath())
}

func TestTeeWriter_Sync(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	tw, err := NewTeeWriter(&bytes.Buffer{}, logPath)
	require.NoError(t, err)
	defer tw.Close()

	tw.Write([]byte("data"))
	err = tw.Sync()
	require.NoError(t, err)
}

func TestLogOnlyWriter(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	tw, err := LogOnlyWriter(logPath)
	require.NoError(t, err)
	defer tw.Close()

	n, err := tw.Write([]byte("log only content"))
	require.NoError(t, err)
	assert.Equal(t, 16, n)

	tw.Close()
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "log only content", string(data))
}

func TestSessionWriters(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "session.log")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sw, err := NewSessionWriters(stdout, stderr, logPath)
	require.NoError(t, err)

	// Write to stdout
	sw.Stdout.Write([]byte("stdout line\n"))

	// Write to stderr
	sw.Stderr.Write([]byte("stderr line\n"))

	// Both should appear in primary writers
	assert.Equal(t, "stdout line\n", stdout.String())
	assert.Equal(t, "stderr line\n", stderr.String())

	sw.Close()

	// Both should appear in log file (interleaved)
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "stdout line\nstderr line\n", string(data))
}

func TestSessionWritersAppend(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "session.log")

	// Create initial file
	err := os.WriteFile(logPath, []byte("previous\n"), 0644)
	require.NoError(t, err)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sw, err := NewSessionWritersAppend(stdout, stderr, logPath)
	require.NoError(t, err)

	sw.Stdout.Write([]byte("new stdout\n"))
	sw.Stderr.Write([]byte("new stderr\n"))
	sw.Close()

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "previous\nnew stdout\nnew stderr\n", string(data))
}

func TestSessionWriters_Sync(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "session.log")

	sw, err := NewSessionWriters(&bytes.Buffer{}, &bytes.Buffer{}, logPath)
	require.NoError(t, err)
	defer sw.Close()

	sw.Stdout.Write([]byte("data"))
	err = sw.Sync()
	require.NoError(t, err)
}
