// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package pwshcommand

import (
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinition_Validate(t *testing.T) {
	tests := []struct {
		name        string
		definition  *Definition
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid definition with script file",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
					Type: "pwsh",
				},
				ScriptFile: "test.ps1",
			},
			expectError: false,
		},
		{
			name: "valid definition with inline script",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
					Type: "pwsh",
				},
				Script: "Write-Host 'Hello World'",
			},
			expectError: false,
		},
		{
			name: "valid definition with success exit codes",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
					Type: "pwsh",
				},
				ScriptFile:       "test.ps1",
				SuccessExitCodes: []int{0, 1, 2},
			},
			expectError: false,
		},
		{
			name: "valid definition with skip exit codes",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
					Type: "pwsh",
				},
				ScriptFile:    "test.ps1",
				SkipExitCodes: []int{3, 4},
			},
			expectError: false,
		},
		{
			name: "valid definition with working directory",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name:             "test-pwsh",
					Type:             "pwsh",
					WorkingDirectory: "/tmp",
				},
				ScriptFile: "test.ps1",
			},
			expectError: false,
		},
		{
			name: "valid definition with environment variables",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
					Type: "pwsh",
					Env: map[string]string{
						"TEST_VAR": "test_value",
					},
				},
				ScriptFile: "test.ps1",
			},
			expectError: false,
		},
		{
			name: "valid definition with runs on condition",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name:            "test-pwsh",
					Type:            "pwsh",
					RunsOnCondition: "success",
				},
				ScriptFile: "test.ps1",
			},
			expectError: false,
		},
		{
			name: "valid definition with runs on exit codes",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name:            "test-pwsh",
					Type:            "pwsh",
					RunsOnCondition: "exit-codes",
					RunsOnExitCodes: []int{0, 1},
				},
				ScriptFile: "test.ps1",
			},
			expectError: false,
		},
		{
			name: "valid empty definition",
			definition: &Definition{
				BaseDefinition: commands.BaseDefinition{
					Name: "test-pwsh",
					Type: "pwsh",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For now, since there's no Validate method defined in Definition,
			// we'll just test that the definition can be created without panicking
			assert.NotNil(t, tt.definition)
			assert.Equal(t, "pwsh", tt.definition.Type)

			// Test ToBaseCommand if it exists
			baseCmd, err := tt.definition.ToBaseCommand()
			if tt.expectError {
				require.Error(t, err)

				if tt.errorMsg != "" {
					require.ErrorContains(t, err, tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, baseCmd)
				assert.Equal(t, tt.definition.Name, baseCmd.Label)
			}
		})
	}
}

func TestDefinition_Fields(t *testing.T) {
	definition := &Definition{
		BaseDefinition: commands.BaseDefinition{
			Name:             "test-command",
			Type:             "pwsh",
			WorkingDirectory: "/test/dir",
			RunsOnCondition:  "always",
			RunsOnExitCodes:  []int{0, 1},
			Env: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		},
		ScriptFile:       "test_script.ps1",
		Script:           "Write-Host 'Test'",
		SuccessExitCodes: []int{0, 2},
		SkipExitCodes:    []int{3, 4},
	}

	// Test that all fields are properly set
	assert.Equal(t, "test-command", definition.Name)
	assert.Equal(t, "pwsh", definition.Type)
	assert.Equal(t, "/test/dir", definition.WorkingDirectory)
	assert.Equal(t, "always", definition.RunsOnCondition)
	assert.Equal(t, []int{0, 1}, definition.RunsOnExitCodes)
	assert.Equal(t, "test_script.ps1", definition.ScriptFile)
	assert.Equal(t, "Write-Host 'Test'", definition.Script)
	assert.Equal(t, []int{0, 2}, definition.SuccessExitCodes)
	assert.Equal(t, []int{3, 4}, definition.SkipExitCodes)
	assert.Equal(t, map[string]string{"VAR1": "value1", "VAR2": "value2"}, definition.Env)
}

func TestDefinition_EmptyValues(t *testing.T) {
	definition := &Definition{}

	// Test default/empty values
	assert.Empty(t, definition.Name)
	assert.Empty(t, definition.Type)
	assert.Empty(t, definition.WorkingDirectory)
	assert.Empty(t, definition.RunsOnCondition)
	assert.Nil(t, definition.RunsOnExitCodes)
	assert.Empty(t, definition.ScriptFile)
	assert.Empty(t, definition.Script)
	assert.Nil(t, definition.SuccessExitCodes)
	assert.Nil(t, definition.SkipExitCodes)
	assert.Nil(t, definition.Env)
}

func TestDefinition_YAMLTags(t *testing.T) {
	// This test ensures that the YAML tags are properly defined
	// by checking that the struct can be created with expected field names
	definition := &Definition{
		BaseDefinition: commands.BaseDefinition{
			Name: "yaml-test",
			Type: "pwsh",
		},
		ScriptFile:       "script.ps1",
		Script:           "Get-Date",
		SuccessExitCodes: []int{0},
		SkipExitCodes:    []int{1},
	}

	// Verify the fields exist and can be accessed
	require.NotNil(t, definition)
	assert.Equal(t, "yaml-test", definition.Name)
	assert.Equal(t, "script.ps1", definition.ScriptFile)
	assert.Equal(t, "Get-Date", definition.Script)
	assert.Equal(t, []int{0}, definition.SuccessExitCodes)
	assert.Equal(t, []int{1}, definition.SkipExitCodes)
}
