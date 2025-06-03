// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import (
	"fmt"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands/shellcommand"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForEachDirectoryParallel(t *testing.T) {
	// register the command type
	commandregistry.Register(commandType, &shellcommand.Commander{})

	yamlPayloadFmt := `type: "foreachdirectory"
name: "For Each Directory"
working_directory: "testdata/foreachdir"
mode: %q
depth: 1
include_hidden: %t
commands:
  - type: "shell"
    name: "echo item var"
    command_line: "echo $ITEM"
`
	tcs := []struct {
		name          string
		includeHidden bool
		mode          string
		expected      map[string]struct{}
	}{
		{
			name:          "Parallel - Without Hidden",
			includeHidden: false,
			mode:          "parallel",
			expected: map[string]struct{}{
				"dir1": {},
				"dir2": {},
				"dir3": {},
			},
		},
		{
			name:          "Parallel - With Hidden",
			includeHidden: true,
			mode:          "parallel",
			expected: map[string]struct{}{
				"dir1":    {},
				"dir2":    {},
				"dir3":    {},
				".hidden": {},
			},
		},
		{
			name:          "Serial - Without Hidden",
			includeHidden: false,
			mode:          "serial",
			expected: map[string]struct{}{
				"dir1": {},
				"dir2": {},
				"dir3": {},
			},
		},
		{
			name:          "Serial - With Hidden",
			includeHidden: true,
			mode:          "serial",
			expected: map[string]struct{}{
				"dir1":    {},
				"dir2":    {},
				"dir3":    {},
				".hidden": {},
			},
		},
	}

	commander := &Commander{}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			yamlPayload := fmt.Sprintf(yamlPayloadFmt, tc.mode, tc.includeHidden)
			runnable, err := commander.Create(t.Context(), []byte(yamlPayload))
			require.NoError(t, err)
			require.NotNil(t, runnable)
			forEachCommand, ok := runnable.(*runbatch.ForEachCommand)
			require.True(t, ok, "Expected ForEachCommand, got %T", runnable)
			assert.Equal(t, "For Each Directory", forEachCommand.Label)
			assert.Equal(t, "testdata/foreachdir", forEachCommand.Cwd)
			require.Equalf(
				t,
				tc.mode,
				forEachCommand.Mode.String(),
				"Expected mode to be %q, got %q",
				tc.mode,
				forEachCommand.Mode.String(),
			)

			results := runnable.Run(t.Context())
			require.NotNil(t, results)
			require.Len(t, results, 1, "Expected 1 result for foreach command")
			results = results[0].Children
			assert.Lenf(t, results, len(tc.expected), "Expected %d directories to be processed", len(tc.expected))

			for _, result := range results {
				assert.Len(t, result.Children, 1, "Expected each directory to have 1 child command")
				res := result.Children[0]
				res.StdOut = res.StdOut[:len(res.StdOut)-1] // remove trailing newline

				if _, ok := tc.expected[string(res.StdOut)]; ok {
					delete(tc.expected, string(res.StdOut))
					continue
				}
			}

			require.Empty(t, tc.expected, "All expected items should have been processed")
		})
	}
}
