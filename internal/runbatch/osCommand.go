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

	"github.com/matt-FFFFFF/avmtool/internal/ctxlog"
	"github.com/matt-FFFFFF/avmtool/internal/signalbroker"
)

const (
	maxBufferSize = 8 * 1024 * 1024 // 8MB
)

var _ Runnable = (*OSCommand)(nil)

var (
	ErrBufferOverflow       = fmt.Errorf("output exceeds max size of %d bytes", maxBufferSize)
	ErrCouldNotStartProcess = errors.New("could not start process")
	ErrCouldNotKillProcess  = errors.New("could not kill process after timeout")
	ErrFailedToReadBuffer   = errors.New("failed to read buffer")
	ErrTimeoutExceeded      = errors.New("timeout exceeded")
	ErrFailedToCreatePipe   = errors.New("failed to create pipe")
	ErrSignalReceived       = errors.New("signal received")
)

// OSCommand represents a single command to be run in the batch.
type OSCommand struct {
	Path  string            // The command to run (e.g., executable name)
	Cwd   string            // Working directory for the command, if empty then use previous working directory
	Args  []string          // Arguments to the command, do not include the executable name itself
	Env   map[string]string // Environment variables
	Label string            // Optional label or description
	sigCh chan os.Signal    // Channel to receive signals
}

// GetLabel returns the label of the command (to satisfy Runnable interface)
func (c *OSCommand) GetLabel() string {
	return c.Label
}

// SetCwd sets the working directory for the command
func (c *OSCommand) SetCwd(cwd string) {
	c.Cwd = cwd
}

func (c *OSCommand) Run(ctx context.Context) Results {
	logger := ctxlog.Logger(ctx).
		With("runnableType", "OSCommand").
		With("label", c.Label).
		With("path", c.Path).
		With("args", c.Args).
		With("cwd", c.Cwd).
		With("env", c.Env)

	if c.sigCh == nil {
		c.sigCh = signalbroker.New(ctx)
	}

	res := &Result{
		Label:    c.Label,
		ExitCode: 0,
	}

	env := os.Environ()
	for k, v := range c.Env {
		logger.Debug("runbatch", "detail", "adding environment variable", "key", k, "value", v)
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

	logger.Debug("runbatch", "detail", "starting process")
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

	logger = logger.With("pid", ps.Pid)

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
					logger.Info("runbatch", "detail", "received duplicate signal, killing process", "signal", s.String())
					fmt.Fprintf(wErr, "received duplicate signal, killing process: %s\n", s.String())
					killPs(ctx, wasKilled, ps)
					return
				}
				signalCount[s] = struct{}{}
				logger.Info("runbatch", "detail", "received signal", "signal", s.String())
				fmt.Fprintf(wErr, "received signal: %s\n", s.String())
				ps.Signal(s)
			case <-ctx.Done():
				logger.Info("runbatch", "detail", "context done, killing process")
				fmt.Fprintln(wErr, "context done, killing process")
				killPs(ctx, wasKilled, ps)
				return
			case <-done:
				return
			}
		}
	}()

	// Wait for the process to finish and close the pipes
	logger.Debug("runbatch", "detail", "waiting for process to finish")
	state, psErr := ps.Wait()
	logger.Debug("runbatch", "detail", "process finished", "exitCode", res.ExitCode)

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

	logger.Debug("runbatch", "detail", "read stdout")
	stdout, err := readAllUpToMax(ctx, rOut, maxBufferSize)
	logger.Debug("runbatch", "detail", "stdout length", "bytes", len(stdout), "maxBytes", maxBufferSize)
	res.StdOut = stdout
	if err != nil {
		res.Error = errors.Join(res.Error, err)
		res.ExitCode = -1
	}

	logger.Debug("runbatch", "detail", "read stderr")
	stderr, err := readAllUpToMax(ctx, rErr, maxBufferSize)
	logger.Debug("runbatch", "detail", "stderr length", "bytes", len(stderr), "maxBytes", maxBufferSize)
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
		return nil, err
	}
	if n > maxBufferSize {
		ctxlog.Logger(ctx).Debug("runbatch", "detail", "buffer overflow in readAllUpToMax", "bytesRead", n, "maxBytes", maxBufferSize)
		return buf.Bytes()[:maxBufferSize], ErrBufferOverflow
	}
	return buf.Bytes(), nil
}

// killPs kills the process and closes the notification channel
func killPs(ctx context.Context, ch chan struct{}, ps *os.Process) {
	if err := ps.Kill(); err != nil {
		ctxlog.Logger(ctx).Debug("process kill error", "pid", ps.Pid, "error", err)
	}
	close(ch)
	ctxlog.Logger(ctx).Info("process killed", "pid", ps.Pid)
}
