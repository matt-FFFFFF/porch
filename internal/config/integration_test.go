// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config_test

import (
	"context"
	"testing"

	"github.com/matt-FFFFFF/pporch/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFromYAML_ShellCommand(t *testing.T) {
	yamlData := `
name: "Test Shell Command"
description: "Test shell command execution"
commands:
  - type: "shell"
    name: "List Files"
    exec: "ls"
    args:
      - "-la"
    cwd: "/tmp"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, []byte(yamlData))

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
	runnable, err := config.BuildFromYAML(ctx, []byte(yamlData))

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
	_, err := config.BuildFromYAML(ctx, []byte(yamlData))

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
        exec: "echo"
        args:
          - "Starting workflow"

      - type: "parallel"
        name: "Parallel Tasks"
        commands:
          - type: "shell"
            name: "Task 1"
            exec: "echo"
            args:
              - "Task 1 running"

          - type: "shell"
            name: "Task 2"
            exec: "echo"
            args:
              - "Task 2 running"

      - type: "shell"
        name: "Cleanup"
        exec: "echo"
        args:
          - "Workflow complete"
`

	ctx := context.Background()
	runnable, err := config.BuildFromYAML(ctx, []byte(yamlData))

	require.NoError(t, err)
	assert.NotNil(t, runnable)
}
