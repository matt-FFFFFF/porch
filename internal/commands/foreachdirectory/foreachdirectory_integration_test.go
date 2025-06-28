// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands/copycwdtotemp"
	"github.com/matt-FFFFFF/porch/internal/commands/parallelcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/serialcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/shellcommand"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForEachDirectoryParallel(t *testing.T) {
	// register the command type
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
	f := commandregistry.New(
		serialcommand.Register,
		parallelcommand.Register,
		shellcommand.Register,
		copycwdtotemp.Register,
		Register,
	)

	absCwd, _ := filepath.Abs(".")

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			parent := &runbatch.SerialBatch{
				BaseCommand: &runbatch.BaseCommand{
					Label: "Test Parent",
					Cwd:   absCwd,
				},
			}
			yamlPayload := fmt.Sprintf(yamlPayloadFmt, tc.mode, tc.includeHidden)
			runnable, err := commander.Create(t.Context(), f, []byte(yamlPayload), parent)
			require.NoError(t, err)
			require.NotNil(t, runnable)
			forEachCommand, ok := runnable.(*runbatch.ForEachCommand)
			require.True(t, ok, "Expected ForEachCommand, got %T", runnable)
			assert.Equal(t, "For Each Directory", forEachCommand.Label)

			pwd, _ := os.Getwd()
			relPath, _ := filepath.Rel(pwd, forEachCommand.Cwd)
			assert.Equal(t, "testdata/foreachdir", relPath)
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
				require.Len(t, result.Children, 1, "Expected each directory to have 1 child command")
				res := result.Children[0]

				// Safely remove trailing newline if present
				stdout := string(res.StdOut)
				if len(stdout) > 0 && stdout[len(stdout)-1] == '\n' {
					stdout = stdout[:len(stdout)-1]
				}

				if _, ok := tc.expected[stdout]; ok {
					delete(tc.expected, stdout)
					continue
				}
			}

			require.Empty(t, tc.expected, "All expected items should have been processed")
		})
	}
}

func TestForEachDirectorySkipNotExist(t *testing.T) {
	yamlPayload := `type: "foreachdirectory"
name: "For Each Directory"
working_directory: "./does-not-exist"
mode: serial
depth: 1
skip_on_not_exist: true
include_hidden: false
commands:
  - type: "shell"
    name: "echo item var"
    command_line: "echo $ITEM"
`
	commander := &Commander{}
	f := commandregistry.New(
		serialcommand.Register,
		parallelcommand.Register,
		shellcommand.Register,
		copycwdtotemp.Register,
		Register,
	)

	parent := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: "Test Parent",
			Cwd:   t.TempDir(),
		},
	}

	runnable, err := commander.Create(t.Context(), f, []byte(yamlPayload), parent)
	require.NoError(t, err)
	require.NotNil(t, runnable)
	forEachCommand, ok := runnable.(*runbatch.ForEachCommand)
	require.True(t, ok, "Expected ForEachCommand, got %T", runnable)
	assert.Equal(t, forEachCommand.ItemsSkipOnErrors[0], os.ErrNotExist, "Expected skip error to be os.ErrNotExist")

	results := runnable.Run(t.Context())
	require.NotNil(t, results)
	require.Len(t, results, 1, "Expected 1 result for foreach command")
	assert.Equal(t, "For Each Directory", results[0].Label)
	assert.Equal(t, runbatch.ResultStatusSkipped, results[0].Status,
		"Expected result to be skipped due to non-existent working directory",
	)
}
