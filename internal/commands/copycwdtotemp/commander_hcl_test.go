// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import (
	"context"
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
			name: "valid HCL with default working directory",
			hclCommand: &hcl.CommandBlock{
				Type:            "copycwdtotemp",
				Name:            "copy-test",
				RunsOnCondition: "success",
			},
			expectError: false,
			validateResult: func(t *testing.T, runnable runbatch.Runnable) {
				funcCmd, ok := runnable.(*runbatch.FunctionCommand)
				require.True(t, ok, "expected FunctionCommand")
				assert.Contains(t, funcCmd.GetLabel(), "copy-test")
			},
		},
		{
			name: "valid HCL with specific working directory",
			hclCommand: &hcl.CommandBlock{
				Type:             "copycwdtotemp",
				Name:             "copy-specific",
				WorkingDirectory: "/custom/path",
			},
			expectError: false,
			validateResult: func(t *testing.T, runnable runbatch.Runnable) {
				funcCmd, ok := runnable.(*runbatch.FunctionCommand)
				require.True(t, ok, "expected FunctionCommand")
				assert.Contains(t, funcCmd.GetLabel(), "copy-specific")
			},
		},
		{
			name: "valid HCL with environment variables",
			hclCommand: &hcl.CommandBlock{
				Type: "copycwdtotemp",
				Name: "copy-with-env",
				Env: map[string]string{
					"TEST_VAR": "test_value",
				},
			},
			expectError: false,
		},
		{
			name: "valid HCL with runs on condition",
			hclCommand: &hcl.CommandBlock{
				Type:            "copycwdtotemp",
				Name:            "copy-conditional",
				RunsOnCondition: "success",
			},
			expectError: false,
		},
		{
			name: "valid HCL with runs on exit codes",
			hclCommand: &hcl.CommandBlock{
				Type:            "copycwdtotemp",
				Name:            "copy-exit-codes",
				RunsOnExitCodes: []int{0, 1},
			},
			expectError: false,
		},
		{
			name: "invalid runs on condition",
			hclCommand: &hcl.CommandBlock{
				Type:            "copycwdtotemp",
				Name:            "copy-invalid",
				RunsOnCondition: "invalid-condition",
			},
			expectError: true,
			errorType:   commands.ErrHclConfig,
		},
		{
			name: "empty working directory gets default",
			hclCommand: &hcl.CommandBlock{
				Type:             "copycwdtotemp",
				Name:             "copy-empty-wd",
				WorkingDirectory: "",
			},
			expectError: false,
			validateResult: func(t *testing.T, runnable runbatch.Runnable) {
				// Should work as empty working directory gets set to "."
				funcCmd, ok := runnable.(*runbatch.FunctionCommand)
				require.True(t, ok, "expected FunctionCommand")
				assert.Contains(t, funcCmd.GetLabel(), "copy-empty-wd")
			},
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
				require.NoError(t, err)
				assert.NotNil(t, runnable)

				if tc.validateResult != nil {
					tc.validateResult(t, runnable)
				}
			}
		})
	}
}
