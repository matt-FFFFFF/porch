// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
)

// ErrResultChildrenHasError is the error returned when a result has children with at least one error.
var ErrResultChildrenHasError = fmt.Errorf("result has children with errors")

// ErrGobDecode is the base error for gob decoding operations.
var ErrGobDecode = errors.New("gob decode error")

// Results is a slice of Result pointers, used to represent multiple results.
type Results []*Result

// gobResult is a helper struct for gob encoding/decoding that handles the error interface
// and unexported fields.
type gobResult struct {
	ExitCode int
	ErrorMsg string // Store error message as string instead of reflect.Value
	HasError bool   // Track whether there was an error
	StdOut   []byte
	StdErr   []byte
	Label    string
	Children Results
	NewCwd   string // Exported version of newCwd
}

// Result represents the outcome of running a command or batch.
type Result struct {
	ExitCode int     // Exit code of the command or batch
	Error    error   // Error, if any
	StdOut   []byte  // Output from the command(s)
	StdErr   []byte  // Error output from the command(s)
	Label    string  // Label of the command or batch
	Children Results // Nested results for tree output
	newCwd   string  // New working directory, if changed. Only used for serial batches.
}

// GobEncode implements the gob.GobEncoder interface for Result.
func (r *Result) GobEncode() ([]byte, error) {
	gr := gobResult{
		ExitCode: r.ExitCode,
		StdOut:   r.StdOut,
		StdErr:   r.StdErr,
		Label:    r.Label,
		Children: r.Children,
		NewCwd:   r.newCwd,
	}

	// Convert error to string
	if r.Error != nil {
		gr.HasError = true
		gr.ErrorMsg = r.Error.Error()
	}

	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(gr); err != nil {
		return nil, fmt.Errorf("failed to encode Result: %w", err)
	}

	return buf.Bytes(), nil
}

// GobDecode implements the gob.GobDecoder interface for Result.
func (r *Result) GobDecode(data []byte) error {
	var gr gobResult

	buf := bytes.NewReader(data)

	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&gr); err != nil {
		return fmt.Errorf("failed to decode Result: %w", err)
	}

	r.ExitCode = gr.ExitCode
	r.StdOut = gr.StdOut
	r.StdErr = gr.StdErr
	r.Label = gr.Label
	r.Children = gr.Children
	r.newCwd = gr.NewCwd

	// Convert error message back to error
	if gr.HasError {
		r.Error = fmt.Errorf("%w: %s", ErrGobDecode, gr.ErrorMsg)
	}

	return nil
}

// HasError if any of the results in the hierarchy has an error or non-zero exit code.
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
	return writeTextResults(os.Stdout, r, nil)
}

// PrintWithOptions outputs the results to stdout with the specified options.
func (r Results) PrintWithOptions(options *OutputOptions) error {
	return writeTextResults(os.Stdout, r, options)
}

// WriteText outputs the results to the specified writer with default options.
func (r Results) WriteText(w io.Writer) error {
	return writeTextResults(w, r, nil)
}

// WriteTextWithOptions outputs the results to the specified writer with the specified options.
func (r Results) WriteTextWithOptions(w io.Writer, options *OutputOptions) error {
	return writeTextResults(w, r, options)
}

// WriteBinary outputs the results to the specified writer in binary format using gob encoding.
func (r Results) WriteBinary(w io.Writer) error {
	return writeResultGob(w, r)
}
