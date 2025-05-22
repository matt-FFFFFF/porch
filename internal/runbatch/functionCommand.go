// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"errors"
	"fmt"
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

// NewErrFunctionCmdPanic creates a new ErrFunctionCmdPanic with the given value.§.
func NewErrFunctionCmdPanic(v any) error {
	return &ErrFunctionCmdPanic{v: v}
}

// FunctionCommand is a command that runs a function. It implements the Runnable interface.
type FunctionCommand struct {
	Label string
	Cwd   string
	// The function to run, the string parameter is the working directory
	Func func(context.Context, string) FunctionCommandReturn
}

// FunctionCommandReturn is the return type of the function run by FunctionCommand.
type FunctionCommandReturn struct {
	NewCwd string
	Err    error
}

// GetLabel returns the label of the command (to satisfy Runnable interface).
func (f *FunctionCommand) GetLabel() string {
	return f.Label
}

// SetCwd sets the working directory for the command.
func (f *FunctionCommand) SetCwd(cwd string) {
	f.Cwd = cwd
}

// Run implements the Runnable interface for FunctionCommand.
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
				case error:
					err = errors.Join(NewErrFunctionCmdPanic(x), err)
				default:
					err = NewErrFunctionCmdPanic(x)
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
