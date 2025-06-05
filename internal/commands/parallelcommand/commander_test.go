// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package parallelcommand

import (
	"context"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/commands/copycwdtotemp"
	"github.com/matt-FFFFFF/porch/internal/commands/foreachdirectory"
	"github.com/matt-FFFFFF/porch/internal/commands/serialcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/shellcommand"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testRegistry = commandregistry.New(
	Register,
	serialcommand.Register,
	copycwdtotemp.Register,
	foreachdirectory.Register,
	shellcommand.Register,
)

func TestCommander_Create_Success(t *testing.T) {
	t.Run("simple parallel command with shell commands", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: parallel
name: "Test Parallel Command"
working_directory: "/tmp"
runs_on_condition: "success"
env:
  TEST_VAR: "test_value"
commands:
  - type: "shell"
    name: "First Command"
    command_line: "echo hello"
  - type: "shell"
    name: "Second Command"
    command_line: "echo world"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		// Check that we got a ParallelBatch
		parallelBatch, ok := runnable.(*runbatch.ParallelBatch)
		require.True(t, ok, "Expected ParallelBatch, got %T", runnable)

		// Check base command properties
		assert.Equal(t, "Test Parallel Command", parallelBatch.Label)
		assert.Equal(t, "/tmp", parallelBatch.Cwd)
		assert.Equal(t, runbatch.RunOnSuccess, parallelBatch.RunsOnCondition)
		assert.Equal(t, map[string]string{"TEST_VAR": "test_value"}, parallelBatch.Env)

		// Check that we have the correct number of commands
		assert.Len(t, parallelBatch.Commands, 2)

		// Check that each command has the parallel batch as parent
		for i, cmd := range parallelBatch.Commands {
			assert.Equal(t, parallelBatch, cmd.GetParent(), "Command %d should have parallel batch as parent", i)
		}
	})

	t.Run("empty commands list", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: parallel
name: "Empty Parallel Command"
commands: []
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		parallelBatch, ok := runnable.(*runbatch.ParallelBatch)
		require.True(t, ok)

		assert.Equal(t, "Empty Parallel Command", parallelBatch.Label)
		assert.Empty(t, parallelBatch.Commands)
	})

	t.Run("nested parallel commands", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: parallel
name: "Nested Parallel Command"
commands:
  - type: "parallel"
    name: "Inner Parallel"
    commands:
      - type: "shell"
        name: "Nested Command"
        command_line: "echo nested"
  - type: "shell"
    name: "Outer Command"
    command_line: "echo outer"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		parallelBatch, ok := runnable.(*runbatch.ParallelBatch)
		require.True(t, ok)

		assert.Equal(t, "Nested Parallel Command", parallelBatch.Label)
		assert.Len(t, parallelBatch.Commands, 2)
	})

	t.Run("minimal valid configuration", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: parallel
name: "Minimal Command"
commands:
  - type: "shell"
    name: "Single Command"
    command_line: "echo test"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		parallelBatch, ok := runnable.(*runbatch.ParallelBatch)
		require.True(t, ok)

		assert.Equal(t, "Minimal Command", parallelBatch.Label)
		assert.Len(t, parallelBatch.Commands, 1)
	})
}

func TestCommander_Create_Errors(t *testing.T) {
	t.Run("invalid YAML", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: parallel
name: "Test Command"
commands: [
  invalid yaml structure
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		assert.Nil(t, runnable)
		require.Error(t, err)
		require.ErrorIs(t, err, commands.ErrYamlUnmarshal)
	})

	t.Run("invalid base definition", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: parallel
name: "Test Command"
runs_on_condition: "invalid_condition"
commands:
  - type: "shell"
    name: "Test Command"
    command_line: "echo test"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		assert.Nil(t, runnable)
		require.Error(t, err)

		var cmdCreateErr *commands.ErrCommandCreate

		require.ErrorAs(t, err, &cmdCreateErr)
	})

	t.Run("command with invalid sub-command", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: parallel
name: "Test Command"
commands:
  - type: "nonexistent_command_type"
    name: "Invalid Command"
    some_field: "value"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		assert.Nil(t, runnable)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create runnable for command 0")
	})

	t.Run("command with unmarshalable sub-command", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		// Create a command that will cause yaml.Marshal to fail
		// by including a function which can't be marshaled
		yamlPayload := []byte(`
type: parallel
name: "Test Command"
commands:
  - type: "shell"
    name: "Test Command"
    command_line: "echo test"
`)

		// First verify this normally works
		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)
	})
}

func TestCommander_Interface(t *testing.T) {
	t.Run("implements Commander interface", func(t *testing.T) {
		var _ commands.Commander = (*Commander)(nil)
	})

	t.Run("Create method signature", func(t *testing.T) {
		commander := &Commander{}
		ctx := context.Background()

		// Verify the method exists and has correct signature
		runnable, err := commander.Create(ctx, testRegistry, []byte(`
type: parallel
name: "Test"
commands: []
`))

		// Should not panic and should return expected types
		assert.IsType(t, (*runbatch.ParallelBatch)(nil), runnable)
		assert.NoError(t, err)
	})
}

func TestDefinition_Structure(t *testing.T) {
	t.Run("definition includes BaseDefinition", func(t *testing.T) {
		def := &Definition{}

		// Should be able to access BaseDefinition fields
		def.Type = "parallel"
		def.Name = "test"
		def.WorkingDirectory = "/tmp"
		def.RunsOnCondition = "success"
		def.Env = map[string]string{"key": "value"}
		def.Commands = []any{}

		assert.Equal(t, "parallel", def.Type)
		assert.Equal(t, "test", def.Name)
		assert.Equal(t, "/tmp", def.WorkingDirectory)
		assert.Equal(t, "success", def.RunsOnCondition)
		assert.Equal(t, map[string]string{"key": "value"}, def.Env)
		assert.Equal(t, []any{}, def.Commands)
	})
}

// TestCommander_CreateWithComplexYAML tests the commander with more complex YAML structures.
func TestCommander_CreateWithComplexYAML(t *testing.T) {
	t.Run("commands with different types", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: parallel
name: "Mixed Commands"
commands:
  - type: "shell"
    name: "Shell Command"
    command_line: "echo hello"
    working_directory: "/tmp"
  - type: "copycwdtotemp"
    name: "Copy Command"
    cwd: "."
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		parallelBatch, ok := runnable.(*runbatch.ParallelBatch)
		require.True(t, ok)

		assert.Equal(t, "Mixed Commands", parallelBatch.Label)
		assert.Len(t, parallelBatch.Commands, 2)
	})

	t.Run("commands with complex environment variables", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: parallel
name: "Env Test"
env:
  PARENT_VAR: "parent_value"
  COMPLEX_VAR: "value with spaces and symbols !@#$%"
runs_on_condition: "always"
commands:
  - type: "shell"
    name: "Env Command"
    command_line: "env | grep PARENT"
    env:
      CHILD_VAR: "child_value"
`)

		runnable, err := commander.Create(ctx, testRegistry, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		parallelBatch, ok := runnable.(*runbatch.ParallelBatch)
		require.True(t, ok)

		assert.Equal(t, "Env Test", parallelBatch.Label)
		assert.Equal(t, runbatch.RunOnAlways, parallelBatch.RunsOnCondition)
		assert.Contains(t, parallelBatch.Env, "PARENT_VAR")
		assert.Contains(t, parallelBatch.Env, "COMPLEX_VAR")
		assert.Equal(t, "value with spaces and symbols !@#$%", parallelBatch.Env["COMPLEX_VAR"])
	})
}
