// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commandinpath

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	// Create temp directory for test commands
	tempDir, err := os.MkdirTemp("", "commandinpath_test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir) //nolint:errcheck

	// Create a mock command file in tempDir
	mockCommandName := "mockcommand"
	if os.Getenv("GOOS") == "windows" {
		mockCommandName += ".exe"
	}

	mockCommandPath := filepath.Join(tempDir, mockCommandName)

	// Create an executable file
	f, err := os.Create(mockCommandPath)
	require.NoError(t, err, "Failed to create mock command")
	defer f.Close() //nolint:errcheck

	// Make it executable (for Unix systems)
	if runtime.GOOS != "windows" {
		err = os.Chmod(mockCommandPath, 0755)
		require.NoError(t, err, "Failed to make mock command executable")
	}

	// Test cases
	tests := []struct {
		name         string
		label        string
		command      string
		cwd          string
		args         []string
		path         string // PATH environment variable to set
		expectedNil  bool
		expectedPath string
		fileMode     os.FileMode
	}{
		{
			name:         "Command found",
			label:        "test-label",
			command:      mockCommandName,
			cwd:          "/test/cwd",
			args:         []string{},
			path:         tempDir,
			expectedNil:  false,
			expectedPath: mockCommandPath,
			fileMode:     0755,
		},
		{
			name:        "Command not found",
			label:       "test-label",
			command:     "nonexistentcommand",
			cwd:         "/test/cwd",
			args:        []string{},
			path:        tempDir,
			expectedNil: true,
			fileMode:    0755,
		},
		{
			name:         "Multiple paths in PATH - command found",
			label:        "test-label",
			command:      mockCommandName,
			cwd:          "/test/cwd",
			args:         []string{},
			path:         "/non/existent/path" + string(os.PathListSeparator) + tempDir,
			expectedNil:  false,
			expectedPath: mockCommandPath,
			fileMode:     0755,
		},
		{
			name:        "Empty PATH",
			label:       "test-label",
			command:     mockCommandName,
			cwd:         "/test/cwd",
			args:        []string{},
			path:        "",
			expectedNil: true,
			fileMode:    0755,
		},
		{
			name:        "File not executable",
			label:       "test-label",
			command:     mockCommandName,
			cwd:         "/test/cwd",
			args:        []string{},
			path:        tempDir,
			expectedNil: true,
			fileMode:    0644,
		},
		{
			name:        "Empty command",
			label:       "test-label",
			command:     "",
			cwd:         "/test/cwd",
			args:        []string{},
			path:        tempDir,
			expectedNil: true,
			fileMode:    0755,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set PATH for this test
			t.Setenv("PATH", tc.path)

			require.NoError(t, f.Chmod(tc.fileMode))

			// Call the function under test
			result, _ := New(tc.label, tc.command, tc.cwd, tc.args)

			// Check if result is nil
			if tc.expectedNil {
				assert.Nil(t, result, "Expected nil result")
			} else {
				assert.NotNil(t, result, "Expected non-nil result")
				assert.Equal(t, tc.label, result.Label, "Label should match")
				assert.Equal(t, tc.expectedPath, result.Path, "Path should match")
				assert.Equal(t, tc.cwd, result.Cwd, "Cwd should match")
				assert.Len(t, result.Args, len(tc.args), "Args length should match")

				for i, arg := range tc.args {
					assert.Equal(t, arg, result.Args[i], "Arg %d should match", i)
				}
			}
		})
	}
}
