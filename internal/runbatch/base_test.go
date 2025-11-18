// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseCommand_PrependCwd(t *testing.T) {
	tests := []struct {
		name        string
		parentCwd   string
		initialCwd  string
		newCwd      string
		expectedCwd string
	}{
		{
			name:        "empty new cwd should not change current cwd",
			parentCwd:   "/initial/path",
			newCwd:      "",
			expectedCwd: "/initial/path",
		},
		{
			name:        "absolute new cwd should replace current cwd",
			parentCwd:   "/initial/path",
			newCwd:      "/new/absolute/path",
			expectedCwd: "/new/absolute/path",
		},
		{
			name:        "relative new cwd should be joined with current absolute cwd",
			parentCwd:   "/initial/path",
			newCwd:      "relative/subdir",
			expectedCwd: "/initial/path/relative/subdir",
		},
		{
			name:        "relative new cwd with ./ prefix should be joined correctly",
			parentCwd:   "/initial/path",
			newCwd:      "./relative/subdir",
			expectedCwd: "/initial/path/relative/subdir", // filepath.Join cleans the path
		},
		{
			name:        "empty initial cwd should treat as root for absolute new cwd",
			parentCwd:   "",
			newCwd:      "/new/path",
			expectedCwd: "/new/path",
		},
		{
			name:        "relative initial cwd with absolute new cwd should replace",
			parentCwd:   "relative/path",
			newCwd:      "/new/path",
			expectedCwd: "/new/path",
		},
		{
			name:        "empty initial cwd with relative new cwd should error",
			parentCwd:   "",
			newCwd:      "relative/path",
			expectedCwd: "relative/path",
		},
		{
			name:        "relative inital cwd with absolute new cwd should join",
			parentCwd:   "/initial/path",
			initialCwd:  "relative",
			newCwd:      "/tmp/foo",
			expectedCwd: "/tmp/foo/relative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := NewBaseCommand("parent", tt.parentCwd, RunOnAlways, nil, nil)
			cmd := NewBaseCommand("test", tt.initialCwd, RunOnAlways, nil, nil)
			cmd.parent = parent
			err := cmd.PrependCwd(tt.newCwd)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCwd, cmd.GetCwd())
		})
	}
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
			assert.Equal(t, tt.expectedCwd, cmd.GetCwd())
			assert.Equal(t, tt.expectedRunsOn, cmd.RunsOnCondition)
			assert.Equal(t, tt.expectedExitCodes, cmd.RunsOnExitCodes)
			assert.Len(t, cmd.Env, tt.expectedEnvLen)
			assert.NotNil(t, cmd.Env, "env map should never be nil")
		})
	}
}
