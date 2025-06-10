// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/signalbroker"
)

const (
	maxBufferSize  = 8 * 1024 * 1024  // 8MB
	tickerInterval = 10 * time.Second // Interval for the process watchdog ticker
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
	// ErrDuplicateSignalReceived is returned when a duplicate signal is received, forcing process termination.
	ErrDuplicateSignalReceived = errors.New("duplicate signal received, process forcefully terminated")
)

// OSCommand represents a single command to be run in the batch.
type OSCommand struct {
	*BaseCommand
	Args             []string                  // Arguments to the command, do not include the executable name itself.
	Path             string                    // The command to run (e.g. executable full path).
	SuccessExitCodes []int                     // Exit codes that indicate success, defaults to 0.
	SkipExitCodes    []int                     // Exit codes that indicate skip remaining tasks, defaults to empty.
	cleanup          func(ctx context.Context) // Cleanup function to run after the command finishes.
	sigCh            chan os.Signal            // Channel to receive signals, allows mocking in test.
}

func (c *OSCommand) SetCleanup(fn func(ctx context.Context)) {
	if c == nil {
		return
	}
	c.cleanup = fn
}

// Run implements the Runnable interface for OSCommand.
func (c *OSCommand) Run(ctx context.Context) Results {
	logger := ctxlog.Logger(ctx)
	logger = logger.With("runnableType", "OSCommand").
		With("label", c.Label)

	logger.Debug("command info", "path", c.Path, "cwd", c.Cwd, "args", c.Args)

	if c.SuccessExitCodes == nil {
		c.SuccessExitCodes = []int{0} // Default to success on exit code 0
	}

	if c.SkipExitCodes == nil {
		c.SkipExitCodes = []int{} // Default to no skip exit codes
	}

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
	startTime := time.Now()
	startTimeStr := startTime.Format(ctxlog.TimeFormat)

	fullLabel := FullLabel(c)
	fmt.Printf("Starting %s: at %s\n", fullLabel, startTimeStr)

	if err != nil {
		res.Error = errors.Join(ErrCouldNotStartProcess, err)
		res.ExitCode = -1
		res.Status = ResultStatusError

		return Results{res}
	}

	logger.Debug("process started", "pid", ps.Pid)

	// This is the process watchdog that will kill the process if it exceeds the timeout
	// or pass on any signals to the process.
	done := make(chan struct{})
	// This allows us to track why the processes was killed.
	wasKilled := make(chan error)

	// watchdog for process signals and context cancellation
	go func() {
		signalCount := make(map[os.Signal]struct{})

		ticker := time.NewTicker(tickerInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				diff := time.Since(startTime)
				diff = diff.Round(time.Second) // Round to the nearest second for display
				fmt.Printf("Running %s: [%s]...\n", fullLabel, diff)

			case s := <-c.sigCh:
				// is this the second signal received of this type?
				if _, ok := signalCount[s]; ok {
					logger.Info("received duplicate signal, killing process", "signal", s.String())
					fmt.Fprintf(wErr, "received duplicate signal, killing process: %s\n", s.String()) //nolint:errcheck
					killPs(ctx, ps)

					// Send error for duplicate signal (different from first signal)
					select {
					case wasKilled <- ErrDuplicateSignalReceived:
					case <-done:
						// Channel was closed, process already finished
					}

					return
				}

				signalCount[s] = struct{}{}

				logger.Info("received signal", "signal", s.String())
				fmt.Fprintf(wErr, "received signal: %s\n", s.String()) //nolint:errcheck

				if err := ps.Signal(s); err != nil {
					logger.Info("failed to send signal", "signal", s.String(), "error", err)
				}

				select {
				case wasKilled <- ErrSignalReceived:
				case <-done:
					// Channel was closed, process already finished
				}

			case <-ctx.Done():
				logger.Info("context done, killing process")
				fmt.Fprintln(wErr, "context done, killing process") //nolint:errcheck
				killPs(ctx, ps)

				select {
				case wasKilled <- ErrTimeoutExceeded:
				case <-done:
					// Channel was closed, process already finished
				}

				return

			case <-done:
				return
			}
		}
	}()

	// Wait for the process to finish and close the pipes
	logger.Debug("waiting for process to finish")

	state, psErr := ps.Wait()

	fmt.Printf("Finished %s: at %s\n", fullLabel, time.Now().Format(ctxlog.TimeFormat))

	_ = wOut.Close()
	_ = wErr.Close()
	res.ExitCode = state.ExitCode()
	res.Error = psErr
	res.Status = ResultStatusUnknown

	logger.Debug("process finished", "exitCode", res.ExitCode)

	// Check if the process was killed due to timeout or signal
	select {
	case e := <-wasKilled:
		res.Error = errors.Join(res.Error, e)
		res.ExitCode = -1
		res.Status = ResultStatusError
	default:
		// No error from watchdog, process completed normally
	}

	close(done)

	// Close wasKilled channel after signaling done to prevent race condition
	select {
	case <-wasKilled:
		// Already received an error from watchdog
	default:
		close(wasKilled)
	}

	switch {
	// Exit code is success and error is nil or intentional skip. Return success.
	case slices.Contains(c.SuccessExitCodes, res.ExitCode) && res.Error == nil:
		logger.Debug("process exit code indicates success", "exitCode", res.ExitCode)
		res.Status = ResultStatusSuccess
	// Exit code is skippable and error is nil. Return success.
	case slices.Contains(c.SkipExitCodes, res.ExitCode) && res.Error == nil:
		logger.Debug("process exit code indicates skip remaining tasks", "exitCode", res.ExitCode)
		res.Error = ErrSkipIntentional
		res.Status = ResultStatusSuccess
	// Exit code is not successful or process error is not nil. Return error.
	// A non-zero exit code does not generate an error, so this needs to be an OR.
	case res.Error != nil || !slices.Contains(c.SuccessExitCodes, res.ExitCode):
		logger.Debug("process error", "error", res.Error, "exitCode", res.ExitCode)

		if res.ExitCode == 0 {
			res.ExitCode = -1 // If exit code is 0 but there is an error, set exit code to -1
		}

		res.Status = ResultStatusError
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

	if c.cleanup != nil {
		logger.Debug("running cleanup function")
		c.cleanup(ctx)
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
func killPs(ctx context.Context, ps *os.Process) {
	if err := ps.Kill(); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			ctxlog.Logger(ctx).Debug("process already done", "pid", ps.Pid)
			return
		}

		ctxlog.Logger(ctx).Error("process kill error", "pid", ps.Pid, "error", err)
	}

	ctxlog.Logger(ctx).Info("process killed", "pid", ps.Pid)
}
