// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCopyCwdToTempWithNewCwd tests the CopyCwdToTemp command with a new working directory.
func TestCopyCwdToTempWithNewCwd(t *testing.T) {
	requiredTree := []string{
		"./subdir",
		"./subdir/test2.txt",
		"./test.txt",
	}
	cwd, _ := os.Getwd()
	path := path.Join(cwd, "testdata/copyCwdToTemp")
	base := runbatch.NewBaseCommand("copyCwdToTemp", path, runbatch.RunOnAlways, nil, nil)
	copyCommand := New(base)
	pwdCommand := &runbatch.OSCommand{
		BaseCommand: runbatch.NewBaseCommand("pwd", path, runbatch.RunOnAlways, nil, nil),
		Path: "/bin/sh",
		Args: []string{"-c", "pwd"},
	}
	checkFilesCommand := &runbatch.OSCommand{
		BaseCommand: runbatch.NewBaseCommand("checkFiles", path, runbatch.RunOnAlways, nil, nil),
		Path: "/usr/bin/find",
		Args: []string{"."},
	}
	serialCommands := &runbatch.SerialBatch{
		BaseCommand: runbatch.NewBaseCommand("test", path, runbatch.RunOnAlways, nil, nil),
		Commands: []runbatch.Runnable{
			copyCommand,
			pwdCommand,
			checkFilesCommand,
		},
	}

	for _, cmd := range serialCommands.Commands {
		cmd.SetParent(serialCommands)
	}

	results := serialCommands.Run(context.Background())
	assert.Len(t, results, 1)
	assert.Equalf(t, 0, results[0].ExitCode, "Expected exit code 0, got %d", results[0].ExitCode)
	require.NoErrorf(t, results[0].Error, "Expected no error, got %v", results[0].Error)
	assert.Len(t, results[0].Children, 3)
	assert.NotEqual(t, path, string(results[0].Children[1].StdOut))

	for _, line := range requiredTree {
		assert.Contains(t, string(results[0].Children[2].StdOut), line)
	}
}
