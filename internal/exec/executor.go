package exec

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

type executor struct{}

// New returns a new Executor that uses os/exec.
func New() Executor {
	return &executor{}
}

func (e *executor) Run(ctx context.Context, opts *RunOptions) (*Result, error) {
	// G204: This is intentional - we're an executor that runs user-specified commands.
	// The caller is responsible for validating the command and arguments.
	cmd := exec.CommandContext(ctx, opts.Name, opts.Args...) //nolint:gosec // Intentional subprocess execution

	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	} else {
		cmd.Stdout = &stdoutBuf
	}

	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	} else {
		cmd.Stderr = &stderrBuf
	}

	err := cmd.Run()

	result := &Result{
		ExitCode: cmd.ProcessState.ExitCode(),
	}
	if opts.Stdout == nil {
		result.Stdout = stdoutBuf.Bytes()
	}
	if opts.Stderr == nil {
		result.Stderr = stderrBuf.Bytes()
	}

	return result, err
}

func (e *executor) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
