// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config_test

import (
	"context"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands/copycwdtotemp"
	"github.com/matt-FFFFFF/porch/internal/commands/foreachdirectory"
	"github.com/matt-FFFFFF/porch/internal/commands/parallelcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/serialcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/shellcommand"
	"github.com/matt-FFFFFF/porch/internal/config"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testRegistry = commandregistry.New(
	serialcommand.Register,
	parallelcommand.Register,
	copycwdtotemp.Register,
	foreachdirectory.Register,
	shellcommand.Register,
)

func TestBuildFromYAML_ShellCommand(t *testing.T) {
	yamlData := `
name: "Test Shell Command"
description: "Test shell command execution"
commands:
  - type: "shell"
    name: "List Files"
    command_line: "ls -la"
    working_directory: "/tmp"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

	require.NoError(t, err)
	assert.NotNil(t, runnable)
}

func TestBuildFromYAML_CopyCommand(t *testing.T) {
	yamlData := `
name: "Test Copy Command"
description: "Test copy directory command"
commands:
  - type: "copycwdtotemp"
    name: "Copy Current Directory"
    cwd: "/tmp"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

	require.NoError(t, err)
	assert.NotNil(t, runnable)
}

func TestBuildFromYAML_UnknownCommandType(t *testing.T) {

	yamlData := `
name: "Test Unknown Command"
description: "Test unknown command type"
commands:
  - type: "unknown"
    name: "Unknown Command"
`

	ctx := context.Background()
	_, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command type")
}

func TestBuildFromYAML_ComplexWorkflow(t *testing.T) {
	yamlData := `
name: "Complex Workflow Example"
description: "Example showing nested serial and parallel commands"
commands:
  - type: "serial"
    name: "Main Workflow"
    commands:
      - type: "shell"
        name: "Setup"
        command_line: "echo \"Starting workflow\""

      - type: "parallel"
        name: "Parallel Tasks"
        commands:
          - type: "shell"
            name: "Task 1"
            command_line: "echo \"Task 1 running\""

          - type: "shell"
            name: "Task 2"
            command_line: "echo \"Task 2 running\""

      - type: "shell"
        name: "Cleanup"
        command_line: "echo \"Workflow complete\""
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

	require.NoError(t, err)
	assert.NotNil(t, runnable)
}

func TestSerialBatchSkipAndErrorHandling(t *testing.T) {
	yamlData := `
name: "Batch with Skip and Error Handling"
description: "Example showing skip and error handling in a complex batch"
commands:
  - type: "shell"
    name: "Inner Command 1"
    command_line: "echo 'inner command 1 success'"
  - type: "shell"
    name: "Inner Command 2"
    command_line: "/bin/notexist" # This will fail
  - type: "shell"
    name: "Inner Command 3 (should not run)"
    command_line: "echo 'inner command 3 success'"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))
	require.NoError(t, err)
	assert.NotNil(t, runnable)

	// Run the runnable and check results
	res := runnable.Run(ctx)
	require.Len(t, res, 1)
	assert.Len(t, res[0].Children, 3)
	require.Error(t, res[0].Error)
	assert.Equal(t, runbatch.ResultStatusError, res[0].Status)
	assert.Equal(t, -1, res[0].ExitCode)

	// Check that the second command failed and the third was skipped
	assert.Equal(t, "Inner Command 2", res[0].Children[1].Label)
	assert.Equal(t, runbatch.ResultStatusError, res[0].Children[1].Status)
	assert.Equal(t, "Inner Command 3 (should not run)", res[0].Children[2].Label)
	assert.Equal(t, runbatch.ResultStatusSkipped, res[0].Children[2].Status)
	require.ErrorIs(t, res[0].Children[2].Error, runbatch.ErrSkipOnError)
	//res.WriteText(os.Stdout)
}

func TestSerialBatchSkipOnExitCodeAndErrorHandling(t *testing.T) {
	yamlData := `
name: "Batch with Skip and Error Handling"
description: "Example showing skip and error handling in a complex batch"
commands:
  - type: "shell"
    name: "Inner Command 1"
    command_line: "echo 'inner command 1 success'"
  - type: "shell"
    name: "Inner Command 2"
    command_line: "exit 123" # This should succeed but skip the next command
    skip_exit_codes: [123]
  - type: "shell"
    name: "Inner Command 3 (should not run)"
    command_line: "echo 'inner command 3 success'"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))
	require.NoError(t, err)
	assert.NotNil(t, runnable)

	// Run the runnable and check results
	res := runnable.Run(ctx)
	require.Len(t, res, 1)
	assert.Len(t, res[0].Children, 3)
	require.NoError(t, res[0].Error)
	assert.Equal(t, runbatch.ResultStatusSuccess, res[0].Status)
	assert.Equal(t, 0, res[0].ExitCode)

	// Check that the second command failed and the third was skipped
	assert.Equal(t, "Inner Command 2", res[0].Children[1].Label)
	assert.Equalf(
		t,
		runbatch.ResultStatusSuccess,
		res[0].Children[1].Status,
		"Expected Inner Command 2 to succeed but skip the next command. Got %s", res[0].Children[1].Status,
	)
	assert.Equal(t, 123, res[0].Children[1].ExitCode)
	require.ErrorIs(t, res[0].Children[1].Error, runbatch.ErrSkipIntentional)
	assert.Equal(t, "Inner Command 3 (should not run)", res[0].Children[2].Label)
	assert.Equal(t, runbatch.ResultStatusSkipped, res[0].Children[2].Status)
	require.ErrorIs(t, res[0].Children[2].Error, runbatch.ErrSkipIntentional)
	//res.WriteText(os.Stdout)
}
