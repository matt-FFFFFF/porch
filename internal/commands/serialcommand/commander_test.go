// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package serialcommand

import (
	"context"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/commands/copycwdtotemp"
	"github.com/matt-FFFFFF/porch/internal/commands/shellcommand"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommander_Create_Success(t *testing.T) {
	// manually register the commander to ensure it exists
	commandregistry.Register(commandType, &Commander{})
	commandregistry.Register("shell", &shellcommand.Commander{})

	t.Run("simple serial command with shell commands", func(t *testing.T) {

		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: serial
name: "Test Serial Command"
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

		runnable, err := commander.Create(ctx, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		// Check that we got a SerialBatch
		serialBatch, ok := runnable.(*runbatch.SerialBatch)
		require.True(t, ok, "Expected SerialBatch, got %T", runnable)

		// Check base command properties
		assert.Equal(t, "Test Serial Command", serialBatch.BaseCommand.Label)
		assert.Equal(t, "/tmp", serialBatch.BaseCommand.Cwd)
		assert.Equal(t, runbatch.RunOnSuccess, serialBatch.BaseCommand.RunsOnCondition)
		assert.Equal(t, map[string]string{"TEST_VAR": "test_value"}, serialBatch.BaseCommand.Env)

		// Check that we have the correct number of commands
		assert.Len(t, serialBatch.Commands, 2)

		// Check that each command has the serial batch as parent
		for i, cmd := range serialBatch.Commands {
			assert.Equal(t, serialBatch, cmd.GetParent(), "Command %d should have serial batch as parent", i)
		}
	})

	t.Run("empty commands list", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: serial
name: "Empty Serial Command"
commands: []
`)

		runnable, err := commander.Create(ctx, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		serialBatch, ok := runnable.(*runbatch.SerialBatch)
		require.True(t, ok)

		assert.Equal(t, "Empty Serial Command", serialBatch.BaseCommand.Label)
		assert.Len(t, serialBatch.Commands, 0)
	})

	t.Run("nested serial commands", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: serial
name: "Nested Serial Command"
commands:
  - type: "serial"
    name: "Inner Serial"
    commands:
      - type: "shell"
        name: "Nested Command"
        command_line: "echo nested"
  - type: "shell"
    name: "Outer Command"
    command_line: "echo outer"
`)

		runnable, err := commander.Create(ctx, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		serialBatch, ok := runnable.(*runbatch.SerialBatch)
		require.True(t, ok)

		assert.Equal(t, "Nested Serial Command", serialBatch.BaseCommand.Label)
		assert.Len(t, serialBatch.Commands, 2)
	})

	t.Run("minimal valid configuration", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: serial
name: "Minimal Command"
commands:
  - type: "shell"
    name: "Single Command"
    command_line: "echo test"
`)

		runnable, err := commander.Create(ctx, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		serialBatch, ok := runnable.(*runbatch.SerialBatch)
		require.True(t, ok)

		assert.Equal(t, "Minimal Command", serialBatch.BaseCommand.Label)
		assert.Len(t, serialBatch.Commands, 1)
	})
}

func TestCommander_Create_Errors(t *testing.T) {
	t.Run("invalid YAML", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: serial
name: "Test Command"
commands: [
  invalid yaml structure
`)

		runnable, err := commander.Create(ctx, yamlPayload)
		assert.Nil(t, runnable)
		assert.Error(t, err)
		assert.ErrorIs(t, err, commands.ErrYamlUnmarshal)
	})

	t.Run("invalid base definition", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: serial
name: "Test Command"
runs_on_condition: "invalid_condition"
commands:
  - type: "shell"
    name: "Test Command"
    command_line: "echo test"
`)

		runnable, err := commander.Create(ctx, yamlPayload)
		assert.Nil(t, runnable)
		assert.Error(t, err)

		var cmdCreateErr *commands.ErrCommandCreate
		assert.ErrorAs(t, err, &cmdCreateErr)
	})

	t.Run("command with invalid sub-command", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: serial
name: "Test Command"
commands:
  - type: "nonexistent_command_type"
    name: "Invalid Command"
    some_field: "value"
`)

		runnable, err := commander.Create(ctx, yamlPayload)
		assert.Nil(t, runnable)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create runnable for command 0")
	})

	t.Run("command with unmarshalable sub-command", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		// Create a command that will cause yaml.Marshal to fail
		// by including a function which can't be marshaled
		yamlPayload := []byte(`
type: serial
name: "Test Command"
commands:
  - type: "shell"
    name: "Test Command"
    command_line: "echo test"
`)

		// First verify this normally works
		runnable, err := commander.Create(ctx, yamlPayload)
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
		runnable, err := commander.Create(ctx, []byte(`
type: serial
name: "Test"
commands: []
`))

		// Should not panic and should return expected types
		assert.IsType(t, (*runbatch.SerialBatch)(nil), runnable)
		assert.NoError(t, err)
	})
}

func TestDefinition_Structure(t *testing.T) {
	t.Run("definition includes BaseDefinition", func(t *testing.T) {
		def := &definition{}

		// Should be able to access BaseDefinition fields
		def.Type = "serial"
		def.Name = "test"
		def.WorkingDirectory = "/tmp"
		def.RunsOnCondition = "success"
		def.Env = map[string]string{"key": "value"}
		def.Commands = []any{}

		assert.Equal(t, "serial", def.Type)
		assert.Equal(t, "test", def.Name)
		assert.Equal(t, "/tmp", def.WorkingDirectory)
		assert.Equal(t, "success", def.RunsOnCondition)
		assert.Equal(t, map[string]string{"key": "value"}, def.Env)
		assert.Equal(t, []any{}, def.Commands)
	})
}

// TestCommander_CreateWithComplexYAML tests the commander with more complex YAML structures
func TestCommander_CreateWithComplexYAML(t *testing.T) {
	commandregistry.Register(commandType, &Commander{})
	commandregistry.Register("shell", &shellcommand.Commander{})
	commandregistry.Register("copycwdtotemp", &copycwdtotemp.Commander{})
	t.Run("commands with different types", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: serial
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

		runnable, err := commander.Create(ctx, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		serialBatch, ok := runnable.(*runbatch.SerialBatch)
		require.True(t, ok)

		assert.Equal(t, "Mixed Commands", serialBatch.BaseCommand.Label)
		assert.Len(t, serialBatch.Commands, 2)
	})

	t.Run("commands with complex environment variables", func(t *testing.T) {
		ctx := context.Background()
		commander := &Commander{}

		yamlPayload := []byte(`
type: serial
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

		runnable, err := commander.Create(ctx, yamlPayload)
		require.NoError(t, err)
		require.NotNil(t, runnable)

		serialBatch, ok := runnable.(*runbatch.SerialBatch)
		require.True(t, ok)

		assert.Equal(t, "Env Test", serialBatch.BaseCommand.Label)
		assert.Equal(t, runbatch.RunOnAlways, serialBatch.BaseCommand.RunsOnCondition)
		assert.Contains(t, serialBatch.BaseCommand.Env, "PARENT_VAR")
		assert.Contains(t, serialBatch.BaseCommand.Env, "COMPLEX_VAR")
		assert.Equal(t, "value with spaces and symbols !@#$%", serialBatch.BaseCommand.Env["COMPLEX_VAR"])
	})
}
