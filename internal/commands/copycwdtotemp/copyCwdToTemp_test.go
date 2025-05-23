// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tmpDir = "/tmp"
	srcDir = "src"
)

// CwdTrackerCommand is a simplified mock command that only tracks its cwd.
type CwdTrackerCommand struct {
	label       string
	executedCwd string
}

func (c *CwdTrackerCommand) Run(_ context.Context) runbatch.Results {
	return runbatch.Results{
		{
			Label:    c.label,
			ExitCode: 0,
		},
	}
}

func (c *CwdTrackerCommand) SetCwd(cwd string) {
	c.executedCwd = cwd
}

func (c *CwdTrackerCommand) InheritEnv(_ map[string]string) {}

func TestCopyCwdToTemp(t *testing.T) {
	// Create a mock filesystem
	mockFS := fstest.MapFS{
		srcDir + "/file1.txt":         &fstest.MapFile{Data: []byte("content of file 1")},
		srcDir + "/file2.txt":         &fstest.MapFile{Data: []byte("content of file 2")},
		srcDir + "/subdir/file3.txt":  &fstest.MapFile{Data: []byte("content in subdirectory")},
		srcDir + "/subdir/file4.txt":  &fstest.MapFile{Data: []byte("more content in subdirectory")},
		srcDir + "/subdir2/file5.txt": &fstest.MapFile{Data: []byte("content in another subdirectory")},
	}

	// Save original values to restore after test
	originalCwdFS := FS
	originalTempDirPath := TempDirPath
	originalRandomName := RandomName

	defer func() {
		// Restore original values
		FS = originalCwdFS
		TempDirPath = originalTempDirPath
		RandomName = originalRandomName
	}()

	// Create and populate our mock filesystem
	FS = afero.NewMemMapFs()

	// Set the current working directory for the test
	cwd := srcDir

	// Mock RandomName to return a known value
	RandomName = func(prefix string, n int) string {
		return prefix + "testrun"
	}

	// Mock TempDirPath to return a simple path
	TempDirPath = func() string {
		return tmpDir
	}

	// Add the files to our mock filesystem by copying from fstest.MapFS to afero.MemMapFs
	for path, mapFile := range mockFS {
		// Create directory if needed
		dir := filepath.Dir(path)
		if dir != "." {
			err := FS.MkdirAll(dir, 0755)
			require.NoError(t, err, "Failed to create directory: %s", dir)
		}

		// Create and write to the file
		err := afero.WriteFile(FS, path, mapFile.Data, 0644)
		require.NoError(t, err, "Failed to write file: %s", path)
	}

	// Run the command
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// We need to capture what temp directory was created
	var capturedTempDir string

	f := New(cwd)
	results := f.Run(ctx)

	// The temp dir should now be
	capturedTempDir = filepath.Join("/tmp", "avmtool_testrun")

	// Check the results
	require.Len(t, results, 1)
	assert.Equal(t, 0, results[0].ExitCode)
	require.NoError(t, results[0].Error)

	// Verify that we have a temp dir
	assert.NotEmpty(t, capturedTempDir)

	// Verify each file was copied correctly
	for path, mapFile := range mockFS {
		// Extract the filename relative to the cwd
		var relativePath string
		if filepath.Dir(path) == cwd {
			relativePath = filepath.Base(path)
		} else {
			relativePath = strings.TrimPrefix(path, cwd+afero.FilePathSeparator)
		}

		copiedFilePath := filepath.Join(capturedTempDir, relativePath)
		// Use the mock filesystem to read files instead of real OS
		content, err := afero.ReadFile(FS, copiedFilePath)
		require.NoError(t, err, "should be able to read %s", relativePath)
		assert.Equal(t, string(mapFile.Data), string(content), "file content should match for %s", relativePath)
	}

	// Clean up
	FS.RemoveAll(capturedTempDir) //nolint:errcheck
}

func TestCopyCwdToTemp_ErrorHandling(t *testing.T) {
	// Save original values to restore after test
	originalCwdFS := FS
	originalTempDirPath := TempDirPath
	originalRandomName := RandomName

	defer func() {
		// Restore original values
		FS = originalCwdFS
		TempDirPath = originalTempDirPath
		RandomName = originalRandomName
	}()

	// Mock RandomName to return a known value
	RandomName = func(prefix string, n int) string {
		return prefix + "testrun"
	}

	FS = afero.NewMemMapFs()

	// Test case: MkdirTemp error
	t.Run("MkdirTemp error", func(t *testing.T) {
		// Set the current working directory for the test
		cwd := srcDir

		// Mock TempDirPath to return a simple path
		TempDirPath = func() string {
			return tmpDir
		}

		baseFs := afero.NewMemMapFs()
		errPath := filepath.Join(TempDirPath(), "avmtool_testrun")
		FS = &errorFS{fs: baseFs, errorPath: errPath} // Create an errorFS that will return an error for directory

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		f := New(cwd)
		results := f.Run(ctx)

		require.Len(t, results, 1)
		assert.Equal(t, -1, results[0].ExitCode) // FunctionCommand.Run returns -1 for errors
		assert.ErrorIs(t, results[0].Error, os.ErrPermission)
	})

	// Test case: WalkDir error
	t.Run("WalkDir error", func(t *testing.T) {
		// Create a base filesystem with a file
		baseFs := afero.NewMemMapFs()

		// Set the current working directory for the test
		cwd := srcDir

		// Create the directory and file structure
		err := baseFs.MkdirAll(cwd, 0755)
		require.NoError(t, err)

		testFilePath := filepath.Join(cwd, "file1.txt")
		_ = afero.WriteFile(baseFs, testFilePath, []byte("content"), 0644)

		// Mock TempDirPath to return a simple path
		TempDirPath = func() string {
			return tmpDir
		}

		FS = &errorFS{fs: baseFs, errorPath: testFilePath} // Create an errorFS that will return an error for file1.txt

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		f := New(cwd)
		results := f.Run(ctx)

		require.Len(t, results, 1)
		assert.Equal(t, -1, results[0].ExitCode) // FunctionCommand.Run returns -1 for errors
		assert.Error(t, results[0].Error)
	})

	t.Run("context canceled", func(t *testing.T) {
		// Set the current working directory for the test
		cwd := srcDir

		// Mock TempDirPath to return a simple path
		TempDirPath = func() string {
			return tmpDir
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cancel()

		f := New(cwd)
		results := f.Run(ctx)

		require.Len(t, results, 1)
		assert.Equal(t, -1, results[0].ExitCode) // FunctionCommand.Run returns -1 for errors
		assert.Equal(t, ctx.Err(), results[0].Error)
	})
}

// the behavior.
func TestCwdChangePropagation(t *testing.T) {
	const newCwd = tmpDir + "	/new_cwd_path"

	cwdChangingCmd := &runbatch.FunctionCommand{
		Label: "Change CWD",
		Func: func(_ context.Context, _ string, _ ...string) runbatch.FunctionCommandReturn {
			return runbatch.FunctionCommandReturn{
				NewCwd: newCwd,
			}
		},
	}

	// Command that tracks its CWD
	tracker := &CwdTrackerCommand{
		label: "Subsequent command",
	}

	// Create the batch
	batch := &runbatch.SerialBatch{
		Label:    "Test CWD change batch",
		Commands: []runbatch.Runnable{cwdChangingCmd, tracker},
	}

	// Run the batch
	batch.Run(context.Background())

	// Verify the tracker received the new CWD
	assert.Equal(t, newCwd, tracker.executedCwd,
		"The subsequent command should have received the new CWD")
}

// TestCopyCwdTempIntegration does an integration test with the actual CopyCwdToTemp command.
func TestCopyCwdTempIntegration(t *testing.T) {
	// Save original values to restore after test
	originalCwdFS := FS
	originalTempDirPath := TempDirPath
	originalRandomName := RandomName

	defer func() {
		// Restore original values
		FS = originalCwdFS
		TempDirPath = originalTempDirPath
		RandomName = originalRandomName
	}()

	// Create and configure mock filesystem
	memFs := afero.NewMemMapFs()
	FS = memFs

	// Define constants for test paths
	const (
		initialCwd     = "/test/initial/path"
		tempDir        = tmpDir
		randomSuffix   = "testrun"
		expectedNewCwd = tempDir + "/avmtool_testrun"
	)

	// Setup temp directory
	TempDirPath = func() string {
		return tempDir
	}

	// Setup deterministic random name for testing
	RandomName = func(prefix string, n int) string {
		return prefix + randomSuffix
	}

	// Create the initial directory structure
	err := memFs.MkdirAll(initialCwd, 0755)
	require.NoError(t, err)
	err = afero.WriteFile(memFs, filepath.Join(initialCwd, "testfile.txt"), []byte("test content"), 0644)
	require.NoError(t, err)

	// Create our test commands
	copyCwdCmd := New(initialCwd)
	trackerCmd := &CwdTrackerCommand{
		label: "Command after copy",
	}

	// Create and run a serial batch with both commands
	batch := &runbatch.SerialBatch{
		Label:    "Copy and track batch",
		Commands: []runbatch.Runnable{copyCwdCmd, trackerCmd},
	}

	batch.Run(context.Background())

	// Verify the tracker picked up the new working directory
	assert.Equal(t, expectedNewCwd, trackerCmd.executedCwd,
		"The CopyCwdToTemp command should have set the working directory for subsequent commands")

	// Verify the file was copied to the new directory
	content, err := afero.ReadFile(memFs, filepath.Join(expectedNewCwd, "testfile.txt"))
	require.NoError(t, err, "The file should have been copied to the new directory")
	assert.Equal(t, "test content", string(content), "The file content should match")
}
