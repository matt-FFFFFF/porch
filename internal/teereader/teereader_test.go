// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package teereader

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLastLineTeeReader(t *testing.T) {
	reader := strings.NewReader("test data")
	teeReader := NewLastLineTeeReader(reader)

	assert.NotNil(t, teeReader)
	assert.NotNil(t, teeReader.reader)
	assert.NotNil(t, teeReader.fullBuffer)
	assert.Empty(t, teeReader.lastLine)
	assert.Empty(t, teeReader.GetPartialLine())
}

func TestLastLineTeeReader_SingleLine(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedBuffer  string
		expectedLast    string
		expectedPartial string
	}{
		{
			name:            "single line with newline",
			input:           "hello world\n",
			expectedBuffer:  "hello world\n",
			expectedLast:    "hello world",
			expectedPartial: "",
		},
		{
			name:            "single line without newline",
			input:           "hello world",
			expectedBuffer:  "hello world",
			expectedLast:    "",
			expectedPartial: "hello world",
		},
		{
			name:            "empty string",
			input:           "",
			expectedBuffer:  "",
			expectedLast:    "",
			expectedPartial: "",
		},
		{
			name:            "just newline",
			input:           "\n",
			expectedBuffer:  "\n",
			expectedLast:    "",
			expectedPartial: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			teeReader := NewLastLineTeeReader(reader)

			// Read all data
			data, err := io.ReadAll(teeReader)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedBuffer, string(data))
			assert.Equal(t, tt.expectedBuffer, string(teeReader.GetFullBufferBytes()))
			assert.Equal(t, tt.expectedLast, teeReader.GetLastLine(0))
			assert.Equal(t, tt.expectedPartial, teeReader.GetPartialLine())
		})
	}
}

func TestLastLineTeeReader_MultipleLines(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedLast    string
		expectedPartial string
	}{
		{
			name:            "two lines with newline",
			input:           "line1\nline2\n",
			expectedLast:    "line2",
			expectedPartial: "",
		},
		{
			name:            "two lines without final newline",
			input:           "line1\nline2",
			expectedLast:    "line1",
			expectedPartial: "line2",
		},
		{
			name:            "three lines mixed",
			input:           "first\nsecond\nthird\n",
			expectedLast:    "third",
			expectedPartial: "",
		},
		{
			name:            "multiple empty lines",
			input:           "line1\n\n\nline4\n",
			expectedLast:    "line4",
			expectedPartial: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			teeReader := NewLastLineTeeReader(reader)

			// Read all data
			data, err := io.ReadAll(teeReader)
			require.NoError(t, err)

			assert.Equal(t, tt.input, string(data))
			assert.Equal(t, tt.input, string(teeReader.GetFullBufferBytes()))
			assert.Equal(t, tt.expectedLast, teeReader.GetLastLine(0))
			assert.Equal(t, tt.expectedPartial, teeReader.GetPartialLine())
		})
	}
}

func TestLastLineTeeReader_ChunkedReading(t *testing.T) {
	input := "first line\nsecond line\nthird line\nfourth line"
	reader := strings.NewReader(input)
	teeReader := NewLastLineTeeReader(reader)

	// Read in small chunks to test incremental processing
	buffer := make([]byte, 5)

	var result []byte

	for {
		n, err := teeReader.Read(buffer)
		if n > 0 {
			result = append(result, buffer[:n]...)
		}

		if err == io.EOF {
			break
		}

		require.NoError(t, err)
	}

	assert.Equal(t, input, string(result))
	assert.Equal(t, input, string(teeReader.GetFullBufferBytes()))
	assert.Equal(t, "third line", teeReader.GetLastLine(0))
	assert.Equal(t, "fourth line", teeReader.GetPartialLine())
}

func TestLastLineTeeReader_ProgressiveReading(t *testing.T) {
	// Test that we can track the last line as data comes in progressively
	reader := strings.NewReader("line1\nline2\nline3\n")
	teeReader := NewLastLineTeeReader(reader)

	// Read first chunk that includes first line
	buffer := make([]byte, 7) // "line1\nl"
	n, err := teeReader.Read(buffer)
	require.NoError(t, err)
	assert.Equal(t, 7, n)
	assert.Equal(t, "line1", teeReader.GetLastLine(0))
	assert.Equal(t, "l", teeReader.GetPartialLine())

	// Read second chunk
	buffer = make([]byte, 6) // "ine2\nl"
	n, err = teeReader.Read(buffer)
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, "line2", teeReader.GetLastLine(0))
	assert.Equal(t, "l", teeReader.GetPartialLine())

	// Read final chunk
	buffer = make([]byte, 6) // "ine3\n"
	n, err = teeReader.Read(buffer)
	require.NoError(t, err)
	assert.Equal(t, 5, n) // Only 5 bytes left
	assert.Equal(t, "line3", teeReader.GetLastLine(0))
	assert.Empty(t, teeReader.GetPartialLine())
}

func TestLastLineTeeReader_ConcurrentAccess(t *testing.T) {
	input := strings.Repeat("line\n", 1000)
	reader := strings.NewReader(input)
	teeReader := NewLastLineTeeReader(reader)

	var wg sync.WaitGroup

	// Start reading in one goroutine
	wg.Add(1)

	go func() {
		defer wg.Done()

		_, err := io.ReadAll(teeReader)
		assert.NoError(t, err)
	}()

	// Access last line and buffer from multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				_ = teeReader.GetLastLine(0)
				_ = teeReader.GetFullBufferBytes()
				_ = teeReader.GetPartialLine()
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, "line", teeReader.GetLastLine(0))
	assert.Empty(t, teeReader.GetPartialLine())
	assert.Equal(t, input, string(teeReader.GetFullBufferBytes()))
}

func TestLastLineTeeReader_Reset(t *testing.T) {
	reader := strings.NewReader("line1\nline2\npartial")
	teeReader := NewLastLineTeeReader(reader)

	// Read some data
	_, err := io.ReadAll(teeReader)
	require.NoError(t, err)

	// Verify data is captured
	assert.Equal(t, "line2", teeReader.GetLastLine(0)) // line2 is the last complete line
	assert.Equal(t, "partial", teeReader.GetPartialLine())
	assert.NotEmpty(t, teeReader.GetFullBufferBytes())

	// Reset
	teeReader.Reset()

	// Verify everything is cleared
	assert.Empty(t, teeReader.GetLastLine(0))
	assert.Empty(t, teeReader.GetPartialLine())
	assert.Empty(t, teeReader.GetFullBufferBytes())
}

func TestLastLineTeeReader_ErrorHandling(t *testing.T) {
	// Create a reader that will return an error
	errorReader := &errorReader{data: "some data", shouldError: true}
	teeReader := NewLastLineTeeReader(errorReader)

	buffer := make([]byte, 100)
	n, err := teeReader.Read(buffer)

	// Should get the data and the error
	assert.Equal(t, 9, n) // len("some data")
	require.Error(t, err)
	assert.Equal(t, "assert.AnError general error for testing", err.Error())

	// Buffer should still contain the data
	assert.Equal(t, "some data", string(teeReader.GetFullBufferBytes()))
}

func TestLastLineTeeReader_LargeData(t *testing.T) {
	// Test with larger amounts of data
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = strings.Repeat("x", 100) // 100 char lines
	}

	input := strings.Join(lines, "\n") + "\n"

	reader := strings.NewReader(input)
	teeReader := NewLastLineTeeReader(reader)

	data, err := io.ReadAll(teeReader)
	require.NoError(t, err)

	assert.Equal(t, input, string(data))
	assert.Equal(t, input, string(teeReader.GetFullBufferBytes()))
	assert.Equal(t, lines[999], teeReader.GetLastLine(0)) // Last line
	assert.Empty(t, teeReader.GetPartialLine())
}

// errorReader is a test helper that returns an error after returning some data.
type errorReader struct {
	data        string
	shouldError bool
	read        bool
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	if e.read {
		return 0, io.EOF
	}

	e.read = true
	n = copy(p, e.data)

	if e.shouldError {
		return n, assert.AnError
	}

	return n, nil
}
