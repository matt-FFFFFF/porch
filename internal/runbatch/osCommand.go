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
}

// GetLabel returns the label of the command (to satisfy Runnable interface)
func (c *OSCommand) GetLabel() string {
	return c.Label
}

// SetCwd sets the working directory for the command
func (c *OSCommand) SetCwd(cwd string) {
	c.Cwd = cwd
}

func (c *OSCommand) Run(ctx context.Context, sig <-chan os.Signal) Results {
	res := &Result{
		Label:    c.Label,
		ExitCode: 0,
	}

	env := os.Environ()
	for k, v := range c.Env {
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

	// This is the process watchdog that will kill the process if it exceeds the timeout
	// or pass on any signals to the process.
	done := make(chan struct{})
	killed := make(chan struct{})
	go func() {
		for {
			select {
			case s := <-sig:
				fmt.Fprintf(wErr, "signal received: %s\n", s)
				ps.Signal(s)
			case <-ctx.Done():
				close(killed)
				if err := ps.Kill(); err != nil {
					fmt.Fprintf(wErr, "failed to kill process after timeout %s: %s\n", c.Label, err)
					return
				}
				fmt.Fprintf(wErr, "timeout exceeded and process was killed: %s\n", c.Label)
				return
			case <-done:
				return
			}
		}
	}()

	// Wait for the process to finish and close the pipes
	state, psErr := ps.Wait()

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
	case <-killed:
		res.Error = errors.Join(res.Error, ErrTimeoutExceeded)
		res.ExitCode = -1
	default:
		close(killed)
	}

	stdout, err := readAllUpToMax(rOut, maxBufferSize)
	res.StdOut = stdout
	if err != nil {
		res.Error = errors.Join(res.Error, err)
		res.ExitCode = -1
	}

	stderr, err := readAllUpToMax(rErr, maxBufferSize)
	res.StdErr = stderr
	if err != nil {
		res.ExitCode = -1
		res.Error = errors.Join(res.Error, err)
	}

	return Results{res}
}

func readAllUpToMax(r io.Reader, maxBufferSize int64) ([]byte, error) {
	var buf bytes.Buffer
	n, err := io.CopyN(&buf, r, maxBufferSize+1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if n > maxBufferSize {
		return buf.Bytes()[:maxBufferSize], ErrBufferOverflow
	}
	return buf.Bytes(), nil
}
