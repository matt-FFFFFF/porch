package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
)

func main() {
	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a signal channel
	sig := make(chan os.Signal, 1)

	// Create a batch with nested commands
	batch := &runbatch.SerialBatch{
		Label: "Demo Batch",
		Commands: []runbatch.Runnable{
			&runbatch.OSCommand{
				Path:  "/bin/echo",
				Args:  []string{"This is a successful command"},
				Label: "Echo Command",
			},
			&runbatch.ParallelBatch{
				Label: "Parallel Commands",
				Commands: []runbatch.Runnable{
					&runbatch.OSCommand{
						Path:  "/bin/cat",
						Args:  []string{"/etc/hosts"},
						Label: "Cat Hosts",
					},
					&runbatch.OSCommand{
						Path:  "/bin/sh",
						Args:  []string{"-c", "echo 'This command will fail with exit code 1' && echo 'This is stderr line 1' >&2 && echo 'This is stderr line 2' >&2 && echo '  Indented stderr line' >&2 && exit 1"},
						Label: "Failing Command",
					},
				},
			},
			// Nested serial batch
			&runbatch.SerialBatch{
				Label: "Nested Batch",
				Commands: []runbatch.Runnable{
					&runbatch.OSCommand{
						Path:  "/bin/echo",
						Args:  []string{"This is a nested command"},
						Label: "Nested Echo",
					},
				},
			},
		},
	}

	// Run the batch and get results
	results := batch.Run(ctx, sig)

	// Display results with different output options
	fmt.Println("\n=== Default Output (errors only) ===")
	results.Print()

	fmt.Println("\n=== Full Output (includes stdout) ===")
	fullOptions := runbatch.DefaultOutputOptions()
	fullOptions.IncludeStdOut = true
	fullOptions.ShowSuccessDetails = true
	results.PrintWithOptions(fullOptions)

	fmt.Println("\n=== Compact Output (no colors) ===")
	compactOptions := runbatch.DefaultOutputOptions()
	compactOptions.ColorOutput = false
	compactOptions.IncludeStdErr = true
	results.PrintWithOptions(compactOptions)

	fmt.Println("\n=== Error Output Focus (stderr only) ===")
	errorOptions := runbatch.DefaultOutputOptions()
	errorOptions.IncludeStdOut = false
	errorOptions.IncludeStdErr = true
	errorOptions.ShowSuccessDetails = false // Only show errors
	results.PrintWithOptions(errorOptions)

	// You can also write to custom writers
	fmt.Println("\n=== Custom Writer Example ===")
	// Create a file for output (in a real app)
	// file, err := os.Create("results.txt")
	// if err != nil {
	//     fmt.Println("Error creating file:", err)
	//     return
	// }
	// defer file.Close()
	// results.Write(file)

	// For now, just write to stdout
	results.Write(os.Stdout)
}
