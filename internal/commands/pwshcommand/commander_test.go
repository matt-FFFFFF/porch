// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package pwshcommand

import (
	"context"
	"errors"
	"iter"
	"strings"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/config/hcl"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommander(t *testing.T) {
	commander := NewCommander()
	require.NotNil(t, commander)
	assert.NotNil(t, commander.schemaGenerator)
}

func TestCommander_GetCommandType(t *testing.T) {
	commander := NewCommander()
	commandType := commander.GetCommandType()
	assert.Equal(t, "pwsh", commandType)
}

func TestCommander_GetCommandDescription(t *testing.T) {
	commander := NewCommander()
	description := commander.GetCommandDescription()
	assert.NotEmpty(t, description)
	assert.Contains(t, description, "pwsh")
	assert.Contains(t, description, "script")
}

func TestCommander_GetExampleDefinition(t *testing.T) {
	commander := NewCommander()
	example := commander.GetExampleDefinition()
	require.NotNil(t, example)

	def, ok := example.(*Definition)
	require.True(t, ok, "expected example to be of type *Definition")

	assert.Equal(t, "pwsh", def.Type)
	assert.Equal(t, "example-pwsh-command", def.Name)
	assert.NotEmpty(t, def.ScriptFile)
	assert.NotEmpty(t, def.Script)
	assert.Contains(t, def.SuccessExitCodes, 0)
	assert.Contains(t, def.SkipExitCodes, 2)
}

func TestCommander_Create(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	commander := NewCommander()

	testCases := []struct {
		name        string
		yaml        string
		expectError bool
		errorType   error
	}{
		{
			name: "valid YAML with script file",
			yaml: `
type: pwsh
name: test-command
script_file: test.ps1
`,
			expectError: false,
		},
		{
			name: "valid YAML with inline script",
			yaml: `
type: pwsh
name: test-script
script: Write-Host "Hello World"
`,
			expectError: false,
		},
		{
			name: "valid YAML with success exit codes",
			yaml: `
type: pwsh
name: test-command
script_file: test.ps1
success_exit_codes: [0, 1, 2]
`,
			expectError: false,
		},
		{
			name: "valid YAML with skip exit codes",
			yaml: `
type: pwsh
name: test-command
script_file: test.ps1
skip_exit_codes: [3, 4]
`,
			expectError: false,
		},
		{
			name: "valid YAML with working directory",
			yaml: `
type: pwsh
name: test-command
script_file: test.ps1
working_directory: /tmp
`,
			expectError: false,
		},
		{
			name: "valid YAML with environment variables",
			yaml: `
type: pwsh
name: test-command
script_file: test.ps1
env:
  TEST_VAR: test_value
  ANOTHER_VAR: another_value
`,
			expectError: false,
		},
		{
			name: "invalid YAML syntax",
			yaml: `
type: pwsh
name: test-command
script_file: test.ps1
invalid_yaml: [
`,
			expectError: true,
			errorType:   commands.ErrYamlUnmarshal,
		},
		{
			name:        "empty YAML",
			yaml:        "",
			expectError: false, // Empty YAML should create a definition with default values
		},
	}

	parent := &runbatch.SerialBatch{
		BaseCommand: runbatch.NewBaseCommand("parent-batch", "/", runbatch.RunOnAlways, nil, nil),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runnable, err := commander.CreateFromYaml(ctx, nil, []byte(tc.yaml), parent)

			if tc.expectError {
				require.Error(t, err)

				if tc.errorType != nil {
					require.ErrorIs(t, err, tc.errorType)
				}

				assert.Nil(t, runnable)
			} else {
				// If pwsh is not available, the New function might fail
				if err != nil && strings.Contains(err.Error(), "pwsh") {
					t.Skip("pwsh not available, skipping test")
					return
				}

				require.NoError(t, err)
				assert.NotNil(t, runnable)
			}
		})
	}
}

func TestCommander_CreateFromHcl(t *testing.T) {
	ctx := context.Background()
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	commander := NewCommander()

	testCases := []struct {
		name           string
		hclCommand     *hcl.CommandBlock
		expectError    bool
		errorType      error
		validateResult func(t *testing.T, runnable runbatch.Runnable)
	}{
		{
			name: "valid HCL with script file",
			hclCommand: &hcl.CommandBlock{
				Type:       "pwsh",
				Name:       "test-command",
				ScriptFile: "test.ps1",
			},
			expectError: false,
			validateResult: func(t *testing.T, runnable runbatch.Runnable) {
				osCmd, ok := runnable.(*runbatch.OSCommand)
				require.True(t, ok, "expected OSCommand")
				assert.Contains(t, osCmd.GetLabel(), "test-command")
			},
		},
		{
			name: "valid HCL with inline script",
			hclCommand: &hcl.CommandBlock{
				Type:   "pwsh",
				Name:   "test-script",
				Script: "Write-Host \"Hello World\"",
			},
			expectError: false,
			validateResult: func(t *testing.T, runnable runbatch.Runnable) {
				osCmd, ok := runnable.(*runbatch.OSCommand)
				require.True(t, ok, "expected OSCommand")
				assert.Contains(t, osCmd.GetLabel(), "test-script")
			},
		},
		{
			name: "valid HCL with success exit codes",
			hclCommand: &hcl.CommandBlock{
				Type:             "pwsh",
				Name:             "test-command",
				ScriptFile:       "test.ps1",
				SuccessExitCodes: []int{0, 1, 2},
			},
			expectError: false,
		},
		{
			name: "valid HCL with skip exit codes",
			hclCommand: &hcl.CommandBlock{
				Type:          "pwsh",
				Name:          "test-command",
				ScriptFile:    "test.ps1",
				SkipExitCodes: []int{3, 4},
			},
			expectError: false,
		},
		{
			name: "valid HCL with working directory",
			hclCommand: &hcl.CommandBlock{
				Type:             "pwsh",
				Name:             "test-command",
				ScriptFile:       "test.ps1",
				WorkingDirectory: "/tmp",
			},
			expectError: false,
		},
		{
			name: "valid HCL with environment variables",
			hclCommand: &hcl.CommandBlock{
				Type:       "pwsh",
				Name:       "test-command",
				ScriptFile: "test.ps1",
				Env: map[string]string{
					"TEST_VAR":    "test_value",
					"ANOTHER_VAR": "another_value",
				},
			},
			expectError: false,
		},
		{
			name: "valid HCL with runs on condition",
			hclCommand: &hcl.CommandBlock{
				Type:            "pwsh",
				Name:            "test-command",
				ScriptFile:      "test.ps1",
				RunsOnCondition: "always",
			},
			expectError: false,
		},
		{
			name: "valid HCL with runs on exit codes",
			hclCommand: &hcl.CommandBlock{
				Type:            "pwsh",
				Name:            "test-command",
				ScriptFile:      "test.ps1",
				RunsOnExitCodes: []int{0, 1},
			},
			expectError: false,
		},
		{
			name: "invalid runs on condition",
			hclCommand: &hcl.CommandBlock{
				Type:            "pwsh",
				Name:            "test-command",
				ScriptFile:      "test.ps1",
				RunsOnCondition: "invalid-condition",
			},
			expectError: true,
			errorType:   commands.ErrHclConfig,
		},
	}

	parent := &runbatch.SerialBatch{
		BaseCommand: runbatch.NewBaseCommand("parent-batch", "/", runbatch.RunOnAlways, nil, nil),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runnable, err := commander.CreateFromHcl(ctx, nil, tc.hclCommand, parent)

			if tc.expectError {
				require.Error(t, err)

				if tc.errorType != nil {
					require.ErrorIs(t, err, tc.errorType)
				}

				assert.Nil(t, runnable)
			} else {
				// If pwsh is not available, the New function might fail
				if err != nil && strings.Contains(err.Error(), "pwsh") {
					t.Skip("pwsh not available, skipping test")
					return
				}

				require.NoError(t, err)
				assert.NotNil(t, runnable)

				if tc.validateResult != nil {
					tc.validateResult(t, runnable)
				}
			}
		})
	}
}

func TestCommander_GetSchemaFields(t *testing.T) {
	commander := NewCommander()
	fields := commander.GetSchemaFields()

	// Should return fields even if there might be errors in schema generation
	assert.NotNil(t, fields)
}

func TestCommander_WriteYAMLExample(t *testing.T) {
	commander := NewCommander()

	var buf strings.Builder

	err := commander.WriteYAMLExample(&buf)
	require.NoError(t, err)

	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "type: pwsh")
	assert.Contains(t, output, "name:")
}

func TestCommander_WriteMarkdownDoc(t *testing.T) {
	commander := NewCommander()

	var buf strings.Builder

	err := commander.WriteMarkdownDoc(&buf)
	require.NoError(t, err)

	output := buf.String()
	assert.NotEmpty(t, output)
	// Should contain markdown formatting and command information
	assert.Contains(t, output, "pwsh")
}

func TestCommander_WriteJSONSchema(t *testing.T) {
	commander := NewCommander()

	var buf strings.Builder

	// Create a mock factory for testing
	factory := &mockCommanderFactory{}
	err := commander.WriteJSONSchema(&buf, factory)

	// The result depends on the schema generator implementation
	// We just verify it doesn't panic and returns
	assert.NoError(t, err)
}

// mockCommanderFactory is a simple mock for testing.
type mockCommanderFactory struct{}

func (m *mockCommanderFactory) Get(commandType string) (commands.Commander, bool) {
	if commandType == "pwsh" {
		return NewCommander(), true
	}

	return nil, false
}

func (m *mockCommanderFactory) CreateRunnableFromHCL(
	ctx context.Context, hclCommand *hcl.CommandBlock, parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	// Simple mock implementation
	return nil, errors.New("not implemented in mock")
}

func (m *mockCommanderFactory) CreateRunnableFromYAML(
	ctx context.Context, payload []byte, parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	// Simple mock implementation
	return nil, errors.New("not implemented in mock")
}

func (m *mockCommanderFactory) Register(cmdtype string, commander commands.Commander) error {
	// Simple mock implementation
	return nil
}

func (m *mockCommanderFactory) Iter() iter.Seq2[string, commands.Commander] {
	// Simple mock implementation
	return func(yield func(string, commands.Commander) bool) {
		yield("pwsh", NewCommander())
	}
}

func (m *mockCommanderFactory) ResolveCommandGroup(groupName string) ([]any, error) {
	// Simple mock implementation
	return nil, errors.New("not implemented in mock")
}

func (m *mockCommanderFactory) AddCommandGroup(name string, commands []any) {
	// Simple mock implementation - do nothing
}
