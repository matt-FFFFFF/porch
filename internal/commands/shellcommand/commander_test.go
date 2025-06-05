// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package shellcommand

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/commands/copycwdtotemp"
	"github.com/matt-FFFFFF/porch/internal/commands/foreachdirectory"
	"github.com/matt-FFFFFF/porch/internal/commands/parallelcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/serialcommand"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	goOSWindows = "windows"
	goOSLinux   = "linux"
	goOSDarwin  = "darwin"
)

var testRegistry = commandregistry.New(
	serialcommand.Register,
	parallelcommand.Register,
	copycwdtotemp.Register,
	foreachdirectory.Register,
	Register,
)

func TestCommander_Create_Success(t *testing.T) {
	ctx := context.Background()
	commander := &Commander{}

	t.Run("simple command", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
name: "Simple Test"
command_line: "echo hello"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		osCommand, ok := runnable.(*runbatch.OSCommand)
		require.True(t, ok)
		assert.Equal(t, "Simple Test", osCommand.Label)
		assert.Contains(t, osCommand.Args, "echo hello")
	})

	t.Run("minimal required fields", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
command_line: "ls"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		osCommand, ok := runnable.(*runbatch.OSCommand)
		require.True(t, ok)
		assert.Contains(t, osCommand.Args, "ls")
	})

	t.Run("complex command with all fields", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
name: "Complex Test"
command_line: "echo test"
working_directory: "/tmp"
runs_on_condition: "success"
env:
  TEST_VAR: "test_value"
  ANOTHER_VAR: "another_value"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		osCommand, ok := runnable.(*runbatch.OSCommand)
		require.True(t, ok)
		assert.Equal(t, "Complex Test", osCommand.Label)
		assert.Contains(t, osCommand.Args, "echo test")
		assert.Equal(t, "/tmp", osCommand.Cwd)
		assert.Equal(t, runbatch.RunOnSuccess, osCommand.RunsOnCondition)
		assert.Contains(t, osCommand.Env, "TEST_VAR")
		assert.Equal(t, "test_value", osCommand.Env["TEST_VAR"])
		assert.Contains(t, osCommand.Env, "ANOTHER_VAR")
		assert.Equal(t, "another_value", osCommand.Env["ANOTHER_VAR"])
	})

	t.Run("command with error condition", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
name: "Error Test"
command_line: "false"
runs_on_condition: "error"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		osCommand, ok := runnable.(*runbatch.OSCommand)
		require.True(t, ok)
		assert.Equal(t, runbatch.RunOnError, osCommand.RunsOnCondition)
	})

	t.Run("command with always condition", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
name: "Always Test"
command_line: "echo always"
runs_on_condition: "always"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		osCommand, ok := runnable.(*runbatch.OSCommand)
		require.True(t, ok)
		assert.Equal(t, runbatch.RunOnAlways, osCommand.RunsOnCondition)
	})
}

// TestCommander_Create_Errors tests error conditions in Create method.
func TestCommander_Create_Errors(t *testing.T) {
	ctx := context.Background()
	commander := &Commander{}

	t.Run("invalid YAML", func(t *testing.T) {
		yamlPayload := []byte(`
invalid: yaml: content
  - malformed
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.Error(t, err)
		assert.Nil(t, runnable)
	})

	t.Run("empty command line", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
name: "Empty Command"
command_line: ""
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.Error(t, err)
		assert.Nil(t, runnable)
		assert.Contains(t, err.Error(), "command not found")
	})

	t.Run("missing command line", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
name: "Missing Command"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.Error(t, err)
		assert.Nil(t, runnable)
		assert.Contains(t, err.Error(), "command not found")
	})

	t.Run("invalid run condition", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
name: "Invalid Condition"
command_line: "echo test"
runs_on_condition: "invalid_condition"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.Error(t, err)
		assert.Nil(t, runnable)
		assert.Contains(t, err.Error(), "unknown RunCondition")
	})
}

// TestCommander_Interface tests that Commander implements the commands.Commander interface.
func TestCommander_Interface(t *testing.T) {
	var _ commands.Commander = &Commander{}
}

// TestDefinition_Structure tests the Definition struct fields and validation.
func TestDefinition_Structure(t *testing.T) {
	t.Run("definition with all fields", func(t *testing.T) {
		def := Definition{
			BaseDefinition: commands.BaseDefinition{
				Type:             "shell",
				Name:             "Test Command",
				WorkingDirectory: "/tmp",
				RunsOnCondition:  "success",
				RunsOnExitCodes:  []int{0, 1},
				Env:              map[string]string{"VAR": "value"},
			},
			CommandLine: "echo test",
		}

		assert.Equal(t, "shell", def.Type)
		assert.Equal(t, "Test Command", def.Name)
		assert.Equal(t, "echo test", def.CommandLine)
		assert.Equal(t, "/tmp", def.WorkingDirectory)
		assert.Equal(t, "success", def.RunsOnCondition)
		assert.Equal(t, []int{0, 1}, def.RunsOnExitCodes)
		assert.Equal(t, map[string]string{"VAR": "value"}, def.Env)
	})
}

// TestCommander_Create_EdgeCases tests edge cases and additional coverage.
func TestCommander_Create_EdgeCases(t *testing.T) {
	ctx := context.Background()
	commander := &Commander{}

	t.Run("empty environment map", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
command_line: "echo test"
environment: {}
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		osCommand, ok := runnable.(*runbatch.OSCommand)
		require.True(t, ok)
		assert.Empty(t, osCommand.Env)
	})

	t.Run("whitespace in command line", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
command_line: "  echo test  "
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		osCommand, ok := runnable.(*runbatch.OSCommand)
		require.True(t, ok)
		assert.Contains(t, osCommand.Args, "  echo test  ")
	})

	if runtime.GOOS == goOSWindows {
		t.Run("windows command", func(t *testing.T) {
			yamlPayload := []byte(`
type: shell
command_line: "dir"
`)

			runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
			require.NoError(t, err)
			require.NotNil(t, runnable)

			osCommand, ok := runnable.(*runbatch.OSCommand)
			require.True(t, ok)
			assert.Contains(t, osCommand.Args, "dir")
		})
	}
}

// TestDefaultShell_WindowsEdgeCases adds specific tests for Windows edge cases that may not be covered.
func TestDefaultShell_WindowsEdgeCases(t *testing.T) {
	if runtime.GOOS != goOSWindows {
		t.Skip("Windows-specific tests")
	}

	ctx := context.Background()

	t.Run("windows with empty SystemRoot", func(t *testing.T) {
		originalSystemRoot := os.Getenv(winSystemRootEnv)
		os.Setenv(winSystemRootEnv, "")

		defer func() {
			if originalSystemRoot != "" {
				os.Setenv(winSystemRootEnv, originalSystemRoot)
			} else {
				os.Unsetenv(winSystemRootEnv)
			}
		}()

		shell := defaultShell(ctx)
		expected := `C:\Windows\System32\cmd.exe`
		assert.Equal(t, expected, shell)
	})
}

// TestNew_ErrorCoverage ensures error paths are covered.
func TestNew_ErrorCoverage(t *testing.T) {
	ctx := context.Background()
	base := runbatch.NewBaseCommand("test", "", runbatch.RunOnSuccess, nil, nil)

	t.Run("empty command returns error", func(t *testing.T) {
		cmd, err := New(ctx, base, "", nil, nil)
		assert.Nil(t, cmd)
		require.ErrorIs(t, err, ErrCommandNotFound)
		assert.Equal(t, "command not found", err.Error())
	})
}

// TestCommander_CreateWithDifferentYAMLFormats tests various YAML input formats.
func TestCommander_CreateWithDifferentYAMLFormats(t *testing.T) {
	ctx := context.Background()
	commander := &Commander{}

	t.Run("YAML with different run conditions", func(t *testing.T) {
		testCases := []struct {
			condition string
			expected  runbatch.RunCondition
		}{
			{"success", runbatch.RunOnSuccess},
			{"error", runbatch.RunOnError},
			{"always", runbatch.RunOnAlways},
			{"exit-codes", runbatch.RunOnExitCodes},
		}

		for _, tc := range testCases {
			t.Run("condition_"+tc.condition, func(t *testing.T) {
				yamlPayload := []byte(`
type: shell
name: "Test Condition"
command_line: "echo test"
runs_on_condition: "` + tc.condition + `"
`)

				runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
				require.NoError(t, err)
				require.NotNil(t, runnable)

				osCommand, ok := runnable.(*runbatch.OSCommand)
				require.True(t, ok)

				assert.Equal(t, tc.expected, osCommand.RunsOnCondition)
			})
		}
	})

	t.Run("YAML with multiple environment variables", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
name: "Env Test"
command_line: "env"
env:
  VAR1: "value1"
  VAR2: "value2"
  VAR3: "value with spaces"
  VAR4: "value_with_special_chars_!@#$%"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		osCommand, ok := runnable.(*runbatch.OSCommand)
		require.True(t, ok)

		expectedEnv := map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value with spaces",
			"VAR4": "value_with_special_chars_!@#$%",
		}
		assert.Equal(t, expectedEnv, osCommand.Env)
	})

	t.Run("YAML with exit codes", func(t *testing.T) {
		yamlPayload := []byte(`
type: shell
name: "Exit Codes Test"
command_line: "exit 1"
runs_on_condition: "exit-codes"
runs_on_exit_codes: [0, 1, 2, 255]
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		osCommand, ok := runnable.(*runbatch.OSCommand)
		require.True(t, ok)

		assert.Equal(t, runbatch.RunOnExitCodes, osCommand.RunsOnCondition)
		assert.Equal(t, []int{0, 1, 2, 255}, osCommand.RunsOnExitCodes)
	})
}
