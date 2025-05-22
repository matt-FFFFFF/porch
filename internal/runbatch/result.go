// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"fmt"
	"io"
	"os"
	"slices"
)

var ErrResultChildrenHasError = fmt.Errorf("result has children with errors")

// Result represents the outcome of running a command or batch.
type Result struct {
	ExitCode int     // Exit code of the command or batch
	Error    error   // Error, if any
	StdOut   []byte  // Output from the command(s)
	StdErr   []byte  // Error output from the command(s)
	Label    string  // Label of the command or batch
	Children Results // Nested results for tree output
	newCwd   string  // New working directory, if changed
}

// Results is a slice of Result pointers, used to represent multiple results.
type Results []*Result

func (r Results) HasError() bool {
	for v := range slices.Values(r) {
		if v.Error != nil || v.ExitCode != 0 {
			return true
		}

		if v.Children != nil {
			if v.Children.HasError() {
				return true
			}
		}
	}

	return false
}

// Print outputs the results to stdout with default options.
func (r Results) Print() error {
	return WriteResults(os.Stdout, r, nil)
}

// PrintWithOptions outputs the results to stdout with the specified options.
func (r Results) PrintWithOptions(options *OutputOptions) error {
	return WriteResults(os.Stdout, r, options)
}

// Write outputs the results to the specified writer with default options.
func (r Results) Write(w io.Writer) error {
	return WriteResults(w, r, nil)
}

// WriteWithOptions outputs the results to the specified writer with the specified options.
func (r Results) WriteWithOptions(w io.Writer, options *OutputOptions) error {
	return WriteResults(w, r, options)
}
