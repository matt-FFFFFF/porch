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
	requiredTree := `.
./subdir
./subdir/test2.txt
./test.txt`
	cwd, _ := os.Getwd()
	path := path.Join(cwd, "testdata/copyCwdToTemp")
	base := &runbatch.BaseCommand{
		Label: "copyCwdToTemp",
		Cwd:   path,
	}
	copyCommand := New(base)
	pwdCommand := &runbatch.OSCommand{
		BaseCommand: &runbatch.BaseCommand{
			Label: "pwd",
			Cwd:   "",
		},
		Path: "/bin/sh",
		Args: []string{"-c", "pwd"},
	}
	checkFilesCommand := &runbatch.OSCommand{
		BaseCommand: &runbatch.BaseCommand{
			Label: "checkFiles",
			Cwd:   "",
		},
		Path: "/usr/bin/find",
		Args: []string{"."},
	}
	serialCommands := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: "test",
		},
		Commands: []runbatch.Runnable{
			copyCommand,
			pwdCommand,
			checkFilesCommand,
		},
	}
	results := serialCommands.Run(context.Background())
	assert.Len(t, results, 1)
	assert.Equal(t, 0, results[0].ExitCode)
	require.NoError(t, results[0].Error)
	assert.Len(t, results[0].Children, 3)
	assert.NotEqual(t, path, string(results[0].Children[1].StdOut))
	assert.Contains(t, string(results[0].Children[2].StdOut), requiredTree)
}
