// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseCommand_SetCwd(t *testing.T) {
	tests := []struct {
		name         string
		initialCwd   string
		newCwd       string
		expectedCwd  string
		expectError  bool
		errorIsValue error
	}{
		{
			name:        "empty new cwd should not change current cwd",
			initialCwd:  "/initial/path",
			newCwd:      "",
			expectedCwd: "/initial/path",
			expectError: false,
		},
		{
			name:        "absolute new cwd should replace current cwd",
			initialCwd:  "/initial/path",
			newCwd:      "/new/absolute/path",
			expectedCwd: "/new/absolute/path",
			expectError: false,
		},
		{
			name:        "relative new cwd should be joined with current absolute cwd",
			initialCwd:  "/initial/path",
			newCwd:      "relative/subdir",
			expectedCwd: "/initial/path/relative/subdir",
			expectError: false,
		},
		{
			name:        "relative new cwd with ./ prefix should be joined correctly",
			initialCwd:  "/initial/path",
			newCwd:      "./relative/subdir",
			expectedCwd: "/initial/path/relative/subdir", // filepath.Join cleans the path
			expectError: false,
		},
		{
			name:         "empty initial cwd should error - no exceptions",
			initialCwd:   "",
			newCwd:       "/new/path",
			expectError:  true,
			errorIsValue: ErrSetCwd,
		},
		{
			name:         "relative initial cwd should error - all commands must have absolute cwd",
			initialCwd:   "relative/path",
			newCwd:       "/new/path",
			expectError:  true,
			errorIsValue: ErrSetCwd,
		},
		{
			name:         "empty initial cwd with relative new cwd should error",
			initialCwd:   "",
			newCwd:       "relative/path",
			expectError:  true,
			errorIsValue: ErrSetCwd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &BaseCommand{
				Cwd: tt.initialCwd,
				parent: &BaseCommand{
					Cwd: t.TempDir(),
				},
			}
			err := cmd.SetCwd(tt.newCwd)

			if tt.expectError {
				require.Error(t, err)

				if tt.errorIsValue != nil {
					require.ErrorIs(t, err, tt.errorIsValue)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCwd, cmd.Cwd)
		})
	}
}

// TestBaseCommand_SetCwd_CopyCwdToTempScenario tests the specific scenario that was broken:
// copycwdtotemp creates a temp directory and subsequent foreachdirectory with relative path
// should resolve correctly.
func TestBaseCommand_SetCwd_CopyCwdToTempScenario(t *testing.T) {
	// Simulate the scenario from the bug report:
	// 1. foreachdirectory command is created with working_directory: "./internal"
	//    which gets resolved to an absolute path at creation time
	workingDir, err := filepath.Abs("./internal")
	require.NoError(t, err)

	cmd := &BaseCommand{
		Label: "foreachdirectory-cmd",
		Cwd:   workingDir, // Now absolute from creation time
		parent: &BaseCommand{
			Cwd: t.TempDir(), // Parent command has a temp directory as its working directory
		},
	}

	// 2. copycwdtotemp runs and sets new working directory (absolute path)
	tempDir := "/tmp/porch_abc123"
	require.NoError(t, cmd.SetCwd(tempDir))

	// 3. The working directory should now be the temp directory (absolute path replaces absolute path)
	expectedCwd := "/tmp/porch_abc123"
	assert.Equal(t, expectedCwd, cmd.Cwd,
		"copycwdtotemp should update the absolute path to the temp directory")
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
			cmd := NewBaseCommand(tt.label, tt.cwd, "", tt.runsOn, tt.runOnExitCodes, tt.env)

			assert.Equal(t, tt.expectedLabel, cmd.Label)
			assert.Equal(t, tt.expectedCwd, cmd.Cwd)
			assert.Equal(t, tt.expectedRunsOn, cmd.RunsOnCondition)
			assert.Equal(t, tt.expectedExitCodes, cmd.RunsOnExitCodes)
			assert.Len(t, cmd.Env, tt.expectedEnvLen)
			assert.NotNil(t, cmd.Env, "env map should never be nil")
		})
	}
}
