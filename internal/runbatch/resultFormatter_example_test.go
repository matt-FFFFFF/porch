// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"fmt"
	"strings"
)

// ExampleWriteResults_stderrOutput demonstrates how to format stderr output with the result formatter.
func ExampleResults_WriteTextWithOptions_stderrOutput() {
	// Create a sample result hierarchy with stderr output
	cmdWithStderr := &Result{
		Label:    "Command with StdErr",
		ExitCode: 1,
		Error:    fmt.Errorf("command failed with output error"),
		StdErr:   []byte("Error message line 1\nError message line 2\n  Indented error line\nAnother error line"),
		Status:   ResultStatusError,
	}

	nestedCmd := &Result{
		Label:    "Nested Command with StdErr",
		ExitCode: 2,
		Error:    fmt.Errorf("nested command timed out"),
		StdErr:   []byte("Nested error 1\nNested error 2"),
		Status:   ResultStatusError,
	}

	// Create a parent result that contains both commands
	parentResult := &Result{
		Label:    "Parent Batch",
		ExitCode: -1,
		Error:    ErrResultChildrenHasError,
		Children: Results{cmdWithStderr, nestedCmd},
		Status:   ResultStatusError,
	}

	results := Results{parentResult}

	// Create a string writer for the example output
	var buf strings.Builder

	// Create options that show stderr and use plain text (no colors) for example output
	options := &OutputOptions{
		IncludeStdOut:      false,
		IncludeStdErr:      true,
		ShowSuccessDetails: false,
	}

	// Write the results to the buffer
	_ = results.WriteTextWithOptions(&buf, options)

	// For the example, print to stdout
	fmt.Println(buf.String())

	// Output:
	// ===== Results =====
	//
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
