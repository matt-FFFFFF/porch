// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package pwshcommand

import (
	"context"
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// TestMain is used to run the goleak verification before and after tests.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestNew_UnitTests(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	testCases := []struct {
		name          string
		definition    *Definition
		expectedError error
		skipOS        []string
		setupFunc     func(t *testing.T) (cleanup func())
	}{
		{
			name: "valid script file definition",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
				},
				ScriptFile: "test.ps1",
			},
			expectedError: nil,
		},
		{
			name: "valid inline script definition",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
				},
				Script: "Write-Host 'Hello World'",
			},
			expectedError: nil,
		},
		{
			name: "both script and script file specified",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
				},
				Script:     "Write-Host 'Hello World'",
				ScriptFile: "test.ps1",
			},
			expectedError: ErrBothScriptAndScriptFileSpecified,
		},
		{
			name: "with success exit codes",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
				},
				ScriptFile:       "test.ps1",
				SuccessExitCodes: []int{0, 1},
			},
			expectedError: nil,
		},
		{
			name: "with skip exit codes",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
				},
				ScriptFile:    "test.ps1",
				SkipExitCodes: []int{2, 3},
			},
			expectedError: nil,
		},
		{
			name: "with working directory",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name:             "test-pwsh",
					WorkingDirectory: "/tmp",
				},
				ScriptFile: "test.ps1",
			},
			expectedError: nil,
		},
		{
			name: "with environment variables",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
					Env: map[string]string{
						"TEST_VAR": "test_value",
					},
				},
				ScriptFile: "test.ps1",
			},
			expectedError: nil,
		},
		{
			name: "empty script file path",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
				},
				ScriptFile: "",
			},
			expectedError: nil, // This should work, but the command might fail at runtime
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
			var cleanup func()
			if tc.setupFunc != nil {
				cleanup = tc.setupFunc(t)
				defer cleanup()
			}

			parent := &runbatch.SerialBatch{
				BaseCommand: &runbatch.BaseCommand{
					Label: "parent-batch",
					Cwd:   "/",
				},
			}

			cmd, err := New(ctx, tc.definition, parent)

			if tc.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedError)
				assert.Nil(t, cmd)

				return
			}

			if err != nil {
				// If pwsh is not found, that's expected on some systems
				if errors.Is(err, ErrCannotFindPwsh) {
					t.Skip("pwsh not found in PATH, skipping test")
					return
				}
			}

			require.NoError(t, err)
			assert.NotNil(t, cmd)

			// Verify it's an OSCommand
			osCmd, ok := cmd.(*runbatch.OSCommand)
			require.True(t, ok, "expected command to be OSCommand")

			// Verify command arguments
			assert.Len(t, osCmd.Args, 6)
			assert.Equal(t, "-File", osCmd.Args[4]) // Expecting -File argument for PowerShell script execution

			// Verify success and skip exit codes
			if tc.definition.SuccessExitCodes != nil {
				assert.Equal(t, tc.definition.SuccessExitCodes, osCmd.SuccessExitCodes)
			}

			if tc.definition.SkipExitCodes != nil {
				assert.Equal(t, tc.definition.SkipExitCodes, osCmd.SkipExitCodes)
			}

			// Verify base command properties
			assert.Equal(t, tc.definition.Name, osCmd.Label)

			if tc.definition.WorkingDirectory != "" {
				assert.Equal(t, tc.definition.WorkingDirectory, osCmd.Cwd)
			}

			if tc.definition.Env != nil {
				assert.Equal(t, tc.definition.Env, osCmd.Env)
			}
		})
	}
}

func TestNew_InlineScriptCreatesTemporaryFile(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	script := "Write-Host 'Hello from inline script'"
	definition := &Definition{
		BaseDefinition: commands.BaseDefinition{
			Name: "test-inline-script",
		},
		Script: script,
	}

	parent := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: "parent-batch",
			Cwd:   "/",
		},
	}

	cmd, err := New(ctx, definition, parent)
	if err != nil && errors.Is(err, ErrCannotFindPwsh) {
		t.Skip("pwsh not found in PATH, skipping test")
		return
	}

	require.NoError(t, err)
	require.NotNil(t, cmd)

	osCmd, ok := cmd.(*runbatch.OSCommand)
	require.True(t, ok)

	// Verify that a script file was created
	assert.NotEmpty(t, osCmd.Args[5], "script file path should be set")

	// Verify the temporary file exists and contains our script
	scriptFile := osCmd.Args[5]
	assert.FileExists(t, scriptFile)

	content, err := os.ReadFile(scriptFile)
	require.NoError(t, err)
	assert.Equal(t, script, string(content))

	// Clean up the temporary file
	os.Remove(scriptFile)
}

func TestNew_ExecutablePathSelection(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	definition := &Definition{
		BaseDefinition: commands.BaseDefinition{
			Name: "test-path",
		},
		ScriptFile: "test.ps1",
	}

	parent := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: "parent-batch",
			Cwd:   "/",
		},
	}

	cmd, err := New(ctx, definition, parent)
	if err != nil && assert.ErrorIs(t, err, ErrCannotFindPwsh) {
		t.Skip("pwsh not found in PATH, skipping test")
		return
	}

	require.NoError(t, err)
	require.NotNil(t, cmd)

	osCmd, ok := cmd.(*runbatch.OSCommand)
	require.True(t, ok)

	// Verify the executable path is set
	assert.NotEmpty(t, osCmd.Path)

	// On Windows, expect pwsh.exe, on other platforms expect pwsh
	if runtime.GOOS == goOSWindows {
		assert.Contains(t, osCmd.Path, "pwsh.exe")
	} else {
		assert.Contains(t, osCmd.Path, "pwsh")
	}
}

func TestNew_InvalidBaseDefinition(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	// Test with invalid base definition that would cause ToBaseCommand to fail
	definition := &Definition{
		BaseDefinition: commands.BaseDefinition{
			Name:            "", // Empty name should cause an error
			RunsOnCondition: "invalid condition syntax",
		},
		ScriptFile: "test.ps1",
	}

	parent := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: "parent-batch",
			Cwd:   "/",
		},
	}

	cmd, err := New(ctx, definition, parent)

	// The function should handle ToBaseCommand errors gracefully
	if err != nil {
		// If pwsh is not found, that error takes precedence
		if !errors.Is(err, ErrCannotFindPwsh) {
			// Otherwise, expect a command creation error
			var commandErr *commands.ErrCommandCreate

			require.ErrorAs(t, err, &commandErr)
		}

		assert.Nil(t, cmd)
	}
}

func TestErrorConstants(t *testing.T) {
	// Test that error constants are properly defined
	assert.NotEmpty(t, ErrCannotFindPwsh.Error())
	assert.NotEmpty(t, ErrBothScriptAndScriptFileSpecified.Error())
	assert.NotEmpty(t, ErrCannotCreateTempFile.Error())
	assert.NotEmpty(t, ErrCannotWriteTempFile.Error())

	// Test error messages are descriptive
	assert.Contains(t, ErrCannotFindPwsh.Error(), "pwsh")
	assert.Contains(t, ErrCannotFindPwsh.Error(), "PATH")
	assert.Contains(t, ErrBothScriptAndScriptFileSpecified.Error(), "script")
	assert.Contains(t, ErrBothScriptAndScriptFileSpecified.Error(), "scriptFile")
}

func TestCommandType(t *testing.T) {
	// Test that command type constant is properly defined
	assert.Equal(t, "pwsh", commandType)
}
