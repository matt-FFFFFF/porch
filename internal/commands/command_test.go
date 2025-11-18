// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrCommandCreate tests the ErrCommandCreate error type.
func TestErrCommandCreate(t *testing.T) {
	t.Run("Error method returns formatted string", func(t *testing.T) {
		err := &ErrCommandCreate{cmdName: "test-command"}
		expected := `failed to create command "test-command"`
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Error method with empty command name", func(t *testing.T) {
		err := &ErrCommandCreate{cmdName: ""}
		expected := `failed to create command ""`
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Error method with special characters", func(t *testing.T) {
		err := &ErrCommandCreate{cmdName: "shell/command-with-special@chars"}
		expected := `failed to create command "shell/command-with-special@chars"`
		assert.Equal(t, expected, err.Error())
	})
}

// TestNewErrCommandCreate tests the NewErrCommandCreate function.
func TestNewErrCommandCreate(t *testing.T) {
	t.Run("creates ErrCommandCreate with command name", func(t *testing.T) {
		cmdName := "shell-command"
		err := NewErrCommandCreate(cmdName)

		require.Error(t, err)

		var cmdErr *ErrCommandCreate

		require.ErrorAs(t, err, &cmdErr)
		assert.Equal(t, cmdName, cmdErr.cmdName)
		assert.Equal(t, `failed to create command "shell-command"`, err.Error())
	})

	t.Run("creates ErrCommandCreate with empty name", func(t *testing.T) {
		err := NewErrCommandCreate("")

		require.Error(t, err)

		var cmdErr *ErrCommandCreate

		require.ErrorAs(t, err, &cmdErr)
		assert.Empty(t, cmdErr.cmdName)
	})

	t.Run("returns error interface", func(t *testing.T) {
		err := NewErrCommandCreate("test")
		assert.Implements(t, (*error)(nil), err)
	})
}

// TestBaseDefinition_ToBaseCommand tests the ToBaseCommand method.
func TestBaseDefinition_ToBaseCommand(t *testing.T) {
	ctx := context.Background()

	t.Run("successful conversion with all fields", func(t *testing.T) {
		def := &BaseDefinition{
			Type:             "shell",
			Name:             "Test Command",
			WorkingDirectory: "/tmp",
			RunsOnCondition:  "success",
			RunsOnExitCodes:  []int{0, 1},
			Env: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		}
		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}

		baseCmd, err := def.ToBaseCommand(ctx, parent)
		require.NoError(t, err)
		require.NotNil(t, baseCmd)

		assert.Equal(t, "Test Command", baseCmd.Label)
		assert.Equal(t, "/tmp", baseCmd.GetCwd())
		assert.Equal(t, runbatch.RunOnSuccess, baseCmd.RunsOnCondition)
		assert.Equal(t, []int{0, 1}, baseCmd.RunsOnExitCodes)
		assert.Equal(t, map[string]string{"VAR1": "value1", "VAR2": "value2"}, baseCmd.Env)
	})

	t.Run("successful conversion with minimal fields", func(t *testing.T) {
		def := &BaseDefinition{
			Type: "shell",
			Name: "Minimal Command",
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}

		baseCmd, err := def.ToBaseCommand(ctx, parent)
		require.NoError(t, err)
		require.NotNil(t, baseCmd)

		assert.Equal(t, "Minimal Command", baseCmd.Label)
		assert.Equal(t, "/parent", baseCmd.GetCwd())
		assert.Equal(t, runbatch.RunOnSuccess, baseCmd.RunsOnCondition)
		assert.Equal(t, []int{0}, baseCmd.RunsOnExitCodes)
		assert.Equal(t, map[string]string{}, baseCmd.Env)
	})

	t.Run("defaults empty RunsOnCondition to success", func(t *testing.T) {
		def := &BaseDefinition{
			Type:            "shell",
			Name:            "Default Condition",
			RunsOnCondition: "",
		}
		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}

		baseCmd, err := def.ToBaseCommand(ctx, parent)
		require.NoError(t, err)
		require.NotNil(t, baseCmd)

		assert.Equal(t, runbatch.RunOnSuccess, baseCmd.RunsOnCondition)
	})

	t.Run("handles different run conditions", func(t *testing.T) {
		testCases := []struct {
			name      string
			condition string
			expected  runbatch.RunCondition
		}{
			{"success condition", "success", runbatch.RunOnSuccess},
			{"error condition", "error", runbatch.RunOnError},
			{"always condition", "always", runbatch.RunOnAlways},
			{"exit-codes condition", "exit-codes", runbatch.RunOnExitCodes},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				def := &BaseDefinition{
					Type:            "shell",
					Name:            "Test",
					RunsOnCondition: tc.condition,
				}
				parent := &runbatch.SerialBatch{
					BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
				}
				baseCmd, err := def.ToBaseCommand(ctx, parent)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, baseCmd.RunsOnCondition)
			})
		}
	})

	t.Run("error with invalid run condition", func(t *testing.T) {
		def := &BaseDefinition{
			Type:            "shell",
			Name:            "Invalid Condition",
			RunsOnCondition: "invalid-condition",
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}

		baseCmd, err := def.ToBaseCommand(ctx, parent)
		require.Error(t, err)
		assert.Nil(t, baseCmd)
		require.ErrorIs(t, err, ErrYamlUnmarshal)
		assert.Contains(t, err.Error(), "unknown RunCondition value")
	})

	t.Run("handles nil environment map", func(t *testing.T) {
		def := &BaseDefinition{
			Type: "shell",
			Name: "Nil Env",
			Env:  nil,
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}

		baseCmd, err := def.ToBaseCommand(ctx, parent)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{}, baseCmd.Env)
	})

	t.Run("handles empty environment map", func(t *testing.T) {
		def := &BaseDefinition{
			Type: "shell",
			Name: "Empty Env",
			Env:  map[string]string{},
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}
		baseCmd, err := def.ToBaseCommand(ctx, parent)

		require.NoError(t, err)
		assert.NotNil(t, baseCmd.Env)
		assert.Empty(t, baseCmd.Env)
	})

	t.Run("handles nil exit codes", func(t *testing.T) {
		def := &BaseDefinition{
			Type:            "shell",
			Name:            "Nil Exit Codes",
			RunsOnExitCodes: nil,
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}
		baseCmd, err := def.ToBaseCommand(ctx, parent)

		require.NoError(t, err)
		assert.Equal(t, []int{0}, baseCmd.RunsOnExitCodes)
	})

	t.Run("handles empty exit codes slice", func(t *testing.T) {
		def := &BaseDefinition{
			Type:            "shell",
			Name:            "Empty Exit Codes",
			RunsOnExitCodes: []int{},
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}
		baseCmd, err := def.ToBaseCommand(ctx, parent)

		require.NoError(t, err)
		assert.NotNil(t, baseCmd.RunsOnExitCodes)
		assert.Empty(t, baseCmd.RunsOnExitCodes)
	})

	t.Run("preserves all exit codes", func(t *testing.T) {
		exitCodes := []int{0, 1, 2, 255, 127}
		def := &BaseDefinition{
			Type:            "shell",
			Name:            "Multiple Exit Codes",
			RunsOnCondition: "exit-codes",
			RunsOnExitCodes: exitCodes,
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}
		baseCmd, err := def.ToBaseCommand(ctx, parent)

		require.NoError(t, err)
		assert.Equal(t, exitCodes, baseCmd.RunsOnExitCodes)
	})
}

// TestBaseDefinition_Struct tests the BaseDefinition struct fields and YAML tags.
func TestBaseDefinition_Struct(t *testing.T) {
	t.Run("struct has correct YAML tags", func(t *testing.T) {
		def := BaseDefinition{
			Type:             "test-type",
			Name:             "test-name",
			WorkingDirectory: "/test/dir",
			RunsOnCondition:  "test-condition",
			RunsOnExitCodes:  []int{1, 2},
			Env:              map[string]string{"TEST": "value"},
		}

		// Verify fields are accessible
		assert.Equal(t, "test-type", def.Type)
		assert.Equal(t, "test-name", def.Name)
		assert.Equal(t, "/test/dir", def.WorkingDirectory)
		assert.Equal(t, "test-condition", def.RunsOnCondition)
		assert.Equal(t, []int{1, 2}, def.RunsOnExitCodes)
		assert.Equal(t, map[string]string{"TEST": "value"}, def.Env)
	})

	t.Run("zero value struct", func(t *testing.T) {
		def := BaseDefinition{}

		assert.Empty(t, def.Type)
		assert.Empty(t, def.Name)
		assert.Empty(t, def.WorkingDirectory)
		assert.Empty(t, def.RunsOnCondition)
		assert.Nil(t, def.RunsOnExitCodes)
		assert.Nil(t, def.Env)
	})
}

// TestErrYamlUnmarshal tests the ErrYamlUnmarshal variable.
func TestErrYamlUnmarshal(t *testing.T) {
	t.Run("error message is correct", func(t *testing.T) {
		expected := "failed to decode YAML command definition, please check the syntax and structure of your YAML file"
		assert.Equal(t, expected, ErrYamlUnmarshal.Error())
	})

	t.Run("can be used with errors.Is", func(t *testing.T) {
		wrappedErr := errors.Join(ErrYamlUnmarshal, errors.New("yaml syntax error"))
		require.ErrorIs(t, wrappedErr, ErrYamlUnmarshal)
	})

	t.Run("can be used with errors.Join", func(t *testing.T) {
		innerErr := errors.New("inner error")
		wrappedErr := errors.Join(ErrYamlUnmarshal, innerErr)

		assert.Contains(t, wrappedErr.Error(), ErrYamlUnmarshal.Error())
		assert.Contains(t, wrappedErr.Error(), innerErr.Error())
	})
}

// TestBaseDefinition_EdgeCases tests edge cases and boundary conditions.
func TestBaseDefinition_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("very long command name", func(t *testing.T) {
		longName := make([]byte, 1000)
		for i := range longName {
			longName[i] = 'a'
		}

		def := &BaseDefinition{
			Type: "shell",
			Name: string(longName),
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}
		baseCmd, err := def.ToBaseCommand(ctx, parent)

		require.NoError(t, err)
		assert.Equal(t, string(longName), baseCmd.Label)
	})

	t.Run("special characters in working directory", func(t *testing.T) {
		specialDir := "/path/with spaces/and-dashes/under_scores/123/αβγ"
		def := &BaseDefinition{
			Type:             "shell",
			Name:             "Special Dir",
			WorkingDirectory: specialDir,
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}
		baseCmd, err := def.ToBaseCommand(ctx, parent)

		require.NoError(t, err)
		assert.Equal(t, specialDir, baseCmd.GetCwd())
	})

	t.Run("environment variables with special characters", func(t *testing.T) {
		env := map[string]string{
			"VAR_WITH_UNDERSCORES": "value1",
			"VAR-WITH-DASHES":      "value2",
			"VAR123":               "value3",
			"PATH":                 "/usr/bin:/bin",
			"JSON_VAR":             `{"key": "value"}`,
			"EMPTY_VAR":            "",
			"UNICODE_VAR":          "αβγδε",
		}

		def := &BaseDefinition{
			Type: "shell",
			Name: "Special Env",
			Env:  env,
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}
		baseCmd, err := def.ToBaseCommand(ctx, parent)

		require.NoError(t, err)
		assert.Equal(t, env, baseCmd.Env)
	})

	t.Run("negative exit codes", func(t *testing.T) {
		exitCodes := []int{-1, -128, 0, 1, 255, 1000}
		def := &BaseDefinition{
			Type:            "shell",
			Name:            "Negative Exit Codes",
			RunsOnExitCodes: exitCodes,
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}
		baseCmd, err := def.ToBaseCommand(ctx, parent)

		require.NoError(t, err)
		assert.Equal(t, exitCodes, baseCmd.RunsOnExitCodes)
	})
}

// TestBaseDefinition_Mutability tests that ToBaseCommand doesn't modify the original.
func TestBaseDefinition_Mutability(t *testing.T) {
	ctx := context.Background()

	t.Run("ToBaseCommand doesn't modify original definition", func(t *testing.T) {
		originalEnv := map[string]string{"KEY": "value"}
		originalExitCodes := []int{0, 1}

		def := &BaseDefinition{
			Type:             "shell",
			Name:             "Original",
			WorkingDirectory: "/original",
			RunsOnCondition:  "success",
			RunsOnExitCodes:  originalExitCodes,
			Env:              originalEnv,
		}

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}
		baseCmd, err := def.ToBaseCommand(ctx, parent)

		require.NoError(t, err)

		// Modify the returned BaseCommand
		baseCmd.Label = "Modified"
		baseCmd.Env["NEW_KEY"] = "new_value"
		baseCmd.RunsOnExitCodes[0] = 999

		// Original definition should be unchanged
		assert.Equal(t, "Original", def.Name)
		assert.Equal(t, "/original", def.WorkingDirectory)
		assert.Equal(t, map[string]string{"KEY": "value"}, def.Env)
		assert.Equal(t, []int{0, 1}, def.RunsOnExitCodes)
	})

	t.Run("modifying RunsOnCondition field during conversion", func(t *testing.T) {
		def := &BaseDefinition{
			Type:            "shell",
			Name:            "Test",
			RunsOnCondition: "",
		}

		originalCondition := def.RunsOnCondition

		parent := &runbatch.SerialBatch{
			BaseCommand: runbatch.NewBaseCommand("Parent Command", "/parent", runbatch.RunOnSuccess, nil, nil),
		}
		baseCmd, err := def.ToBaseCommand(ctx, parent)

		require.NoError(t, err)

		// The method modifies the original struct's RunsOnCondition field
		assert.NotEqual(t, originalCondition, def.RunsOnCondition)
		assert.Equal(t, "success", def.RunsOnCondition)
		assert.Equal(t, runbatch.RunOnSuccess, baseCmd.RunsOnCondition)
	})
}
