package exec

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	e := New()
	require.NotNil(t, e)
}

func TestExecutor_Run(t *testing.T) {
	e := New()

	t.Run("captures stdout", func(t *testing.T) {
		result, err := e.Run(context.Background(), &RunOptions{
			Name: "echo",
			Args: []string{"hello"},
		})

		require.NoError(t, err)
		assert.Equal(t, "hello\n", string(result.Stdout))
		assert.Empty(t, result.Stderr)
		assert.Equal(t, 0, result.ExitCode)
	})

	t.Run("captures stderr", func(t *testing.T) {
		result, err := e.Run(context.Background(), &RunOptions{
			Name: "sh",
			Args: []string{"-c", "echo error >&2"},
		})

		require.NoError(t, err)
		assert.Empty(t, result.Stdout)
		assert.Equal(t, "error\n", string(result.Stderr))
		assert.Equal(t, 0, result.ExitCode)
	})

	t.Run("captures exit code on failure", func(t *testing.T) {
		result, err := e.Run(context.Background(), &RunOptions{
			Name: "sh",
			Args: []string{"-c", "exit 42"},
		})

		require.Error(t, err)
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 42, result.ExitCode)
	})

	t.Run("streams to provided stdout writer", func(t *testing.T) {
		var buf bytes.Buffer
		result, err := e.Run(context.Background(), &RunOptions{
			Name:   "echo",
			Args:   []string{"streamed"},
			Stdout: &buf,
		})

		require.NoError(t, err)
		assert.Nil(t, result.Stdout, "Stdout should be nil when streaming")
		assert.Equal(t, "streamed\n", buf.String())
	})

	t.Run("streams to provided stderr writer", func(t *testing.T) {
		var buf bytes.Buffer
		result, err := e.Run(context.Background(), &RunOptions{
			Name:   "sh",
			Args:   []string{"-c", "echo error >&2"},
			Stderr: &buf,
		})

		require.NoError(t, err)
		assert.Nil(t, result.Stderr, "Stderr should be nil when streaming")
		assert.Equal(t, "error\n", buf.String())
	})

	t.Run("respects working directory", func(t *testing.T) {
		result, err := e.Run(context.Background(), &RunOptions{
			Name: "pwd",
			Dir:  "/tmp",
		})

		require.NoError(t, err)
		// On macOS, /tmp is a symlink to /private/tmp
		assert.Contains(t, string(result.Stdout), "/tmp",
			"expected output to contain /tmp, got: %s", string(result.Stdout))
	})

	t.Run("passes environment variables", func(t *testing.T) {
		result, err := e.Run(context.Background(), &RunOptions{
			Name: "sh",
			Args: []string{"-c", "echo $TEST_VAR"},
			Env:  []string{"TEST_VAR=hello_env"},
		})

		require.NoError(t, err)
		assert.Equal(t, "hello_env\n", string(result.Stdout))
	})

	t.Run("reads from stdin", func(t *testing.T) {
		result, err := e.Run(context.Background(), &RunOptions{
			Name:  "cat",
			Stdin: strings.NewReader("input data"),
		})

		require.NoError(t, err)
		assert.Equal(t, "input data", string(result.Stdout))
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := e.Run(ctx, &RunOptions{
			Name: "sleep",
			Args: []string{"10"},
		})

		require.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "signal: killed"),
			"expected context deadline or killed signal, got: %v", err)
	})

	t.Run("returns error for nonexistent command", func(t *testing.T) {
		_, err := e.Run(context.Background(), &RunOptions{
			Name: "nonexistent_command_12345",
		})

		require.Error(t, err)
	})
}

func TestExecutor_LookPath(t *testing.T) {
	e := New()

	t.Run("finds existing command", func(t *testing.T) {
		path, err := e.LookPath("echo")

		require.NoError(t, err)
		assert.NotEmpty(t, path)
		assert.True(t, strings.HasSuffix(path, "echo") || strings.Contains(path, "echo"),
			"expected path to contain echo, got: %s", path)
	})

	t.Run("returns error for nonexistent command", func(t *testing.T) {
		_, err := e.LookPath("nonexistent_command_12345")

		require.Error(t, err)
		var execErr *exec.Error
		assert.ErrorAs(t, err, &execErr)
	})
}
