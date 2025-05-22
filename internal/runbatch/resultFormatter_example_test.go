// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ExampleWriteResults_stderrOutput demonstrates how to format stderr output with the result formatter.
func ExampleWriteResults_stderrOutput() {
	// Create a sample result hierarchy with stderr output
	cmdWithStderr := &Result{
		Label:    "Command with StdErr",
		ExitCode: 1,
		Error:    fmt.Errorf("command failed with output error"),
		StdErr:   []byte("Error message line 1\nError message line 2\n  Indented error line\nAnother error line"),
	}

	nestedCmd := &Result{
		Label:    "Nested Command with StdErr",
		ExitCode: 2,
		Error:    fmt.Errorf("nested command timed out"),
		StdErr:   []byte("Nested error 1\nNested error 2"),
	}

	// Create a parent result that contains both commands
	parentResult := &Result{
		Label:    "Parent Batch",
		ExitCode: -1,
		Error:    ErrResultChildrenHasError,
		Children: Results{cmdWithStderr, nestedCmd},
	}

	results := Results{parentResult}

	// Create a string writer for the example output
	var buf strings.Builder

	// Create options that show stderr and use plain text (no colors) for example output
	options := &OutputOptions{
		IncludeStdOut:      false,
		IncludeStdErr:      true,
		ColorOutput:        false,
		ShowSuccessDetails: false,
	}

	// Write the results to the buffer
	_ = WriteResults(&buf, results, options)

	// For the example, print to stdout
	fmt.Println(buf.String())

	// Output:
	// ✗ Parent Batch (exit code: -1)
	//   ✗ Command with StdErr (exit code: 1)
	//     ➜ Error: command failed with output error
	//     ➜ Error Output:
	//        Error message line 1
	//        Error message line 2
	//          Indented error line
	//        Another error line
	//   ✗ Nested Command with StdErr (exit code: 2)
	//     ➜ Error: nested command timed out
	//     ➜ Error Output:
	//        Nested error 1
	//        Nested error 2
	//
}

// Example of using the formatter to display results.
func Example_resultFormatting() {
	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a batch with nested commands
	batch := &SerialBatch{
		Label: "Demo Batch",
		Commands: []Runnable{
			&OSCommand{
				Path:  "/bin/echo",
				Args:  []string{"Hello, world!"},
				Label: "Greeting Command",
			},
			&SerialBatch{
				Label: "Parallel Processes",
				Commands: []Runnable{
					&OSCommand{
						Path:  "/bin/cat",
						Args:  []string{"/etc/hosts"},
						Label: "Show Hosts",
					},
					&OSCommand{
						Path:  "/bin/sh",
						Args:  []string{"-c", "echo 'This will fail' && exit 1"},
						Label: "Failing Command",
					},
				},
			},
		},
	}

	// Run the batch
	results := batch.Run(ctx)

	// Output the results with different options
	fmt.Println("Default Output (failures only):")
	results.Print() //nolint:errcheck
	// Output:
	// Default Output (failures only):
	// [31m✗[0m [1;31mDemo Batch[0m (exit code: -1)
	//   [32m✓[0m [1;32mGreeting Command[0m
	//   [31m✗[0m [1;31mParallel Processes[0m (exit code: -1)
	//     [32m✓[0m [1;32mShow Hosts[0m
	//     [31m✗[0m [1;31mFailing Command[0m (exit code: 1)
}
