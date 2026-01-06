// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestBaseCommand_InheritEnv(t *testing.T) {
	tests := []struct {
		name        string
		initialEnv  map[string]string
		inheritEnv  map[string]string
		expectedEnv map[string]string
	}{
		{
			name:        "inherit_into_empty_env",
			initialEnv:  map[string]string{},
			inheritEnv:  map[string]string{"PARENT": "val"},
			expectedEnv: map[string]string{"PARENT": "val"},
		},
		{
			name:        "inherit_empty_env",
			initialEnv:  map[string]string{"CHILD": "val"},
			inheritEnv:  map[string]string{},
			expectedEnv: map[string]string{"CHILD": "val"},
		},
		{
			name:        "inherit_non_overlapping",
			initialEnv:  map[string]string{"CHILD": "val1"},
			inheritEnv:  map[string]string{"PARENT": "val2"},
			expectedEnv: map[string]string{"CHILD": "val1", "PARENT": "val2"},
		},
		{
			name:        "inherit_overlapping_child_precedence",
			initialEnv:  map[string]string{"SHARED": "child_val"},
			inheritEnv:  map[string]string{"SHARED": "parent_val", "PARENT": "val2"},
			expectedEnv: map[string]string{"SHARED": "child_val", "PARENT": "val2"},
		},
		{
			name:        "inherit_nil_env_into_empty",
			initialEnv:  map[string]string{},
			inheritEnv:  nil,
			expectedEnv: nil,
		},
		{
			name:        "inherit_nil_env_into_non_empty",
			initialEnv:  map[string]string{"CHILD": "val"},
			inheritEnv:  nil,
			expectedEnv: map[string]string{"CHILD": "val"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &BaseCommand{
				Env: tt.initialEnv,
			}

			cmd.InheritEnv(tt.inheritEnv)

			assert.Equal(t, tt.expectedEnv, cmd.Env)
		})
	}
}
