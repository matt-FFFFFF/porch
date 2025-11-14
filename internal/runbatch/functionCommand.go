// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"errors"
	"fmt"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
)

var _ Runnable = (*FunctionCommand)(nil)

// ErrFunctionCmdPanic is the error returned when a function command panics.
// It is constructed with the value that caused the panic.
type ErrFunctionCmdPanic struct {
	v any
}

// Error implements the error interface for ErrFunctionCmdPanic.
func (e *ErrFunctionCmdPanic) Error() string {
	prefix := "function command panic: "

	switch x := e.v.(type) {
	case string:
		return fmt.Sprintf("%s %s", prefix, x)
	case error:
		return fmt.Sprintf("%s %s", prefix, x.Error())
	default:
		return fmt.Sprintf("%s %v", prefix, x)
	}
}

var (
	// ErrSkipIntentional is returned to intentionally skip the remaining batch execution.
	ErrSkipIntentional = errors.New("intentionally skip execution")
	// ErrSkipOnError is returned to intentionally skip the remaining batch execution.
	ErrSkipOnError = errors.New("skip execution due to previous error")
)

// NewErrFunctionCmdPanic creates a new ErrFunctionCmdPanic with the given value.
func NewErrFunctionCmdPanic(v any) error {
	return &ErrFunctionCmdPanic{v: v}
}

// FunctionCommand is a command that runs a function. It implements the Runnable interface.
type FunctionCommand struct {
	*BaseCommand
	Func FunctionCommandFunc // The function to run
}

// FunctionCommandFunc is the type of the function that can be run by FunctionCommand.
// It takes a context, a string (the working directory), and a variadic number of strings (arguments).
type FunctionCommandFunc func(ctx context.Context, workingDirectory string, args ...string) FunctionCommandReturn

// FunctionCommandReturn is the return type of the function run by FunctionCommand.
type FunctionCommandReturn struct {
	NewCwd string // The new working directory, if changed
	Err    error  // Any error that occurred during execution
}

// Run implements the Runnable interface for FunctionCommand.
func (f *FunctionCommand) Run(ctx context.Context) Results {
	fullLabel := FullLabel(f)
	logger := ctxlog.Logger(ctx)
	logger = logger.With("runnableType", "functionCommand").
		With("label", fullLabel)

	// Return success immediately if function is nil
	if f.Func == nil {
		logger.Debug("No function to run, returning success")
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
				// Log the panic
				logger.Error("Function command panicked", "panic", r)

				var err error
				switch x := r.(type) {
				case error:
					err = errors.Join(NewErrFunctionCmdPanic(x), err)
				default:
					err = NewErrFunctionCmdPanic(x)
				}

				// Check if we're done before sending to avoid "send on closed channel"
				select {
				case <-done:
					// Already done, don't send on frCh
					logger.Debug("Function command panic done channel closed, skipping result send")
				default:
					logger.Debug("Function command panic sending error", "error", err)

					frCh <- FunctionCommandReturn{
						Err: err,
					}
				}
			}
		}()

		logger.Info(fmt.Sprintf("Executing: %s", fullLabel))

		// Run the function
		fr := f.Func(ctx, f.GetCwd())

		logger.Debug("Function command completed", "resultErr", fr.Err, "newCwd", fr.NewCwd)

		// Check if we're done before sending to avoid "send on closed channel"
		select {
		case <-done:
			logger.Debug("Function command done channel closed, skipping result send")
			// Already done, don't send on frCh
		default:
			logger.Debug("Function command sending result", "result", fr)

			frCh <- fr
		}
	}()

	res := &Result{
		Label:    f.Label,
		ExitCode: 0,
		Status:   ResultStatusSuccess,
		Cwd:      f.cwd,
		Type:     f.GetType(),
	}

	// Wait for either the function to complete or the context to be cancelled
	select {
	case fr := <-frCh:
		logger.Debug("Function command result received", "error", fr.Err, "newCwd", fr.NewCwd)

		if fr.Err != nil {
			return Results{
				{
					Label:    f.Label,
					ExitCode: -1,
					Error:    fr.Err,
					Status:   ResultStatusError,
				},
			}
		}

		// No error, set new working directory if provided
		if fr.NewCwd != "" {
			res.newCwd = fr.NewCwd
		}

	case <-ctx.Done():
		logger.Debug("Function command context cancelled", "error", ctx.Err())

		return Results{
			{
				Label:    f.Label,
				ExitCode: -1,
				Error:    ctx.Err(),
				Status:   ResultStatusError,
			},
		}
	}

	logger.Debug("Function command completed successfully", "newCwd", res.newCwd)

	return Results{res}
}

// GetType returns the type of the runnable (e.g., "Command", "SerialBatch", "ParallelBatch", etc.).
func (f *FunctionCommand) GetType() string {
	return "FunctionCommand"
}
