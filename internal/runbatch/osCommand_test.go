package runbatch

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCommandRun_Success(t *testing.T) {
	cmd := &OSCommand{
		Path:  "/bin/echo",
		Args:  []string{"hello"},
		Env:   map[string]string{"FOO": "BAR"},
		Label: "echo test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results := cmd.Run(ctx, nil)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, 0, res.ExitCode, "expected exit code 0")
	assert.NoError(t, res.Error, "unexpected error")
	assert.Contains(t, string(res.StdOut), "hello", "expected stdout to contain 'hello'")
}

func TestCommandRun_Failure(t *testing.T) {
	cmd := &OSCommand{
		Path:  "/bin/sh",
		Args:  []string{"-c", "exit 1"},
		Label: "fail test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results := cmd.Run(ctx, nil)
	assert.Len(t, results, 1, "expected 1 result")
	res := results[0]
	assert.Equal(t, 1, res.ExitCode, "expected 1 exit code")
}

func TestCommandRun_NotFound(t *testing.T) {
	cmd := &OSCommand{
		Path:  "/not/a/real/command",
		Args:  []string{""},
		Label: "notfound test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results := cmd.Run(ctx, nil)
	assert.Len(t, results, 1, "expected 1 result")
	res := results[0]
	var notFoundErr *os.PathError
	assert.ErrorAs(t, res.Error, &notFoundErr, "expected PathError")
	assert.ErrorIs(t, res.Error, ErrCouldNotStartProcess, "expected error to be ErrCouldNotStartProcess")
}

func TestCommandRun_EnvAndCwd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping cwd/env test on windows")
	}
	tempDir := t.TempDir()
	cmd := &OSCommand{
		Path:  "/bin/sh",
		Args:  []string{"-c", "echo $FOO; pwd"},
		Env:   map[string]string{"FOO": "BAR"},
		Cwd:   tempDir,
		Label: "env and cwd test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results := cmd.Run(ctx, nil)
	assert.Len(t, results, 1, "expected 1 result")
	res := results[0]
	assert.Equal(t, 0, res.ExitCode, "expected exit code 0")
	out := string(res.StdOut)
	assert.Contains(t, out, "BAR", "expected stdout to contain 'BAR'")
	assert.Contains(t, out, tempDir, "expected stdout to contain tempDir")
}

func TestCommandRun_ContextCancelled(t *testing.T) {
	cmd := &OSCommand{
		Path:  "/bin/sleep",
		Args:  []string{"10"},
		Label: "sleep test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	results := cmd.Run(ctx, nil)
	assert.Len(t, results, 1, "expected 1 result")
	res := results[0]
	assert.Equal(t, -1, res.ExitCode, "expected -1 exit code for killed process")
	assert.Error(t, res.Error, "expected error for killed process, got nil")
	assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded, "expected context to be done, but it was not")
	assert.ErrorIs(t, res.Error, ErrTimeoutExceeded, "expected error to be ErrTimeoutExceeded")
	assert.ErrorIs(t, res.Error, ErrSignalReceived, "expected error to be ErrSignalReceived")
	assert.Contains(t, string(res.StdErr), "killed", "expected stderr to mention killed")
}

func TestCommandRun_SigInt(t *testing.T) {
	cmd := &OSCommand{
		Path:  "/bin/sleep",
		Args:  []string{"10"},
		Label: "sleep test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Second)
	defer cancel()
	sig := make(chan os.Signal, 1)
	go func() {
		time.Sleep(1 * time.Second)
		sig <- os.Interrupt
	}()
	results := cmd.Run(ctx, sig)
	assert.Len(t, results, 1, "expected 1 result")
	res := results[0]
	assert.Equal(t, -1, res.ExitCode, "expected -1 exit code for sigint process")
	assert.NoError(t, ctx.Err(), "expected context to be unclosed")
	assert.ErrorIs(t, res.Error, ErrSignalReceived, "expected error to be ErrSignalReceived")
	assert.Contains(t, string(res.StdErr), "interrupt", "expected stderr to mention interrupt")
}
