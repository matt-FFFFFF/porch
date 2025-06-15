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
	"strings"
	"time"

	"github.com/matt-FFFFFF/porch/internal/color"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/signalbroker"
	"github.com/matt-FFFFFF/porch/internal/teereader"
)

const (
	maxBufferSize        = 8 * 1024 * 1024 // 8MB
	maxLastLineLength    = 120
	defaultTickerSeconds = 10 // Default ticker interval for process status updates
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

// SetCleanup sets the cleanup function to be called after the command finishes.
func (c *OSCommand) SetCleanup(fn func(ctx context.Context)) {
	if c == nil {
		return
	}

	c.cleanup = fn
}

// Run implements the Runnable interface for OSCommand.
func (c *OSCommand) Run(ctx context.Context) Results {
	fullLabel := FullLabel(c)
	logger := ctxlog.Logger(ctx)
	logger = logger.With("runnableType", "OSCommand").
		With("label", fullLabel)

	logger.Debug("command info", "path", c.Path, "cwd", c.Cwd, "args", c.Args)

	tickerInterval := defaultTickerSeconds * time.Second // Interval for the process watchdog ticker

	var logCh chan string

	if logInt := ctx.Value(ProgressiveLogChannelKey{}); logInt != nil {
		if v, ok := logInt.(chan string); ok {
			logCh = v
			defer close(logCh) // Ensure the channel is closed when done
		}
	}

	if updateInt := ctx.Value(ProgressiveLogUpdateInterval{}); updateInt != nil {
		if v, ok := updateInt.(time.Duration); ok {
			tickerInterval = v
		}
	}

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

	// store start time to display progress later
	startTime := time.Now()

	logger.Info(fmt.Sprintf("Starting %s", fullLabel))

	if err != nil {
		res.Error = errors.Join(ErrCouldNotStartProcess, err)
		res.ExitCode = -1
		res.Status = ResultStatusError

		return Results{res}
	}

	logger.Debug("process started", "pid", ps.Pid)

	// Create teereader for stdout to capture last line while preserving all output
	stdoutTeeReader := teereader.NewLastLineTeeReader(rOut)

	// Start a goroutine to continuously read stdout through the teereader
	stdoutDone := make(chan struct{})
	go func() {
		defer close(stdoutDone)
		// Read all data through the teereader to capture it
		_, err := io.Copy(io.Discard, stdoutTeeReader)
		if err != nil && err != io.EOF {
			logger.Debug("error reading stdout through teereader", "error", err)
		}
	}()

	// Simple buffered channel to track kill reasons
	killReason := make(chan error, 1)

	// Channel to signal watchdog to stop - prevents goroutine leak
	done := make(chan struct{})

	// watchdog for process signals and context cancellation
	go func() {
		signalCount := make(map[os.Signal]struct{})

		ticker := time.NewTicker(tickerInterval)
		defer ticker.Stop()

		var lastLogSent string

		for {
			select {
			case <-done:
				return

			case <-ticker.C:
				diff := time.Since(startTime)
				diff = diff.Round(time.Second) // Round to the nearest second for display

				// Format the ticker status message
				lastLine := stdoutTeeReader.GetLastLine(maxLastLineLength)
				sb := strings.Builder{}
				sb.WriteString("Running ")
				sb.WriteString(fullLabel)
				sb.WriteString(": [")
				sb.WriteString(diff.String())
				sb.WriteString("]")

				if lastLine != "" {
					sb.WriteString(". Last output...\n")
					sb.WriteString(color.Colorize("=> ", color.FgGreen))
					sb.WriteString(lastLine)
				}

				msg := sb.String()

				if logCh != nil && lastLine != lastLogSent {
					logCh <- msg // Send the status message to the log channel
				}

				logger.Info(msg)

			case s := <-c.sigCh:
				// is this the second signal received of this type?
				if _, ok := signalCount[s]; ok {
					logger.Debug("received duplicate signal, killing process", "signal", s.String())
					fmt.Fprintf(wErr, "received duplicate signal, killing process: %s\n", s.String()) //nolint:errcheck
					killPs(ctx, ps)

					// Try to send reason (non-blocking)
					select {
					case killReason <- ErrDuplicateSignalReceived:
					default: // Channel full, that's fine
					}

					return
				}

				signalCount[s] = struct{}{}

				logger.Debug("received signal", "signal", s.String())
				fmt.Fprintf(wErr, "received signal: %s\n", s.String()) //nolint:errcheck

				if err := ps.Signal(s); err != nil {
					logger.Debug("failed to send signal", "signal", s.String(), "error", err)
				}

				// Try to send reason (non-blocking)
				select {
				case killReason <- ErrSignalReceived:
				default: // Channel full, that's fine
				}

			case <-ctx.Done():
				logger.Debug("context done, killing process")
				killPs(ctx, ps)

				// Try to send reason (non-blocking)
				select {
				case killReason <- ErrTimeoutExceeded:
				default: // Channel full, that's fine
				}

				return
			}
		}
	}()

	// Wait for the process to finish and close the pipes
	logger.Debug("waiting for process to finish")

	state, psErr := ps.Wait()

	executionTime := time.Since(startTime)
	executionTime = executionTime.Round(time.Second)
	logger.Info(fmt.Sprintf("Finished %s in %s", fullLabel, executionTime))

	// Signal watchdog to stop and prevent goroutine leak
	close(done)

	// Process is now definitively done - safe to close pipes and check kill reason
	_ = wOut.Close()
	_ = wErr.Close()

	// Wait for stdout reading to complete
	<-stdoutDone

	res.ExitCode = state.ExitCode()
	res.Error = psErr
	res.Status = ResultStatusUnknown

	logger.Debug("process finished", "exitCode", res.ExitCode)

	// Check if the process was killed due to timeout or signal (non-blocking)
	select {
	case killErr := <-killReason:
		logger.Debug("received signal that process was killed", "reason", killErr)
		res.Error = errors.Join(res.Error, killErr)
		res.ExitCode = -1
		res.Status = ResultStatusError
	default: // No kill reason, process completed naturally
		logger.Debug("did not receive signal that process was killed")
	}

	switch {
	// Exit code is success and error is nil or intentional skip. Return success.
	case slices.Contains(c.SuccessExitCodes, res.ExitCode) && res.Error == nil:
		logger.Debug("process exit code indicates success", "exitCode", res.ExitCode, "successCodes", c.SuccessExitCodes)
		res.Status = ResultStatusSuccess
	// Exit code is skippable and error is nil. Return success.
	case slices.Contains(c.SkipExitCodes, res.ExitCode) && res.Error == nil:
		logger.Debug("process exit code indicates skip remaining tasks",
			"exitCode", res.ExitCode, "skipCodes", c.SkipExitCodes)

		res.Error = ErrSkipIntentional
		res.Status = ResultStatusSuccess
	// Exit code is not successful or process error is not nil. Return error.
	// A non-zero exit code does not generate an error, so this needs to be an OR.
	case res.Error != nil || !slices.Contains(c.SuccessExitCodes, res.ExitCode):
		logger.Debug("process error", "error", res.Error, "exitCode", res.ExitCode, "successCodes", c.SuccessExitCodes)

		if res.ExitCode == 0 {
			res.ExitCode = -1 // If exit code is 0 but there is an error, set exit code to -1
		}

		res.Status = ResultStatusError
	}

	logger.Debug("read stdout")

	stdoutReader := stdoutTeeReader.GetFullBufferReader()
	logger.Debug("stdout length", "bytes", stdoutReader.Len(), "maxBytes", maxBufferSize)
	stdout, err := readAllUpToMax(ctx, stdoutReader, maxBufferSize)

	if err != nil {
		res.ExitCode = -1
		res.Error = errors.Join(res.Error, err)
	}

	logger.Debug("read stderr")

	stderr, err := readAllUpToMax(ctx, rErr, maxBufferSize)
	logger.Debug("stderr length", "bytes", len(stderr), "maxBytes", maxBufferSize)

	if err != nil {
		res.ExitCode = -1
		res.Error = errors.Join(res.Error, err)
	}

	res.StdOut = stdout
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
