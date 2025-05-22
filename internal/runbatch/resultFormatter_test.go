// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteResults_SimpleSuccess(t *testing.T) {
	results := Results{
		{
			Label:    "simple-command",
			ExitCode: 0,
			StdOut:   []byte("success output"),
		},
	}

	var buf bytes.Buffer

	opts := &OutputOptions{
		IncludeStdOut:      true,
		IncludeStdErr:      true,
		ColorOutput:        false,
		ShowSuccessDetails: true,
	}

	err := WriteResults(&buf, results, opts)
	assert.NoError(t, err)

	output := buf.String()

	assert.Contains(t, output, "✓ simple-command")
	assert.Contains(t, output, "success output")
}

func TestWriteResults_SimpleFailure(t *testing.T) {
	results := Results{
		{
			Label:    "failed-command",
			ExitCode: 1,
			Error:    errors.New("command failed"),
			StdErr:   []byte("error details"),
		},
	}

	var buf bytes.Buffer

	opts := &OutputOptions{
		IncludeStdOut: false,
		IncludeStdErr: true,
		ColorOutput:   false,
	}

	err := WriteResults(&buf, results, opts)
	assert.NoError(t, err)

	output := buf.String()

	assert.Contains(t, output, "✗ failed-command")
	assert.Contains(t, output, "Error: command failed")
	assert.Contains(t, output, "error details")
}

func TestWriteResults_HierarchicalResults(t *testing.T) {
	// Create a nested structure of results
	childResults := Results{
		{
			Label:    "child-success",
			ExitCode: 0,
			StdOut:   []byte("child success output"),
		},
		{
			Label:    "child-failure",
			ExitCode: 2,
			Error:    errors.New("child command failed"),
			StdErr:   []byte("child error details"),
		},
	}

	results := Results{
		{
			Label:    "parent-batch",
			ExitCode: -1,
			Error:    ErrResultChildrenHasError,
			Children: childResults,
		},
	}

	var buf bytes.Buffer

	opts := &OutputOptions{
		IncludeStdOut: true,
		IncludeStdErr: true,
		ColorOutput:   false,
	}

	err := WriteResults(&buf, results, opts)
	assert.NoError(t, err)

	output := buf.String()

	// Check parent output
	assert.Contains(t, output, "✗ parent-batch")

	// Check that parent doesn't show redundant error message
	assert.NotContains(t, output, "Error: result has children with errors")

	// Check child outputs with indentation
	assert.Contains(t, output, "  ✓ child-success")
	assert.Contains(t, output, "  ✗ child-failure")
	assert.Contains(t, output, "  ➜ Error: child command failed")
	assert.Contains(t, output, "child error details")
}

func TestWriteResults_DefaultOptions(t *testing.T) {
	results := Results{
		{
			Label:    "default-options",
			ExitCode: 0,
			StdOut:   []byte("standard output"),
			StdErr:   []byte("error output"),
		},
	}

	var buf bytes.Buffer
	err := WriteResults(&buf, results, nil)
	assert.NoError(t, err)

	// With default options, stdout should not be included but stderr should be
	output := buf.String()
	assert.NotContains(t, output, "standard output")

	// By default, we don't show success details
	assert.NotContains(t, output, "error output")
}

func TestWriteResults_StdErrFormatting(t *testing.T) {
	// Create a result with multiline stderr output
	results := Results{
		{
			Label:    "multiline-stderr",
			ExitCode: 1,
			Error:    errors.New("command failed"),
			StdErr:   []byte("Error line 1\nError line 2\nError line 3\n  Indented error line"),
		},
	}

	var buf bytes.Buffer

	opts := &OutputOptions{
		IncludeStdOut: false,
		IncludeStdErr: true,
		ColorOutput:   false,
	}

	err := WriteResults(&buf, results, opts)
	assert.NoError(t, err)

	output := buf.String()

	// Check the formatting
	assert.Contains(t, output, "✗ multiline-stderr")
	assert.Contains(t, output, "➜ Error: command failed")

	// Check for proper indentation of all stderr lines
	assert.Contains(t, output, "     Error line 1")
	assert.Contains(t, output, "     Error line 2")
	assert.Contains(t, output, "     Error line 3")
	assert.Contains(t, output, "       Indented error line")
}
