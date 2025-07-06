// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package shellcommand

import (
	"context"
	"strings"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/config/hcl"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			name: "valid HCL with command line",
			hclCommand: &hcl.CommandBlock{
				Type:            "shell",
				Name:            "test-command",
				CommandLine:     "echo 'Hello World'",
				RunsOnCondition: "success",
			},
			expectError: false,
			validateResult: func(t *testing.T, runnable runbatch.Runnable) {
				osCmd, ok := runnable.(*runbatch.OSCommand)
				require.True(t, ok, "expected OSCommand")
				assert.Contains(t, osCmd.GetLabel(), "test-command")
			},
		},
		{
			name: "valid HCL with success exit codes",
			hclCommand: &hcl.CommandBlock{
				Type:             "shell",
				Name:             "test-command",
				CommandLine:      "echo 'test'",
				SuccessExitCodes: []int{0, 1, 2},
				RunsOnCondition:  "success",
			},
			expectError: false,
		},
		{
			name: "valid HCL with skip exit codes",
			hclCommand: &hcl.CommandBlock{
				Type:            "shell",
				Name:            "test-command",
				CommandLine:     "echo 'test'",
				SkipExitCodes:   []int{3, 4},
				RunsOnCondition: "success",
			},
			expectError: false,
		},
		{
			name: "valid HCL with working directory",
			hclCommand: &hcl.CommandBlock{
				Type:             "shell",
				Name:             "test-command",
				CommandLine:      "pwd",
				WorkingDirectory: "/tmp",
				RunsOnCondition:  "success",
			},
			expectError: false,
		},
		{
			name: "valid HCL with environment variables",
			hclCommand: &hcl.CommandBlock{
				Type:        "shell",
				Name:        "test-command",
				CommandLine: "env | grep TEST",
				Env: map[string]string{
					"TEST_VAR":    "test_value",
					"ANOTHER_VAR": "another_value",
				},
				RunsOnCondition: "success",
			},
			expectError: false,
		},
		{
			name: "valid HCL with runs on condition",
			hclCommand: &hcl.CommandBlock{
				Type:            "shell",
				Name:            "test-command",
				CommandLine:     "echo 'conditional'",
				RunsOnCondition: "success",
			},
			expectError: false,
		},
		{
			name: "valid HCL with runs on exit codes",
			hclCommand: &hcl.CommandBlock{
				Type:            "shell",
				Name:            "test-command",
				CommandLine:     "echo 'exit codes'",
				RunsOnExitCodes: []int{0, 1},
				RunsOnCondition: "success",
			},
			expectError: false,
		},
		{
			name: "invalid runs on condition",
			hclCommand: &hcl.CommandBlock{
				Type:            "shell",
				Name:            "test-command",
				CommandLine:     "echo 'test'",
				RunsOnCondition: "invalid-condition",
			},
			expectError: true,
			errorType:   commands.ErrHclConfig,
		},
		{
			name: "empty command line",
			hclCommand: &hcl.CommandBlock{
				Type:            "shell",
				Name:            "test-command",
				CommandLine:     "",
				RunsOnCondition: "success",
			},
			expectError: true, // Empty command line should fail
		},
	}

	parent := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: "parent-batch",
			Cwd:   "/",
		},
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
				// If shell is not available, the New function might fail
				if err != nil && strings.Contains(err.Error(), "executable file not found") {
					t.Skip("shell not available, skipping test")
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
