// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/matt-FFFFFF/avmtool/internal/ctxlog"
	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
	"github.com/matt-FFFFFF/avmtool/internal/signalbroker"
)

// signal interrupts with the runbatch package.
func main() {
	// Create a signal broker that listens for interrupt and termination signals
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) //nolint:mnd
	ctxlog.New(ctx, ctxlog.DefaultLogger)
	ctxlog.LevelVar.Set(slog.LevelDebug)

	defer cancel()

	sigCh := signalbroker.New(ctx)

	go signalbroker.Watch(ctx, sigCh, cancel)

	// Create a batch with a mix of command types to demonstrate
	// how both OS processes and function commands can be gracefully
	// shut down with our signal handling approach
	batch := createDemoBatch()

	fmt.Println("=== Signal Handling Demo ===")
	fmt.Println("1. Press Ctrl+C once to gracefully cancel all processes")
	fmt.Println("2. Press Ctrl+C twice to forcefully terminate")
	fmt.Println("3. Wait 30 seconds for auto-timeout")
	fmt.Println("Running commands...")

	// Run the batch with our context and signal channel
	results := batch.Run(ctx)

	// Display the results
	fmt.Println("\n=== Results ===")

	options := runbatch.DefaultOutputOptions()
	options.IncludeStdOut = true
	options.IncludeStdErr = true
	results.PrintWithOptions(options)
}

// and function commands to demonstrate handling both types.
func createDemoBatch() *runbatch.SerialBatch {
	return &runbatch.SerialBatch{
		Label: "Signal Handling Demo",
		Commands: []runbatch.Runnable{
			// A simple OS command that completes quickly
			&runbatch.OSCommand{
				Path:  "/bin/sh",
				Args:  []string{"-c", "echo Starting demo..."},
				Label: "Echo Start",
			},
			// A parallel batch with multiple commands
			&runbatch.ParallelBatch{
				Label: "Parallel Commands",
				Commands: []runbatch.Runnable{
					// A long-running command that will be interrupted
					&runbatch.OSCommand{
						Path:  "/bin/sleep",
						Args:  []string{"30"},
						Label: "Long Sleep",
					},
					// A function command that checks for context cancellation
					&runbatch.FunctionCommand{
						Label: "Cancellable Function",
						Func: func(ctx context.Context, _ string) runbatch.FunctionCommandReturn {
							ticker := time.NewTicker(1 * time.Second)
							defer ticker.Stop()
							count := 0

							fmt.Println("Function running...")

							for {
								select {
								case <-ticker.C:
									count++
									fmt.Printf("Function tick %d\n", count)
									if count >= 60 { //nolint:mnd
										fmt.Println("Function completed naturally")
										return runbatch.FunctionCommandReturn{}
									}
								case <-ctx.Done():
									fmt.Println("Function cancelled")
									return runbatch.FunctionCommandReturn{
										Err: fmt.Errorf("function cancelled"), //nolint:err113
									}
								}
							}
						},
					},
				},
			},
			// This command should never run if we interrupt
			&runbatch.OSCommand{
				Path:  "/bin/sh",
				Args:  []string{"-c", "echo This should only print if no interruption occurred"},
				Label: "Final Echo",
			},
		},
	}
}
