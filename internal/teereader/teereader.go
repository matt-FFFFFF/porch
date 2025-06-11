// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package teereader

import (
	"bytes"
	"io"
	"strings"
	"sync"
)

// LastLineTeeReader wraps an io.Reader and captures both the complete output
// and tracks the last complete line for progress display purposes.
// It is safe for concurrent use.
type LastLineTeeReader struct {
	reader         io.Reader
	fullBuffer     *bytes.Buffer
	lastLine       string
	partialBuilder strings.Builder // Buffer for incomplete lines
	mu             sync.RWMutex
}

// NewLastLineTeeReader creates a new LastLineTeeReader that wraps the given reader.
// The reader will capture all data while maintaining the last complete line.
func NewLastLineTeeReader(r io.Reader) *LastLineTeeReader {
	return &LastLineTeeReader{
		reader:     r,
		fullBuffer: &bytes.Buffer{},
	}
}

// Read implements io.Reader. It reads from the underlying reader and updates
// both the full buffer and the last line tracking.
func (lt *LastLineTeeReader) Read(p []byte) (n int, err error) {
	n, err = lt.reader.Read(p)
	if n > 0 {
		lt.mu.Lock()
		defer lt.mu.Unlock()

		// Write to full buffer
		lt.fullBuffer.Write(p[:n])

		// Process the new data for last line tracking
		lt.processNewData(string(p[:n]))
	}

	return n, err //nolint:wrapcheck
}

// processNewData updates the last line based on new data.
// Must be called with the write lock held.
func (lt *LastLineTeeReader) processNewData(data string) {
	// Combine any existing partial line with new data
	lt.partialBuilder.WriteString(data)
	combined := lt.partialBuilder.String()

	// Split by newlines
	lines := strings.Split(combined, "\n")

	if len(lines) == 1 {
		// No newlines found, partial line is already updated in builder
		return
	}

	// We have at least one complete line
	// The last element is either empty (if data ended with \n) or a partial line
	if data[len(data)-1] == '\n' {
		// Data ended with newline, so last element is empty
		lt.lastLine = lines[len(lines)-2]
		lt.partialBuilder.Reset()
	} else {
		// Data didn't end with newline, so last element is partial
		lt.lastLine = lines[len(lines)-2]
		lt.partialBuilder.Reset()
		lt.partialBuilder.WriteString(lines[len(lines)-1])
	}
}

// GetLastLine returns the last complete line that was read.
// Returns an empty string if no complete lines have been read yet.
// This method is safe for concurrent use.
// If maxLength > 0, it truncates the last line to that length and appends "..." if it exceeds that length.
func (lt *LastLineTeeReader) GetLastLine(maxLength int) string {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	var result string
	result = lt.lastLine
	if maxLength > 0 && len(result) > maxLength {
		result = result[:maxLength-3] + "..."
	}

	return result
}

// GetFullBufferBytes returns a copy of all data that has been read so far.
// This method is safe for concurrent use.
func (lt *LastLineTeeReader) GetFullBufferBytes() []byte {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	return lt.fullBuffer.Bytes()
}

// GetFullBufferReader returns an io.Reader for the full buffer.
// This method is NOT safe for concurrent use and should only be used when
// Read has completed.
func (lt *LastLineTeeReader) GetFullBufferReader() *bytes.Buffer {
	return lt.fullBuffer
}

// GetPartialLine returns the current partial line (data after the last newline).
// This is useful for debugging or showing in-progress output.
// This method is safe for concurrent use.
func (lt *LastLineTeeReader) GetPartialLine() string {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	return lt.partialBuilder.String()
}

// Reset clears all internal buffers and resets the reader to its initial state.
// The underlying reader is not affected.
// This method is safe for concurrent use.
func (lt *LastLineTeeReader) Reset() {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	lt.fullBuffer.Reset()
	lt.lastLine = ""
	lt.partialBuilder.Reset()
}
