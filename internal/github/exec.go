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

// EnvExecutor is an Executor that also accepts a caller-provided env list.
// When env is non-nil it fully replaces the child process environment —
// callers should pass append(os.Environ(), "GH_TOKEN=…") so they don't lose
// PATH and friends. An empty env falls through to Run (ambient env).
type EnvExecutor interface {
	Executor
	RunEnv(ctx context.Context, env []string, name string, args ...string) (stdout, stderr []byte, err error)
}

type execRunner struct{}

// DefaultExecutor returns an Executor backed by os/exec.
func DefaultExecutor() EnvExecutor { return execRunner{} }

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	return execRunner{}.RunEnv(ctx, nil, name, args...)
}

func (execRunner) RunEnv(ctx context.Context, env []string, name string, args ...string) ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if env != nil {
		cmd.Env = env
	}
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
