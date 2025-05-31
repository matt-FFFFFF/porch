// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compareByteslices compares two byte slices, treating nil and empty slices as equivalent.
func compareByteslices(a, b []byte) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}

	return reflect.DeepEqual(a, b)
}

func TestResult_GobEncodeDecode(t *testing.T) {
	tests := []struct {
		name   string
		result *Result
	}{
		{
			name:   "empty result",
			result: &Result{},
		},
		{
			name: "result with exit code only",
			result: &Result{
				ExitCode: 1,
			},
		},
		{
			name: "result with basic error",
			result: &Result{
				ExitCode: 1,
				Error:    fmt.Errorf("command failed"),
			},
		},
		{
			name: "result with predefined error",
			result: &Result{
				ExitCode: 1,
				Error:    ErrResultChildrenHasError,
			},
		},
		{
			name: "result with wrapped error",
			result: &Result{
				ExitCode: 1,
				Error:    fmt.Errorf("failed to execute: %w", errors.New("underlying error")),
			},
		},
		{
			name: "result with output data",
			result: &Result{
				ExitCode: 0,
				StdOut:   []byte("Hello, World!\n"),
				StdErr:   []byte("Warning: deprecated flag\n"),
			},
		},
		{
			name: "result with label and newCwd",
			result: &Result{
				ExitCode: 0,
				Label:    "test-command",
				newCwd:   "/tmp/test-dir",
			},
		},
		{
			name: "result with empty byte slices",
			result: &Result{
				ExitCode: 0,
				StdOut:   []byte{},
				StdErr:   []byte{},
			},
		},
		{
			name: "result with nil byte slices",
			result: &Result{
				ExitCode: 0,
				StdOut:   nil,
				StdErr:   nil,
			},
		},
		{
			name: "result with large output",
			result: &Result{
				ExitCode: 0,
				StdOut:   make([]byte, 10000), // Large output to test buffer handling
				StdErr:   []byte("Error message"),
			},
		},
		{
			name: "result with special characters in output",
			result: &Result{
				ExitCode: 0,
				StdOut:   []byte("UTF-8: 你好世界 🌍\nSpecial chars: !@#$%^&*()"),
				StdErr:   []byte("Error with unicode: ñ é ü"),
				Label:    "unicode-test",
			},
		},
		{
			name: "complete result with all fields",
			result: &Result{
				ExitCode: 42,
				Error:    fmt.Errorf("comprehensive test error"),
				StdOut:   []byte("Standard output content"),
				StdErr:   []byte("Standard error content"),
				Label:    "comprehensive-test",
				newCwd:   "/path/to/new/directory",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode the result
			encoded, err := tt.result.GobEncode()
			require.NoError(t, err, "GobEncode() failed")

			// Decode the result
			decoded := &Result{}
			require.NoError(t, decoded.GobDecode(encoded), "GobDecode() failed")

			// Compare all fields
			assert.Equal(t, tt.result.ExitCode, decoded.ExitCode, "ExitCode mismatch")

			// Compare byte slices with special handling for empty vs nil
			assert.True(t, compareByteslices(decoded.StdOut, tt.result.StdOut), "StdOut mismatch")
			assert.True(t, compareByteslices(decoded.StdErr, tt.result.StdErr), "StdErr mismatch")

			assert.Equal(t, tt.result.Label, decoded.Label, "Label mismatch")
			assert.Equal(t, tt.result.newCwd, decoded.newCwd, "newCwd mismatch")

			// Compare errors
			if tt.result.Error == nil {
				assert.NoError(t, decoded.Error, "Expected nil error")
			} else {
				require.Error(t, decoded.Error, "Expected non-nil error")
				// For gob-decoded errors, we expect them to be wrapped with ErrGobDecode
				// Extract the original message by checking if it's wrapped
				expectedMsg := tt.result.Error.Error()

				actualMsg := decoded.Error.Error()
				actualMsg = strings.TrimPrefix(actualMsg, "gob decode error: ")

				assert.Equal(t, expectedMsg, actualMsg, "Error message mismatch")
			}
		})
	}
}

func TestResult_GobEncodeDecodeWithChildren(t *testing.T) {
	// Create a complex nested structure
	child1 := &Result{
		ExitCode: 1,
		Error:    fmt.Errorf("child 1 failed"),
		StdOut:   []byte("child 1 output"),
		StdErr:   []byte("child 1 error"),
		Label:    "child-1",
	}

	child2 := &Result{
		ExitCode: 0,
		StdOut:   []byte("child 2 output"),
		Label:    "child-2",
	}

	grandchild := &Result{
		ExitCode: 2,
		Error:    fmt.Errorf("grandchild error"),
		StdOut:   []byte("grandchild output"),
		Label:    "grandchild",
	}

	child2.Children = Results{grandchild}

	parent := &Result{
		ExitCode: 0,
		StdOut:   []byte("parent output"),
		Label:    "parent",
		Children: Results{child1, child2},
		newCwd:   "/parent/dir",
	}

	// Encode the parent result
	encoded, err := parent.GobEncode()
	require.NoError(t, err, "GobEncode() failed")

	// Decode the result
	decoded := &Result{}
	require.NoError(t, decoded.GobDecode(encoded), "GobDecode() failed")

	// Verify parent fields
	assert.Equal(t, parent.ExitCode, decoded.ExitCode, "Parent ExitCode mismatch")
	assert.Equal(t, parent.Label, decoded.Label, "Parent Label mismatch")
	assert.Equal(t, parent.newCwd, decoded.newCwd, "Parent newCwd mismatch")

	// Verify children count
	require.Len(t, decoded.Children, len(parent.Children), "Children count mismatch")

	// Verify child 1
	decodedChild1 := decoded.Children[0]
	assert.Equal(t, child1.ExitCode, decodedChild1.ExitCode, "Child1 ExitCode mismatch")
	require.Error(t, decodedChild1.Error, "Child1 should have an error")
	expectedChild1Msg := child1.Error.Error()

	actualChild1Msg := decodedChild1.Error.Error()
	actualChild1Msg = strings.TrimPrefix(actualChild1Msg, "gob decode error: ")

	assert.Equal(t, expectedChild1Msg, actualChild1Msg, "Child1 Error mismatch")
	assert.Equal(t, child1.Label, decodedChild1.Label, "Child1 Label mismatch")

	// Verify child 2 and its grandchild
	decodedChild2 := decoded.Children[1]
	assert.Equal(t, child2.Label, decodedChild2.Label, "Child2 Label mismatch")
	require.Len(t, decodedChild2.Children, 1, "Child2 children count mismatch")

	decodedGrandchild := decodedChild2.Children[0]
	assert.Equal(t, grandchild.ExitCode, decodedGrandchild.ExitCode, "Grandchild ExitCode mismatch")
	require.Error(t, decodedGrandchild.Error, "Grandchild should have an error")
	expectedGrandchildMsg := grandchild.Error.Error()

	actualGrandchildMsg := decodedGrandchild.Error.Error()
	actualGrandchildMsg = strings.TrimPrefix(actualGrandchildMsg, "gob decode error: ")

	assert.Equal(t, expectedGrandchildMsg, actualGrandchildMsg, "Grandchild Error mismatch")
}

func TestResult_GobEncodeError(t *testing.T) {
	// Test with an error type that can't be encoded
	// Note: Most standard error types should work, but this tests error handling
	result := &Result{
		ExitCode: 1,
		Error:    fmt.Errorf("test error"),
	}

	// This should succeed for standard errors
	_, err := result.GobEncode()
	assert.NoError(t, err, "GobEncode() unexpectedly failed")
}

func TestResult_GobDecodeInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "invalid gob data",
			data: []byte{0x1, 0x2, 0x3, 0x4, 0x5},
		},
		{
			name: "truncated data",
			data: []byte{0x07, 0xff, 0x81, 0x03}, // Partial gob header
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{}
			err := result.GobDecode(tt.data)
			assert.Error(t, err, "GobDecode() should have failed with invalid data")
		})
	}
}

func TestResult_GobEncodeDecodeRoundTrip(t *testing.T) {
	// Test multiple encode/decode cycles to ensure consistency
	original := &Result{
		ExitCode: 123,
		Error:    fmt.Errorf("persistent error"),
		StdOut:   []byte("persistent output"),
		StdErr:   []byte("persistent error output"),
		Label:    "persistent-test",
		newCwd:   "/persistent/path",
	}

	current := original

	// Perform 5 encode/decode cycles
	for i := 0; i < 5; i++ {
		encoded, err := current.GobEncode()
		require.NoError(t, err, "Round %d: GobEncode() failed", i+1)

		decoded := &Result{}
		require.NoError(t, decoded.GobDecode(encoded), "Round %d: GobDecode() failed", i+1)

		// Verify the data hasn't changed
		assert.Equal(t, original.ExitCode, decoded.ExitCode, "Round %d: ExitCode changed", i+1)
		require.Error(t, decoded.Error, "Round %d: Error should not be nil", i+1)
		expectedMsg := original.Error.Error()
		actualMsg := decoded.Error.Error()
		// Remove all "gob decode error: " prefixes to get to the original message
		for strings.HasPrefix(actualMsg, "gob decode error: ") {
			actualMsg = strings.TrimPrefix(actualMsg, "gob decode error: ")
		}

		assert.Equal(t, expectedMsg, actualMsg, "Round %d: Error changed", i+1)
		assert.True(t, compareByteslices(original.StdOut, decoded.StdOut), "Round %d: StdOut changed", i+1)

		current = decoded
	}
}

func TestResult_GobEncodeDecodeNilError(t *testing.T) {
	// Specifically test the nil error case
	result := &Result{
		ExitCode: 0,
		Error:    nil,
		StdOut:   []byte("success output"),
		Label:    "success-test",
	}

	encoded, err := result.GobEncode()
	require.NoError(t, err, "GobEncode() failed")

	decoded := &Result{}
	require.NoError(t, decoded.GobDecode(encoded), "GobDecode() failed")

	assert.NoError(t, decoded.Error, "Expected nil error")
}
