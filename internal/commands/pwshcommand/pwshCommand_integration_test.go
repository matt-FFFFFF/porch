// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package pwshcommand

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// TestMain is used to run the goleak verification before and after tests.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// checkPwshAvailable checks if pwsh is available on the system.
func checkPwshAvailable(t *testing.T) {
	execName := "pwsh"
	if runtime.GOOS == goOSWindows {
		execName = "pwsh.exe"
	}

	_, err := exec.LookPath(execName)
	if err != nil {
		t.Skipf("PowerShell Core (pwsh) not available: %v", err)
	}
}

func TestPowerShellCommandExecution_Integration(t *testing.T) {
	checkPwshAvailable(t)

	t.Parallel()

	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	testCases := []struct {
		name             string
		script           string
		expectedOutput   string // Expected substring in output
		expectedExitCode int
		skipOS           []string                        // Operating systems to skip this test on
		setupFunc        func(t *testing.T) string       // Optional setup function that returns cleanup path
		cleanupFunc      func(t *testing.T, path string) // Optional cleanup function
	}{
		{
			name:             "simple write-host command",
			script:           `Write-Host "Hello World"`,
			expectedOutput:   "Hello World",
			expectedExitCode: 0,
		},
		{
			name:             "powershell variables",
			script:           `$name = "PowerShell"; Write-Host "Hello from $name"`,
			expectedOutput:   "Hello from PowerShell",
			expectedExitCode: 0,
		},
		{
			name:             "multiple commands",
			script:           `Write-Host "Line 1"; Write-Host "Line 2"`,
			expectedOutput:   "Line 1",
			expectedExitCode: 0,
		},
		{
			name:             "arithmetic operations",
			script:           `$result = 5 + 3; Write-Host "Result: $result"`,
			expectedOutput:   "Result: 8",
			expectedExitCode: 0,
		},
		{
			name:             "string manipulation",
			script:           `$text = "hello world"; Write-Host $text.ToUpper()`,
			expectedOutput:   "HELLO WORLD",
			expectedExitCode: 0,
		},
		{
			name:             "array operations",
			script:           `$arr = @(1, 2, 3); Write-Host "Array length: $($arr.Length)"`,
			expectedOutput:   "Array length: 3",
			expectedExitCode: 0,
		},
		{
			name:             "conditional statements",
			script:           `if (5 -gt 3) { Write-Host "Five is greater than three" }`,
			expectedOutput:   "Five is greater than three",
			expectedExitCode: 0,
		},
		{
			name:             "loops",
			script:           `1..3 | ForEach-Object { Write-Host "Number: $_" }`,
			expectedOutput:   "Number: 1",
			expectedExitCode: 0,
		},
		{
			name:             "error handling - try catch",
			script:           `try { throw "Test error" } catch { Write-Host "Caught: $($_.Exception.Message)" }`,
			expectedOutput:   "Caught: Test error",
			expectedExitCode: 0,
		},
		{
			name:             "working with objects",
			script:           `$obj = [PSCustomObject]@{Name="Test"; Value=42}; Write-Host "Name: $($obj.Name), Value: $($obj.Value)"`, //nolint:lll
			expectedOutput:   "Name: Test, Value: 42",
			expectedExitCode: 0,
		},
		{
			name:             "unicode characters",
			script:           `Write-Host "Unicode: 世界 🌍"`,
			expectedOutput:   "Unicode: 世界 🌍",
			expectedExitCode: 0,
		},
		{
			name:             "exit with specific code",
			script:           `Write-Host "Exiting with code 42"; exit 42`,
			expectedOutput:   "Exiting with code 42",
			expectedExitCode: 42,
		},
		{
			name:             "output to error stream",
			script:           `Write-Error "This is an error message" -ErrorAction Continue; Write-Host "Script completed"`,
			expectedOutput:   "Script completed",
			expectedExitCode: 0,
		},
		{
			name:             "environment variable access",
			script:           `Write-Host "PATH exists: $([bool]$env:PATH)"`,
			expectedOutput:   "PATH exists: True",
			expectedExitCode: 0,
		},
		{
			name:             "cmdlet pipeline",
			script:           `@("apple", "banana", "cherry") | Sort-Object | ForEach-Object { Write-Host "Fruit: $_" }`,
			expectedOutput:   "Fruit: apple",
			expectedExitCode: 0,
		},
		{
			name:             "json handling",
			script:           `$obj = @{name="test"; value=123} | ConvertTo-Json -Compress; Write-Host "JSON: $obj"`,
			expectedOutput:   `"name":"test"`,
			expectedExitCode: 0,
		},
		{
			name:   "file operations",
			script: `"Test content" | Out-File -FilePath "test_output.txt" -Encoding UTF8; Get-Content "test_output.txt" | Write-Host`, //nolint:lll
			cleanupFunc: func(t *testing.T, _ string) {
				os.Remove("test_output.txt")
			},
			expectedOutput:   "Test content",
			expectedExitCode: 0,
		},
		{
			name:             "regex operations",
			script:           `$text = "Hello123World"; if ($text -match "\d+") { Write-Host "Found numbers: $($matches[0])" }`,
			expectedOutput:   "Found numbers: 123",
			expectedExitCode: 0,
		},
		{
			name:             "hashtable operations",
			script:           `$hash = @{key1="value1"; key2="value2"}; Write-Host "Key1: $($hash.key1)"`,
			expectedOutput:   "Key1: value1",
			expectedExitCode: 0,
		},
		{
			name:             "string formatting",
			script:           `$name = "World"; Write-Host ("Hello, {0}!" -f $name)`,
			expectedOutput:   "Hello, World!",
			expectedExitCode: 0,
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

			definition := &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "integration-test",
				},
				Script: tc.script,
			}

			// Create the command
			cmd, err := New(ctx, definition)
			require.NoError(t, err)
			require.NotNil(t, cmd)

			results := cmd.Run(ctx)
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

func TestPowerShellWithScriptFile_Integration(t *testing.T) {
	checkPwshAvailable(t)

	t.Parallel()

	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	// Create a temporary PowerShell script file
	tempDir := t.TempDir()
	scriptFile := tempDir + "/test_script.ps1"

	scriptContent := `param($Name = "World")
Write-Host "Hello, $Name!"
Write-Host "Script file executed successfully"
$result = 2 + 3
Write-Host "Calculation result: $result"`

	err := os.WriteFile(scriptFile, []byte(scriptContent), 0644)
	require.NoError(t, err)

	definition := &Definition{
		BaseDefinition: commands.BaseDefinition{
			Name:             "script-file-test",
			WorkingDirectory: tempDir,
		},
		ScriptFile: scriptFile,
	}

	cmd, err := New(ctx, definition)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	results := cmd.Run(ctx)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, 0, result.ExitCode,
		"command failed. StdOut: %s, StdErr: %s",
		string(result.StdOut), string(result.StdErr))
	require.NoError(t, result.Error)

	output := string(result.StdOut)
	assert.Contains(t, output, "Hello, World!")
	assert.Contains(t, output, "Script file executed successfully")
	assert.Contains(t, output, "Calculation result: 5")
}

func TestPowerShellWithEnvironmentVariables_Integration(t *testing.T) {
	checkPwshAvailable(t)

	t.Parallel()

	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	testCases := []struct {
		name           string
		env            map[string]string
		script         string
		expectedOutput string
	}{
		{
			name:           "single environment variable",
			env:            map[string]string{"TEST_VAR": "hello_world"},
			script:         `Write-Host "Value: $env:TEST_VAR"`,
			expectedOutput: "Value: hello_world",
		},
		{
			name: "multiple environment variables",
			env: map[string]string{
				"FIRST_VAR":  "first",
				"SECOND_VAR": "second",
			},
			script:         `Write-Host "$env:FIRST_VAR and $env:SECOND_VAR"`,
			expectedOutput: "first and second",
		},
		{
			name:           "environment variable with special characters",
			env:            map[string]string{"SPECIAL_VAR": "hello@world#123"},
			script:         `Write-Host "Special: $env:SPECIAL_VAR"`,
			expectedOutput: "Special: hello@world#123",
		},
		{
			name:           "numeric environment variable",
			env:            map[string]string{"NUMERIC_VAR": "42"},
			script:         `$num = [int]$env:NUMERIC_VAR; Write-Host "Number plus 8: $($num + 8)"`,
			expectedOutput: "Number plus 8: 50",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			definition := &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "env-test",
					Env:  tc.env,
				},
				Script: tc.script,
			}

			cmd, err := New(ctx, definition)
			require.NoError(t, err)
			require.NotNil(t, cmd)

			results := cmd.Run(ctx)
			require.Len(t, results, 1)

			result := results[0]
			assert.Equal(t, 0, result.ExitCode,
				"command failed. StdOut: %s, StdErr: %s",
				string(result.StdOut), string(result.StdErr))
			require.NoError(t, result.Error)

			output := string(result.StdOut)
			assert.Contains(t, output, tc.expectedOutput,
				"expected output not found. Full output: %s", output)
		})
	}
}

func TestPowerShellWithWorkingDirectory_Integration(t *testing.T) {
	checkPwshAvailable(t)

	t.Parallel()

	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	tempDir := t.TempDir()

	// Create a test file in the temp directory
	testFile := "test_file.txt"
	testContent := "PowerShell test content"
	err := os.WriteFile(tempDir+"/"+testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	script := `Get-Content "` + testFile + `" | Write-Host`

	definition := &Definition{
		BaseDefinition: commands.BaseDefinition{
			Name:             "cwd-test",
			WorkingDirectory: tempDir,
		},
		Script: script,
	}

	cmd, err := New(ctx, definition)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	results := cmd.Run(ctx)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, 0, result.ExitCode,
		"command failed. StdOut: %s, StdErr: %s",
		string(result.StdOut), string(result.StdErr))
	require.NoError(t, result.Error)

	output := string(result.StdOut)
	assert.Contains(t, output, testContent,
		"expected file content not found. Full output: %s", output)
}

func TestPowerShellTimeout_Integration(t *testing.T) {
	checkPwshAvailable(t)

	t.Parallel()

	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	definition := &Definition{
		BaseDefinition: commands.BaseDefinition{
			Name: "timeout-test",
		},
		Script: `Start-Sleep -Seconds 5; Write-Host "This should not appear"`,
	}

	cmd, err := New(ctx, definition)
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
	require.Error(t, result.Error, "expected error due to timeout")
}

func TestPowerShellFailure_Integration(t *testing.T) {
	checkPwshAvailable(t)

	t.Parallel()

	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	testCases := []struct {
		name             string
		script           string
		expectedExitCode int
	}{
		{
			name:             "exit with code 1",
			script:           `Write-Host "Exiting with error"; [Environment]::Exit(1)`,
			expectedExitCode: 1,
		},
		{
			name:             "exit with code 2",
			script:           `Write-Host "Exiting with code 2"; [Environment]::Exit(2)`,
			expectedExitCode: 2,
		},
		{
			name:             "terminating error",
			script:           `throw "This is a terminating error"`,
			expectedExitCode: 1,
		},
		{
			name:             "divide by zero",
			script:           `try { 1 / 0 } catch { Write-Host "Error caught"; [Environment]::Exit(1) }`,
			expectedExitCode: 1,
		},
		{
			name:             "non-existent cmdlet",
			script:           `try { NonExistentCmdlet } catch { Write-Host "Error caught"; [Environment]::Exit(1) }`,
			expectedExitCode: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			definition := &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "failure-test",
				},
				Script: tc.script,
			}

			cmd, err := New(ctx, definition)
			require.NoError(t, err)
			require.NotNil(t, cmd)

			results := cmd.Run(ctx)
			require.Len(t, results, 1)

			result := results[0]
			assert.Equal(t, tc.expectedExitCode, result.ExitCode,
				"unexpected exit code. StdOut: %s, StdErr: %s",
				string(result.StdOut), string(result.StdErr))
		})
	}
}

func TestPowerShellWithSuccessExitCodes_Integration(t *testing.T) {
	checkPwshAvailable(t)

	t.Parallel()

	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	definition := &Definition{
		BaseDefinition: commands.BaseDefinition{
			Name: "custom-success-codes",
		},
		Script:           `Write-Host "Exiting with code 42"; [Environment]::Exit(42)`,
		SuccessExitCodes: []int{42, 43},
	}

	cmd, err := New(ctx, definition)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	results := cmd.Run(ctx)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, 42, result.ExitCode)
}

func TestPowerShellLongOutput_Integration(t *testing.T) {
	checkPwshAvailable(t)

	t.Parallel()

	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	// Generate a script that produces a lot of output
	script := `1..100 | ForEach-Object { Write-Host "Line ${_}: ` + strings.Repeat("A", 50) + `" }`

	definition := &Definition{
		BaseDefinition: commands.BaseDefinition{
			Name: "long-output-test",
		},
		Script: script,
	}

	cmd, err := New(ctx, definition)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	results := cmd.Run(ctx)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, 0, result.ExitCode,
		"command failed. StdOut: %s, StdErr: %s",
		string(result.StdOut), string(result.StdErr))
	require.NoError(t, result.Error)

	output := string(result.StdOut)
	assert.Contains(t, output, "Line 1:")
	assert.Contains(t, output, "Line 100:")
}
