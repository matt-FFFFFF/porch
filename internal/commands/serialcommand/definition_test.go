// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package serialcommand

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefinition_Validate(t *testing.T) {
	tests := []struct {
		name          string
		commands      []any
		commandGroup  string
		expectedError error
	}{
		{
			name:          "valid with commands only",
			commands:      []any{"cmd1", "cmd2"},
			commandGroup:  "",
			expectedError: nil,
		},
		{
			name:          "valid with command group only",
			commands:      nil,
			commandGroup:  "test_group",
			expectedError: nil,
		},
		{
			name:          "valid with empty commands and no command group",
			commands:      []any{},
			commandGroup:  "",
			expectedError: nil,
		},
		{
			name:          "invalid with both commands and command group",
			commands:      []any{"cmd1"},
			commandGroup:  "test_group",
			expectedError: ErrBothCommandsAndGroup,
		},
		{
			name:          "invalid with empty command group",
			commands:      nil,
			commandGroup:  "",
			expectedError: nil, // This should be valid now
		},
		{
			name:          "invalid with whitespace-only command group",
			commands:      nil,
			commandGroup:  "   ",
			expectedError: ErrEmptyCommandGroup,
		},
		{
			name:          "invalid with tab-only command group",
			commands:      nil,
			commandGroup:  "\t",
			expectedError: ErrEmptyCommandGroup,
		},
		{
			name:          "invalid with newline-only command group",
			commands:      nil,
			commandGroup:  "\n",
			expectedError: ErrEmptyCommandGroup,
		},
		{
			name:          "valid with command group containing spaces around valid name",
			commands:      nil,
			commandGroup:  "  valid_group  ",
			expectedError: nil, // Trimming happens during validation, but this should still be considered valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &Definition{
				Commands:     tt.commands,
				CommandGroup: tt.commandGroup,
			}

			err := def.Validate()
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
