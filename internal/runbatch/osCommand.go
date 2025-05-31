// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/matt-FFFFFF/pporch/internal/ctxlog"
	"github.com/matt-FFFFFF/pporch/internal/signalbroker"
)

const (
	maxBufferSize = 8 * 1024 * 1024 // 8MB
)

var _ Runnable = (*OSCommand)(nil)

var (
	// ErrBufferOverflow is returned when the output exceeds the max size.
	ErrBufferOverflow = fmt.Errorf("output exceeds max size of %d bytes", maxBufferSize)
	// ErrCouldNotStartProcess is returned when the process could not be started.
	ErrCouldNotStartProcess = errors.New("could not start process")
	// ErrCouldNotKillProcess is returned when the process could not be killed.
	ErrCouldNotKillProcess = errors.New("could not kill process after timeout")
	// ErrFailedToReadBuffer is returned when the buffer from the operating system pipe could not be read.
	ErrFailedToReadBuffer = errors.New("failed to read buffer")
	// ErrTimeoutExceeded is returned when the command exceeds the context deadline.
	ErrTimeoutExceeded = errors.New("timeout exceeded")
	// ErrFailedToCreatePipe is returned when the operating system pipe could not be created.
	ErrFailedToCreatePipe = errors.New("failed to create pipe")
	// ErrSignalReceived is returned when a operating system signal is received by the child process.
	ErrSignalReceived = errors.New("signal received")
)

// OSCommand represents a single command to be run in the batch.
type OSCommand struct {
	Path  string            // The command to run (e.g., executable name).
	Cwd   string            // Working directory for the command, if empty then use previous working directory.
	Args  []string          // Arguments to the command, do not include the executable name itself.
	Env   map[string]string // Environment variables.
	Label string            // Optional label or description.
	sigCh chan os.Signal    // Channel to receive signals, allows mocking in test.
}

// SetCwd sets the working directory for the command.
func (c *OSCommand) SetCwd(cwd string) {
	c.Cwd = cwd
}

// InheritEnv sets the environment variables for the batch.
func (c *OSCommand) InheritEnv(env map[string]string) {
	if len(c.Env) == 0 {
		c.Env = maps.Clone(env)
		return
	}

	for k, v := range maps.All(env) {
		if _, ok := c.Env[k]; !ok {
			c.Env[k] = v
		}
	}
}

// Run implements the Runnable interface for OSCommand.
func (c *OSCommand) Run(ctx context.Context) Results {
	logger := ctxlog.Logger(ctx)
	logger = logger.With("runnableType", "OSCommand").
		With("label", c.Label)

	logger.Debug("command info", "path", c.Path, "cwd", c.Cwd, "args", c.Args)

	if c.sigCh == nil {
		c.sigCh = signalbroker.New(ctx)
	}

	res := &Result{
		Label:    c.Label,
		ExitCode: 0,
	}

	env := os.Environ()

	for k, v := range c.Env {
		logger.Debug("adding environment variable", "key", k, "value", v)
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	rOut, wOut, err := os.Pipe()
	if err != nil {
		res.Error = errors.Join(ErrFailedToCreatePipe, err)
		res.ExitCode = -1

		return Results{res}
	}

	rErr, wErr, err := os.Pipe()
	if err != nil {
		res.Error = errors.Join(ErrFailedToCreatePipe, err)
		res.ExitCode = -1

		return Results{res}
	}

	execName := filepath.Base(c.Path)
	args := slices.Concat([]string{execName}, c.Args)

	logger.Debug("starting process")

	ps, err := os.StartProcess(c.Path, args, &os.ProcAttr{
		Dir:   c.Cwd,
		Env:   env,
		Files: []*os.File{os.Stdin, wOut, wErr},
	})

	if err != nil {
		res.Error = errors.Join(ErrCouldNotStartProcess, err)
		res.ExitCode = -1

		return Results{res}
	}

	logger.Debug("process started", "pid", ps.Pid)

	// This is the process watchdog that will kill the process if it exceeds the timeout
	// or pass on any signals to the process.
	done := make(chan struct{})
	wasKilled := make(chan struct{})

	go func() {
		signalCount := make(map[os.Signal]struct{})

		for {
			select {
			case s := <-c.sigCh:
				// is this the second signal received of this type?
				if _, ok := signalCount[s]; ok {
					logger.Info("received duplicate signal, killing process", "signal", s.String())
					fmt.Fprintf(wErr, "received duplicate signal, killing process: %s\n", s.String()) //nolint:errcheck
					killPs(ctx, wasKilled, ps)

					return
				}

				signalCount[s] = struct{}{}

				logger.Info("received signal", "signal", s.String())
				fmt.Fprintf(wErr, "received signal: %s\n", s.String()) //nolint:errcheck

				if err := ps.Signal(s); err != nil {
					logger.Info("failed to send signal", "signal", s.String(), "error", err)
				}
			case <-ctx.Done():
				logger.Info("context done, killing process")
				fmt.Fprintln(wErr, "context done, killing process") //nolint:errcheck
				killPs(ctx, wasKilled, ps)

				return
			case <-done:
				return
			}
		}
	}()

	// Wait for the process to finish and close the pipes
	logger.Debug("waiting for process to finish")

	state, psErr := ps.Wait()

	logger.Debug("process finished", "exitCode", res.ExitCode)

	close(done)

	_ = wOut.Close()
	_ = wErr.Close()
	res.ExitCode = state.ExitCode()
	res.Error = psErr

	if res.ExitCode == -1 {
		res.Error = errors.Join(res.Error, ErrSignalReceived)
	}

	// Check if the process was killed due to timeout
	select {
	case <-wasKilled:
		res.Error = errors.Join(res.Error, ErrTimeoutExceeded)
		res.ExitCode = -1
	default:
		close(wasKilled)
	}

	logger.Debug("read stdout")

	stdout, err := readAllUpToMax(ctx, rOut, maxBufferSize)
	logger.Debug("stdout length", "bytes", len(stdout), "maxBytes", maxBufferSize)

	res.StdOut = stdout
	if err != nil {
		res.Error = errors.Join(res.Error, err)
		res.ExitCode = -1
	}

	logger.Debug("read stderr")

	stderr, err := readAllUpToMax(ctx, rErr, maxBufferSize)
	logger.Debug("stderr length", "bytes", len(stderr), "maxBytes", maxBufferSize)

	res.StdErr = stderr
	if err != nil {
		res.ExitCode = -1
		res.Error = errors.Join(res.Error, err)
	}

	return Results{res}
}

func readAllUpToMax(ctx context.Context, r io.Reader, maxBufferSize int64) ([]byte, error) {
	var buf bytes.Buffer

	n, err := io.CopyN(&buf, r, maxBufferSize+1)
	if err != nil && err != io.EOF {
		return nil, errors.Join(ErrFailedToReadBuffer, err)
	}

	if n > maxBufferSize {
		ctxlog.Logger(ctx).Debug(
			"buffer overflow in readAllUpToMax",
			"bytesRead", n,
			"maxBytes", maxBufferSize,
		)

		return buf.Bytes()[:maxBufferSize], ErrBufferOverflow
	}

	return buf.Bytes(), nil
}

// killPs kills the process and closes the notification channel.
func killPs(ctx context.Context, ch chan struct{}, ps *os.Process) {
	if err := ps.Kill(); err != nil {
		ctxlog.Logger(ctx).Debug("process kill error", "pid", ps.Pid, "error", err)
	}

	close(ch)
	ctxlog.Logger(ctx).Info("process killed", "pid", ps.Pid)
}
