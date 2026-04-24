package github

import (
	"bytes"
	"context"
	"os/exec"
)

// Executor runs an external command and returns stdout, stderr, and the exec
// error separately. gh mixes JSON on stdout with human-readable messages on
// stderr, so we keep them split instead of using CombinedOutput.
type Executor interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

type execRunner struct{}

// DefaultExecutor returns an Executor backed by os/exec.
func DefaultExecutor() Executor { return execRunner{} }

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
