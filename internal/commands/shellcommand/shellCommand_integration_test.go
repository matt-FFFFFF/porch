// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build integration

package shellcommand

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandLineEdgeCases_Integration(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	testCases := []struct {
		name             string
		command          string
		expectedOutput   string // Expected substring in output
		expectedExitCode int
		skipOS           []string                        // Operating systems to skip this test on
		setupFunc        func(t *testing.T) string       // Optional setup function that returns cleanup path
		cleanupFunc      func(t *testing.T, path string) // Optional cleanup function
	}{
		{
			name:             "simple echo command",
			command:          `echo "Hello World"`,
			expectedOutput:   "Hello World",
			expectedExitCode: 0,
		},
		{
			name:             "command with nested quotes",
			command:          `echo "He said 'Hello World'"`,
			expectedOutput:   "He said 'Hello World'",
			expectedExitCode: 0,
		},
		{
			name:             "command with escaped quotes (unix)",
			command:          `echo "\"Hello World\""`,
			expectedOutput:   `"Hello World"`,
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
		{
			name:             "command with escaped quotes (windows)",
			command:          `echo "\"Hello World\""`,
			expectedOutput:   `\"Hello World\"`,
			expectedExitCode: 0,
			skipOS:           []string{"linux", "darwin"},
		},
		{
			name:             "command with backslashes (unix)",
			command:          `echo "Path: /usr/local/bin"`,
			expectedOutput:   "Path: /usr/local/bin",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
		{
			name:             "command with backslashes (windows)",
			command:          `echo "Path: C:\Program Files\Test"`,
			expectedOutput:   `Path: C:\Program Files\Test`,
			expectedExitCode: 0,
			skipOS:           []string{"linux", "darwin"},
		},
		{
			name:             "command with pipe",
			command:          `echo "hello world" | grep "world"`,
			expectedOutput:   "hello world",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
		{
			name:             "command with pipe (windows)",
			command:          `echo hello world | findstr "world"`,
			expectedOutput:   "hello world",
			expectedExitCode: 0,
			skipOS:           []string{"linux", "darwin"},
		},
		{
			name:             "command with output redirection",
			command:          `echo "test content" > test_output.txt && cat test_output.txt`,
			expectedOutput:   "test content",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
			cleanupFunc: func(t *testing.T, _ string) {
				os.Remove("test_output.txt")
			},
		},
		{
			name:             "command with output redirection (windows)",
			command:          `echo test content > test_output.txt && type test_output.txt`,
			expectedOutput:   "test content",
			expectedExitCode: 0,
			skipOS:           []string{"linux", "darwin"},
			cleanupFunc: func(t *testing.T, _ string) {
				os.Remove("test_output.txt")
			},
		},
		{
			name:             "command with logical AND operator",
			command:          `echo "first" && echo "second"`,
			expectedOutput:   "first",
			expectedExitCode: 0,
		},
		{
			name:             "command with logical OR operator (unix)",
			command:          `false || echo "fallback"`,
			expectedOutput:   "fallback",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
		{
			name:             "command with environment variable",
			command:          `echo "User: $USER"`,
			expectedOutput:   "User:",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
		{
			name:             "command with environment variable (windows)",
			command:          `echo User: %USERNAME%`,
			expectedOutput:   "User:",
			expectedExitCode: 0,
			skipOS:           []string{"linux", "darwin"},
		},
		{
			name:             "command with subshell (unix)",
			command:          `echo "Date: $(date +%Y)"`,
			expectedOutput:   "Date:",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
		{
			name:             "command with multiple commands",
			command:          `echo "Line 1"; echo "Line 2"`,
			expectedOutput:   "Line 1",
			expectedExitCode: 0,
		},
		{
			name:             "command with unicode characters",
			command:          `echo "Hello 世界 🌍"`,
			expectedOutput:   "Hello 世界 🌍",
			expectedExitCode: 0,
		},
		{
			name:             "command with special characters",
			command:          `echo "Special: !@#$%^&*()"`,
			expectedOutput:   "Special: !@#$%^&*()",
			expectedExitCode: 0,
		},
		{
			name:             "command that should fail",
			command:          `exit 1`,
			expectedOutput:   "",
			expectedExitCode: 1,
		},
		{
			name:             "command with working directory test (unix)",
			command:          `pwd`,
			expectedOutput:   "/",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
		{
			name:             "command with working directory test (windows)",
			command:          `cd`,
			expectedOutput:   ":",
			expectedExitCode: 0,
			skipOS:           []string{"linux", "darwin"},
		},
		{
			name:             "command with append redirection (unix)",
			command:          `echo "line1" > append_test.txt && echo "line2" >> append_test.txt && cat append_test.txt`,
			expectedOutput:   "line1",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
			cleanupFunc: func(t *testing.T, _ string) {
				os.Remove("append_test.txt")
			},
		},
		{
			name:             "long command with repeated text",
			command:          `echo "` + strings.Repeat("A", 100) + `"`,
			expectedOutput:   strings.Repeat("A", 100),
			expectedExitCode: 0,
		},
		{
			name:             "command with stdin redirect (unix)",
			command:          `echo "hello" | cat`,
			expectedOutput:   "hello",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
		{
			name:             "command with multiple pipes (unix)",
			command:          `echo "apple\nbanana\ncherry" | sort | head -1`,
			expectedOutput:   "apple",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
		{
			name:             "command with variable assignment (unix)",
			command:          `TEST_VAR="hello" && echo $TEST_VAR`,
			expectedOutput:   "hello",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
		{
			name:             "command with glob pattern (unix)",
			command:          `echo *.go | wc -w`,
			expectedOutput:   "",
			expectedExitCode: 0,
			skipOS:           []string{"windows"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip test if current OS is in the skipOS list
			for _, skipOS := range tc.skipOS {
				if runtime.GOOS == skipOS {
					t.Skipf("Skipping test on %s", runtime.GOOS)
					return
				}
			}

			// Setup if needed
			var cleanupPath string
			if tc.setupFunc != nil {
				cleanupPath = tc.setupFunc(t)
			}

			// Cleanup if needed
			if tc.cleanupFunc != nil {
				defer tc.cleanupFunc(t, cleanupPath)
			}

			base := runbatch.NewBaseCommand("integration-test", "", runbatch.RunOnSuccess, nil, nil)

			// Create the command
			cmd, err := New(ctx, base, tc.command)
			require.NoError(t, err)
			require.NotNil(t, cmd)

			// Run the command with timeout
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			results := cmd.Run(ctxWithTimeout)
			require.Len(t, results, 1, "expected exactly one result")

			result := results[0]

			// Check exit code
			assert.Equal(t, tc.expectedExitCode, result.ExitCode,
				"unexpected exit code. StdOut: %s, StdErr: %s",
				string(result.StdOut), string(result.StdErr))

			// Check expected output if provided
			if tc.expectedOutput != "" {
				output := string(result.StdOut)
				assert.Contains(t, output, tc.expectedOutput,
					"expected output not found. Full output: %s, StdErr: %s",
					output, string(result.StdErr))
			}

			// For successful commands, verify no error
			if tc.expectedExitCode == 0 {
				assert.NoError(t, result.Error,
					"unexpected error for successful command. StdOut: %s, StdErr: %s",
					string(result.StdOut), string(result.StdErr))
			}
		})
	}
}

func TestCommandWithEnvironmentVariables_Integration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping environment variable test on Windows")
	}

	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	testCases := []struct {
		name           string
		env            map[string]string
		command        string
		expectedOutput string
	}{
		{
			name:           "single environment variable",
			env:            map[string]string{"TEST_VAR": "hello_world"},
			command:        `echo "Value: $TEST_VAR"`,
			expectedOutput: "Value: hello_world",
		},
		{
			name: "multiple environment variables",
			env: map[string]string{
				"FIRST_VAR":  "first",
				"SECOND_VAR": "second",
			},
			command:        `echo "$FIRST_VAR and $SECOND_VAR"`,
			expectedOutput: "first and second",
		},
		{
			name:           "environment variable in pipe",
			env:            map[string]string{"GREP_PATTERN": "test"},
			command:        `echo "test line\nother line" | grep "$GREP_PATTERN"`,
			expectedOutput: "test line",
		},
		{
			name:           "environment variable with special characters",
			env:            map[string]string{"SPECIAL_VAR": "hello@world#123"},
			command:        `echo "Special: $SPECIAL_VAR"`,
			expectedOutput: "Special: hello@world#123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			base := runbatch.NewBaseCommand("env-test", "", runbatch.RunOnSuccess, nil, tc.env)

			cmd, err := New(ctx, base, tc.command)
			require.NoError(t, err)
			require.NotNil(t, cmd)

			ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			results := cmd.Run(ctxWithTimeout)
			require.Len(t, results, 1)

			result := results[0]
			assert.Equal(t, 0, result.ExitCode,
				"command failed. StdOut: %s, StdErr: %s",
				string(result.StdOut), string(result.StdErr))
			assert.NoError(t, result.Error)

			output := string(result.StdOut)
			assert.Contains(t, output, tc.expectedOutput,
				"expected output not found. Full output: %s", output)
		})
	}
}

func TestCommandWithWorkingDirectory_Integration(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	tempDir := t.TempDir()

	// Create a test file in the temp directory
	testFile := "test_file.txt"
	testContent := "test content"
	err := os.WriteFile(tempDir+"/"+testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	var command, expectedOutput string
	if runtime.GOOS == "windows" {
		command = `type ` + testFile
		expectedOutput = testContent
	} else {
		command = `cat ` + testFile
		expectedOutput = testContent
	}

	base := runbatch.NewBaseCommand("cwd-test", tempDir, runbatch.RunOnSuccess, nil, nil)

	cmd, err := New(ctx, base, command)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	results := cmd.Run(ctxWithTimeout)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, 0, result.ExitCode,
		"command failed. StdOut: %s, StdErr: %s",
		string(result.StdOut), string(result.StdErr))
	assert.NoError(t, result.Error)

	output := string(result.StdOut)
	assert.Contains(t, output, expectedOutput,
		"expected file content not found. Full output: %s", output)
}

func TestCommandTimeout_Integration(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	base := runbatch.NewBaseCommand("timeout-test", "", runbatch.RunOnSuccess, nil, nil)

	var command string
	if runtime.GOOS == "windows" {
		command = `timeout /t 5`
	} else {
		command = `sleep 5`
	}

	cmd, err := New(ctx, base, command)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	// Use a short timeout to force cancellation
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	results := cmd.Run(ctxWithTimeout)
	require.Len(t, results, 1)

	result := results[0]
	// Should be killed due to timeout
	assert.Equal(t, -1, result.ExitCode,
		"expected command to be killed. StdOut: %s, StdErr: %s",
		string(result.StdOut), string(result.StdErr))
	assert.Error(t, result.Error, "expected error due to timeout")
}

func TestCommandFailure_Integration(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	testCases := []struct {
		name             string
		command          string
		expectedExitCode int
		skipOS           []string
	}{
		{
			name:             "exit with code 1",
			command:          `exit 1`,
			expectedExitCode: 1,
		},
		{
			name:             "exit with code 2",
			command:          `exit 2`,
			expectedExitCode: 2,
		},
		{
			name:             "command not found (unix)",
			command:          `nonexistent_command_12345`,
			expectedExitCode: 127, // Standard "command not found" exit code
			skipOS:           []string{"windows"},
		},
		{
			name:             "false command (unix)",
			command:          `false`,
			expectedExitCode: 1,
			skipOS:           []string{"windows"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip test if current OS is in the skipOS list
			for _, skipOS := range tc.skipOS {
				if runtime.GOOS == skipOS {
					t.Skipf("Skipping test on %s", runtime.GOOS)
					return
				}
			}

			base := runbatch.NewBaseCommand("failure-test", "", runbatch.RunOnSuccess, nil, nil)

			cmd, err := New(ctx, base, tc.command)
			require.NoError(t, err)
			require.NotNil(t, cmd)

			ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			results := cmd.Run(ctxWithTimeout)
			require.Len(t, results, 1)

			result := results[0]
			assert.Equal(t, tc.expectedExitCode, result.ExitCode,
				"unexpected exit code. StdOut: %s, StdErr: %s",
				string(result.StdOut), string(result.StdErr))
		})
	}
}
