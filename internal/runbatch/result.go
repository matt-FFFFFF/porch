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

const (
	resultStatusUnknownStr = "unknown"
	resultStatusSuccessStr = "success"
	resultStatusSkippedStr = "skipped"
	resultStatusWarningStr = "warning"
	resultStatusErrorStr   = "error"
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
	ExitCode int          `json:"exit_code"`       // Exit code of the command or batch
	ErrorMsg string       `json:"error,omitempty"` // Store error message as string instead of reflect.Value
	HasError bool         `json:"has_error"`       // Track whether there was an error
	Status   ResultStatus `json:"status"`          // Track whether the command was skipped
	StdOut   []byte       `json:"stdout"`
	StdErr   []byte       `json:"stderr"`
	Label    string       `json:"label"`
	Children Results      `json:"children,omitempty"` // Nested results for tree output
	NewCwd   string       `json:"new_cwd,omitempty"`  // Exported version of newCwd
}

// Result represents the outcome of running a command or batch.
type Result struct {
	// Exit code of the command or batch.
	ExitCode int
	// Error, if any.
	Error error
	// The status of the result, e.g. success, error, skipped.
	// The Default is ResultStatusUnknown, so always ensure to set this to something meaningful.
	Status ResultStatus
	// Output from the command(s).
	StdOut []byte
	// Error output from the command(s).
	StdErr []byte
	// Label of the command or batch.
	Label string
	// Nested results for tree output.
	Children Results
	// New working directory, if changed. Only processed by serial batches.
	newCwd string
}

// ResultStatus summarizes the status of a command or batch result.
type ResultStatus int

const (
	// ResultStatusUnknown indicates the result status is unknown.
	ResultStatusUnknown ResultStatus = iota
	// ResultStatusSuccess indicates the command or batch completed successfully.
	ResultStatusSuccess
	// ResultStatusSkipped indicates the command or batch was skipped.
	ResultStatusSkipped
	// ResultStatusWarning indicates the command or batch completed with warnings.
	ResultStatusWarning
	// ResultStatusError indicates the command or batch failed.
	ResultStatusError
)

// String implements the Stringer interface for ResultStatus.
func (rs ResultStatus) String() string {
	switch rs {
	case ResultStatusSuccess:
		return resultStatusSuccessStr
	case ResultStatusSkipped:
		return resultStatusSkippedStr
	case ResultStatusWarning:
		return resultStatusWarningStr
	case ResultStatusError:
		return resultStatusErrorStr
	}

	return resultStatusUnknownStr
}

// GobEncode implements the gob.GobEncoder interface for Result.
func (r *Result) GobEncode() ([]byte, error) {
	gr := gobResult{
		ExitCode: r.ExitCode,
		StdOut:   r.StdOut,
		StdErr:   r.StdErr,
		Status:   r.Status,
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
	r.Status = gr.Status

	// Convert error message back to error
	if gr.HasError {
		r.Error = errors.New(gr.ErrorMsg) //nolint:err113
	}

	return nil
}

// HasError if any of the results in the hierarchy has an error or non-zero exit code.
func (r Results) HasError() bool {
	for v := range slices.Values(r) {
		if v.Status == ResultStatusError {
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
