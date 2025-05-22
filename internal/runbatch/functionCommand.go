package runbatch

import (
	"context"
	"fmt"
)

var _ Runnable = (*FunctionCommand)(nil)

// FunctionCommand is a command that runs a function. It implements the Runnable interface.
type FunctionCommand struct {
	Label string
	Cwd   string
	Func  func(context.Context, string) FunctionCommandReturn // The function to run, the string parameter is the working directory
}

type FunctionCommandReturn struct {
	NewCwd string
	Err    error
}

func (f *FunctionCommand) GetLabel() string {
	return f.Label
}

func (f *FunctionCommand) SetCwd(cwd string) {
	f.Cwd = cwd
}

func (f *FunctionCommand) Run(ctx context.Context) Results {
	// Return success immediately if function is nil
	if f.Func == nil {
		return Results{{Label: f.Label, ExitCode: 0, Error: nil}}
	}

	frCh := make(chan FunctionCommandReturn, 1)
	defer close(frCh) // Ensure the channel is closed after use

	done := make(chan struct{})
	defer close(done) // Signal the goroutine to stop if still running

	// Run the function in a goroutine and handle potential panics
	go func() {
		// Recover from panics and convert them to errors
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch x := r.(type) {
				case string:
					err = fmt.Errorf("panic: %s", x)
				case error:
					err = fmt.Errorf("panic: %w", x)
				default:
					err = fmt.Errorf("panic: %v", r)
				}

				// Check if we're done before sending to avoid "send on closed channel"
				select {
				case <-done:
					// Already done, don't send on frCh
				default:
					frCh <- FunctionCommandReturn{
						Err: err,
					}
				}
			}
		}()

		// Run the function
		fr := f.Func(ctx, f.Cwd)

		// Check if we're done before sending to avoid "send on closed channel"
		select {
		case <-done:
			// Already done, don't send on frCh
		default:
			frCh <- fr
		}
	}()

	res := &Result{
		Label:    f.Label,
		ExitCode: 0,
	}
	// Wait for either the function to complete or the context to be cancelled
	select {
	case fr := <-frCh:
		if fr.Err != nil {
			return Results{{Label: f.Label, ExitCode: -1, Error: fr.Err}}
		}

		if fr.NewCwd != "" {
			res.newCwd = fr.NewCwd
		}
	case <-ctx.Done():
		return Results{{Label: f.Label, ExitCode: -1, Error: ctx.Err()}}
	}

	return Results{res}
}
