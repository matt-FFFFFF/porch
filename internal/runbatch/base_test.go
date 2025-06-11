// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseCommand_SetCwd(t *testing.T) {
	tests := []struct {
		name        string
		initialCwd  string
		newCwd      string
		overwrite   bool
		expectedCwd string
		description string
	}{
		{
			name:        "overwrite_false_with_existing_cwd",
			initialCwd:  "/existing/path",
			newCwd:      "/new/path",
			overwrite:   false,
			expectedCwd: "/existing/path",
			description: "should not overwrite existing cwd when overwrite is false",
		},
		{
			name:        "overwrite_false_with_empty_cwd",
			initialCwd:  "",
			newCwd:      "/new/path",
			overwrite:   false,
			expectedCwd: "/new/path",
			description: "should set cwd when current cwd is empty, even with overwrite false",
		},
		{
			name:        "overwrite_true_with_absolute_existing_cwd",
			initialCwd:  "/existing/absolute/path",
			newCwd:      "/new/path",
			overwrite:   true,
			expectedCwd: "/new/path",
			description: "should overwrite absolute cwd with new absolute path",
		},
		{
			name:        "overwrite_true_with_relative_existing_cwd",
			initialCwd:  "./relative/path",
			newCwd:      "/new/base/path",
			overwrite:   true,
			expectedCwd: "/new/base/path/relative/path",
			description: "should join new cwd with existing relative path when overwriting",
		},
		{
			name:        "overwrite_true_with_relative_existing_cwd_complex",
			initialCwd:  "../internal",
			newCwd:      "/tmp/porch_temp123",
			overwrite:   true,
			expectedCwd: "/tmp/internal",
			description: "should join new cwd with complex relative path when overwriting (filepath.Join cleans the path)",
		},
		{
			name:        "overwrite_true_with_dot_relative_path",
			initialCwd:  "./subdir",
			newCwd:      "/temp/workspace",
			overwrite:   true,
			expectedCwd: "/temp/workspace/subdir",
			description: "should handle dot-relative paths correctly",
		},
		{
			name:        "empty_new_cwd_no_overwrite",
			initialCwd:  "/existing/path",
			newCwd:      "",
			overwrite:   false,
			expectedCwd: "/existing/path",
			description: "should not change cwd when new cwd is empty",
		},
		{
			name:        "empty_new_cwd_with_overwrite",
			initialCwd:  "/existing/path",
			newCwd:      "",
			overwrite:   true,
			expectedCwd: "/existing/path",
			description: "should not change cwd when new cwd is empty, even with overwrite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &BaseCommand{
				Label: "test-command",
				Cwd:   tt.initialCwd,
			}

			cmd.SetCwd(tt.newCwd, tt.overwrite)

			assert.Equal(t, tt.expectedCwd, cmd.Cwd, tt.description)
		})
	}
}

// TestBaseCommand_SetCwd_CopyCwdToTempScenario tests the specific scenario that was broken:
// copycwdtotemp creates a temp directory and subsequent foreachdirectory with relative path
// should resolve correctly.
func TestBaseCommand_SetCwd_CopyCwdToTempScenario(t *testing.T) {
	// Simulate the scenario from the bug report:
	// 1. foreachdirectory command is created with working_directory: "./internal"
	cmd := &BaseCommand{
		Label: "foreachdirectory-cmd",
		Cwd:   "./internal",
	}

	// 2. copycwdtotemp runs and sets new working directory (with overwrite=true)
	tempDir := "/tmp/porch_abc123"
	cmd.SetCwd(tempDir, true)

	// 3. The working directory should now be the temp directory + the relative path
	expectedCwd := "/tmp/porch_abc123/internal"
	assert.Equal(t, expectedCwd, cmd.Cwd,
		"foreachdirectory with relative path should resolve to tempdir/internal after copycwdtotemp")
}

func TestBaseCommand_NewBaseCommand(t *testing.T) {
	tests := []struct {
		name              string
		label             string
		cwd               string
		runsOn            RunCondition
		runOnExitCodes    []int
		env               map[string]string
		expectedLabel     string
		expectedCwd       string
		expectedRunsOn    RunCondition
		expectedExitCodes []int
		expectedEnvLen    int
	}{
		{
			name:              "basic_creation",
			label:             "test-command",
			cwd:               "/test/path",
			runsOn:            RunOnSuccess,
			runOnExitCodes:    []int{0, 1},
			env:               map[string]string{"TEST": "value"},
			expectedLabel:     "test-command",
			expectedCwd:       "/test/path",
			expectedRunsOn:    RunOnSuccess,
			expectedExitCodes: []int{0, 1},
			expectedEnvLen:    1,
		},
		{
			name:              "nil_exit_codes_defaults_to_zero",
			label:             "test-command",
			cwd:               "/test/path",
			runsOn:            RunOnSuccess,
			runOnExitCodes:    nil,
			env:               map[string]string{"TEST": "value"},
			expectedLabel:     "test-command",
			expectedCwd:       "/test/path",
			expectedRunsOn:    RunOnSuccess,
			expectedExitCodes: []int{0},
			expectedEnvLen:    1,
		},
		{
			name:              "nil_env_creates_empty_map",
			label:             "test-command",
			cwd:               "/test/path",
			runsOn:            RunOnSuccess,
			runOnExitCodes:    []int{0},
			env:               nil,
			expectedLabel:     "test-command",
			expectedCwd:       "/test/path",
			expectedRunsOn:    RunOnSuccess,
			expectedExitCodes: []int{0},
			expectedEnvLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewBaseCommand(tt.label, tt.cwd, tt.runsOn, tt.runOnExitCodes, tt.env)

			assert.Equal(t, tt.expectedLabel, cmd.Label)
			assert.Equal(t, tt.expectedCwd, cmd.Cwd)
			assert.Equal(t, tt.expectedRunsOn, cmd.RunsOnCondition)
			assert.Equal(t, tt.expectedExitCodes, cmd.RunsOnExitCodes)
			assert.Len(t, cmd.Env, tt.expectedEnvLen)
			assert.NotNil(t, cmd.Env, "env map should never be nil")
		})
	}
}
