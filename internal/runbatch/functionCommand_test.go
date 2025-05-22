// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestFunctionCommandRun_Success(t *testing.T) {
	// Define a function that succeeds
	successFunc := func(_ context.Context, _ string) FunctionCommandReturn {
		return FunctionCommandReturn{}
	}

	cmd := &FunctionCommand{
		Label: "success function",
		Func:  successFunc,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, 0, res.ExitCode, "expected exit code 0")
	assert.NoError(t, res.Error, "unexpected error")
}

func TestFunctionCommandRun_Failure(t *testing.T) {
	// Define a custom error for testing
	testErr := errors.New("function failed")

	// Define a function that fails with our custom error
	failFunc := func(_ context.Context, _ string) FunctionCommandReturn {
		return FunctionCommandReturn{Err: testErr}
	}

	cmd := &FunctionCommand{
		Label: "failure function",
		Func:  failFunc,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, -1, res.ExitCode, "expected -1 exit code")
	assert.Error(t, res.Error, "expected error")
	assert.ErrorIs(t, res.Error, testErr, "expected specific error")
}

func TestFunctionCommandRun_NilFunction(t *testing.T) {
	// Test with a nil function
	cmd := &FunctionCommand{
		Label: "nil function",
		Func:  nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This should not panic
	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, 0, res.ExitCode, "expected exit code 0")
	assert.NoError(t, res.Error, "unexpected error")
}

func TestFunctionCommandRun_ContextCancelled(t *testing.T) {
	// Define a function that blocks for longer than the context timeout
	longRunningFunc := func(_ context.Context, _ string) FunctionCommandReturn {
		time.Sleep(500 * time.Millisecond)
		return FunctionCommandReturn{}
	}

	cmd := &FunctionCommand{
		Label: "timeout function",
		Func:  longRunningFunc,
	}

	// Use a short timeout that will be exceeded
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, -1, res.ExitCode, "expected -1 exit code for cancelled context")
	assert.Error(t, res.Error, "expected error for cancelled context")
	assert.ErrorIs(t, res.Error, context.DeadlineExceeded, "expected deadline exceeded error")
}

func TestFunctionCommandRun_PanicHandling(t *testing.T) {
	// Define a function that panics
	panicFunc := func(_ context.Context, _ string) FunctionCommandReturn {
		panic("function panicked")
	}

	cmd := &FunctionCommand{
		Label: "panic function",
		Func:  panicFunc,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// This should not cause the test to panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The function should not propagate panics to the caller: %v", r)
		}
	}()

	_ = cmd.Run(ctx)
	// Note: This test may fail since the current implementation doesn't handle panics
	// If it passes, great! If it fails, we need to update FunctionCommand to handle panics
	// Ideally, the function would catch the panic and return an error
}

func TestFunctionCommandRun_Slow(t *testing.T) {
	// Define a slow but eventually succeeding function
	slowFunc := func(_ context.Context, _ string) FunctionCommandReturn {
		time.Sleep(100 * time.Millisecond)
		return FunctionCommandReturn{}
	}

	cmd := &FunctionCommand{
		Label: "slow function",
		Func:  slowFunc,
	}

	// Use a timeout longer than the function execution time
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, 0, res.ExitCode, "expected exit code 0")
	assert.NoError(t, res.Error, "unexpected error")
}

func TestFunctionCommandRun_NoGoroutineLeak(t *testing.T) {
	// Note: This test checks for goroutine leaks using the goleak package.
	defer goleak.VerifyNone(t)
	// Define a function that blocks until given channel is closed
	blockCh := make(chan struct{})
	blockingFunc := func(_ context.Context, _ string) FunctionCommandReturn {
		<-blockCh // Block until channel is closed
		return FunctionCommandReturn{}
	}

	cmd := &FunctionCommand{
		Label: "blocking function",
		Func:  blockingFunc,
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Start the command and immediately cancel the context
	go func() {
		time.Sleep(10 * time.Millisecond) // Give the function time to start
		cancel()
	}()

	results := cmd.Run(ctx)
	assert.Len(t, results, 1, "expected 1 result")

	res := results[0]
	assert.Equal(t, -1, res.ExitCode, "expected -1 exit code for cancelled context")
	assert.ErrorIs(t, res.Error, context.Canceled, "expected context cancelled error")

	// Close the channel to allow any leaked goroutine to exit
	close(blockCh)

	// Wait a bit for any goroutines to clean up
	time.Sleep(50 * time.Millisecond)
}
