// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/config"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandGroups_SerialWithCommandGroup(t *testing.T) {
	yamlData := `
name: "Test Command Groups with Serial"
description: "Test command groups functionality with serial execution"
command_groups:
  - name: "test_commands"
    description: "A group of test shell commands"
    commands:
      - type: "shell"
        name: "Echo Hello"
        command_line: "echo 'Hello from command group'"
      - type: "shell"
        name: "Echo World"
        command_line: "echo 'World from command group'"
commands:
  - type: "serial"
    name: "Execute Test Commands"
    command_group: "test_commands"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

	require.NoError(t, err)
	assert.NotNil(t, runnable)

	// Execute the runnable to verify it works
	results := runnable.Run(ctx)
	require.Len(t, results, 1)
	assert.Equal(t, runbatch.ResultStatusSuccess, results[0].Status)
}

func TestCommandGroups_ParallelWithCommandGroup(t *testing.T) {
	yamlData := `
name: "Test Command Groups with Parallel"
description: "Test command groups functionality with parallel execution"
command_groups:
  - name: "parallel_test_commands"
    description: "A group of test shell commands for parallel execution"
    commands:
      - type: "shell"
        name: "Parallel Task 1"
        command_line: "echo 'Parallel task 1'"
      - type: "shell"
        name: "Parallel Task 2"
        command_line: "echo 'Parallel task 2'"
      - type: "shell"
        name: "Parallel Task 3"
        command_line: "echo 'Parallel task 3'"
commands:
  - type: "parallel"
    name: "Execute Parallel Commands"
    command_group: "parallel_test_commands"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

	require.NoError(t, err)
	assert.NotNil(t, runnable)

	// Execute the runnable to verify it works
	results := runnable.Run(ctx)
	require.Len(t, results, 1)
	assert.Equal(t, runbatch.ResultStatusSuccess, results[0].Status)
}

func TestCommandGroups_ForEachDirectoryWithCommandGroup(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "porch_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create subdirectories
	subDirs := []string{"dir1", "dir2", "dir3"}
	for _, dir := range subDirs {
		err := os.Mkdir(filepath.Join(tempDir, dir), 0755)
		require.NoError(t, err)
	}

	yamlData := `
name: "Test Command Groups with ForEachDirectory"
description: "Test command groups functionality with foreach directory execution"
command_groups:
  - name: "directory_commands"
    description: "Commands to run in each directory"
    commands:
      - type: "shell"
        name: "List Directory"
        command_line: "pwd"
      - type: "shell"
        name: "Echo Directory Name"
        command_line: "echo 'Processing directory:' $(basename $(pwd))"
commands:
  - type: "foreachdirectory"
    name: "Process All Directories"
    mode: "serial"
    depth: 1
    include_hidden: false
    working_directory_strategy: "item_relative"
    working_directory: "` + tempDir + `"
    command_group: "directory_commands"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

	require.NoError(t, err)
	assert.NotNil(t, runnable)

	// Execute the runnable to verify it works
	// Note: foreachdirectory creates one result per directory processed
	results := runnable.Run(ctx)
	require.NotEmpty(t, results, "should have at least one result")
	// Check that all results succeeded
	for i, result := range results {
		assert.Equal(t, runbatch.ResultStatusSuccess, result.Status, "result %d should have succeeded", i)
	}
}

func TestCommandGroups_NestedCommandGroups(t *testing.T) {
	yamlData := `
name: "Test Nested Command Groups"
description: "Test command groups with nested serial and parallel commands"
command_groups:
  - name: "setup_commands"
    description: "Setup commands"
    commands:
      - type: "shell"
        name: "Setup Step 1"
        command_line: "echo 'Setting up environment'"
      - type: "shell"
        name: "Setup Step 2"
        command_line: "echo 'Configuration complete'"
  - name: "cleanup_commands"
    description: "Cleanup commands"
    commands:
      - type: "shell"
        name: "Cleanup Step 1"
        command_line: "echo 'Cleaning up resources'"
      - type: "shell"
        name: "Cleanup Step 2"
        command_line: "echo 'Cleanup complete'"
commands:
  - type: "serial"
    name: "Main Workflow"
    commands:
      - type: "serial"
        name: "Setup Phase"
        command_group: "setup_commands"
      - type: "shell"
        name: "Main Task"
        command_line: "echo 'Executing main task'"
      - type: "parallel"
        name: "Cleanup Phase"
        command_group: "cleanup_commands"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

	require.NoError(t, err)
	assert.NotNil(t, runnable)

	// Execute the runnable to verify it works
	results := runnable.Run(ctx)
	require.Len(t, results, 1)
	assert.Equal(t, runbatch.ResultStatusSuccess, results[0].Status)
}

func TestCommandGroups_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		yamlData    string
		expectedErr string
	}{
		{
			name: "nonexistent command group",
			yamlData: `
name: "Test Nonexistent Command Group"
description: "Test error handling for nonexistent command group"
commands:
  - type: "serial"
    name: "Execute Nonexistent Group"
    command_group: "nonexistent_group"
`,
			expectedErr: "unknown command group",
		},
		{
			name: "both commands and command_group specified",
			yamlData: `
name: "Test Invalid Configuration"
description: "Test error handling for invalid configuration"
command_groups:
  - name: "test_group"
    description: "Test group"
    commands:
      - type: "shell"
        name: "Test Command"
        command_line: "echo 'test'"
commands:
  - type: "serial"
    name: "Invalid Command"
    command_group: "test_group"
    commands:
      - type: "shell"
        name: "Inline Command"
        command_line: "echo 'inline'"
`,
			expectedErr: "cannot specify both 'commands' and 'command_group'",
		},
		{
			name: "empty command group name",
			yamlData: `
name: "Test Empty Command Group"
description: "Test error handling for empty command group name"
commands:
  - type: "serial"
    name: "Execute Empty Group"
    command_group: "   "
`,
			expectedErr: "command_group cannot be empty or whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := config.BuildFromYAML(ctx, testRegistry, []byte(tt.yamlData))

			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.expectedErr))
		})
	}
}

func TestCommandGroups_MultipleCommandGroups(t *testing.T) {
	yamlData := `
name: "Test Multiple Command Groups"
description: "Test configuration with multiple command groups"
command_groups:
  - name: "group_a"
    description: "Command group A"
    commands:
      - type: "shell"
        name: "Group A Task 1"
        command_line: "echo 'Group A: Task 1'"
      - type: "shell"
        name: "Group A Task 2"
        command_line: "echo 'Group A: Task 2'"
  - name: "group_b"
    description: "Command group B"
    commands:
      - type: "shell"
        name: "Group B Task 1"
        command_line: "echo 'Group B: Task 1'"
      - type: "shell"
        name: "Group B Task 2"
        command_line: "echo 'Group B: Task 2'"
commands:
  - type: "serial"
    name: "Execute All Groups"
    commands:
      - type: "parallel"
        name: "Run Group A"
        command_group: "group_a"
      - type: "serial"
        name: "Run Group B"
        command_group: "group_b"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

	require.NoError(t, err)
	assert.NotNil(t, runnable)

	// Execute the runnable to verify it works
	results := runnable.Run(ctx)
	require.Len(t, results, 1)
	assert.Equal(t, runbatch.ResultStatusSuccess, results[0].Status)
}
