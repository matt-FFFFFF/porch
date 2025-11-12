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

	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

const (
	tmpDir = "/tmp"
	srcDir = "src"
)

// TestMain is used to run the goleak verification before and after tests.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// cwdTrackerCommand is a simplified mock command that only tracks its cwd.
type cwdTrackerCommand struct {
	*runbatch.BaseCommand
}

func (c *cwdTrackerCommand) Run(_ context.Context) runbatch.Results {
	return runbatch.Results{&runbatch.Result{
		Label:    c.Label,
		ExitCode: 0, // Simulate success
		Error:    nil,
		Status:   runbatch.ResultStatusSuccess,
	}}
}

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

	base := &runbatch.BaseCommand{
		Label: "copyCwdToTemp",
		Cwd:   cwd,
	}
	f := New(base)
	results := f.Run(ctx)

	// Wait for command to complete before restoring globals
	require.Len(t, results, 1)
	assert.Equal(t, 0, results[0].ExitCode)
	require.NoError(t, results[0].Error)

	// The temp dir should now be
	capturedTempDir = filepath.Join("/tmp", "porch_testrun")

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
	tests := []struct {
		name     string
		setupFS  func() afero.Fs
		testFunc func(t *testing.T, fs afero.Fs)
	}{
		{
			name: "MkdirTemp error",
			setupFS: func() afero.Fs {
				baseFs := afero.NewMemMapFs()
				errPath := filepath.Join(tmpDir, "porch_testrun")

				return &errorFS{fs: baseFs, errorPath: errPath}
			},
			testFunc: func(t *testing.T, fs afero.Fs) {
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

				// Setup test environment
				FS = fs
				TempDirPath = func() string { return tmpDir }
				RandomName = func(prefix string, n int) string { return prefix + "testrun" }

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				base := &runbatch.BaseCommand{
					Label: "copyCwdToTemp",
					Cwd:   srcDir,
				}
				f := New(base)
				results := f.Run(ctx)

				// Wait for the command to complete before restoring globals
				require.Len(t, results, 1)
				assert.Equal(t, -1, results[0].ExitCode) // FunctionCommand.Run returns -1 for errors
				require.ErrorIs(t, results[0].Error, os.ErrPermission)
			},
		},
		{
			name: "WalkDir error",
			setupFS: func() afero.Fs {
				baseFs := afero.NewMemMapFs()
				// Create the directory and file structure
				err := baseFs.MkdirAll(srcDir, 0755)
				if err != nil {
					panic(err)
				}

				testFilePath := filepath.Join(srcDir, "file1.txt")
				_ = afero.WriteFile(baseFs, testFilePath, []byte("content"), 0644)

				return &errorFS{fs: baseFs, errorPath: testFilePath}
			},
			testFunc: func(t *testing.T, fs afero.Fs) {
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

				// Setup test environment
				FS = fs
				TempDirPath = func() string { return tmpDir }
				RandomName = func(prefix string, n int) string { return prefix + "testrun" }

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				base := &runbatch.BaseCommand{
					Label: "copyCwdToTemp",
					Cwd:   srcDir,
				}
				f := New(base)
				results := f.Run(ctx)

				// Wait for the command to complete before restoring globals
				require.Len(t, results, 1)
				assert.Equal(t, -1, results[0].ExitCode) // FunctionCommand.Run returns -1 for errors
				require.Error(t, results[0].Error)
			},
		},
		{
			name: "context canceled",
			setupFS: func() afero.Fs {
				return afero.NewMemMapFs()
			},
			testFunc: func(t *testing.T, fs afero.Fs) {
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

				// Setup test environment
				FS = fs
				TempDirPath = func() string { return tmpDir }
				RandomName = func(prefix string, n int) string { return prefix + "testrun" }

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				cancel() // Cancel immediately

				base := &runbatch.BaseCommand{
					Label: "copyCwdToTemp",
					Cwd:   srcDir,
				}
				f := New(base)
				results := f.Run(ctx)

				// Wait for the command to complete before restoring globals
				require.Len(t, results, 1)
				assert.Equal(t, -1, results[0].ExitCode) // FunctionCommand.Run returns -1 for errors
				assert.Equal(t, ctx.Err(), results[0].Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := tt.setupFS()
			tt.testFunc(t, fs)
		})
	}
}

// the behavior.
func TestCwdChangePropagation(t *testing.T) {
	const newCwd = tmpDir + "	/new_cwd_path"

	cwdChangingCmd := &runbatch.FunctionCommand{
		BaseCommand: &runbatch.BaseCommand{
			Label: "Change CWD",
			Cwd:   tmpDir,
		},
		Func: func(_ context.Context, _ string, _ ...string) runbatch.FunctionCommandReturn {
			return runbatch.FunctionCommandReturn{
				NewCwd: newCwd,
			}
		},
	}

	// Command that tracks its CWD
	tracker := &cwdTrackerCommand{
		BaseCommand: &runbatch.BaseCommand{
			Label: "Subsequent command",
			Cwd:   tmpDir, // Initial CWD
		},
	}

	// Create the batch
	batch := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: "CWD Change Test Batch",
		},
		Commands: []runbatch.Runnable{cwdChangingCmd, tracker},
	}

	// Set parent for proper context
	for _, cmd := range batch.Commands {
		cmd.SetParent(batch)
	}

	// Run the batch
	batch.Run(context.Background())

	// Verify the tracker received the new CWD
	assert.Equal(t, newCwd, tracker.Cwd,
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
		expectedNewCwd = tempDir + "/porch_testrun"
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
	base := &runbatch.BaseCommand{
		Label: "CopyCwdToTemp",
		Cwd:   initialCwd,
	}
	copyCwdCmd := New(base)
	trackerCmd := &cwdTrackerCommand{
		BaseCommand: &runbatch.BaseCommand{
			Label: "Tracker Command",
			Cwd:   initialCwd, // Start with the initial CWD
		},
	}

	// Create and run a serial batch with both commands
	batch := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: "Test CopyCwdToTemp Batch",
			Cwd:   initialCwd, // Set the initial CWD for the batch
		},
		Commands: []runbatch.Runnable{copyCwdCmd, trackerCmd},
	}
	// Set parent for proper context
	for _, cmd := range batch.Commands {
		cmd.SetParent(batch)
	}

	// Run the batch and wait for completion
	results := batch.Run(context.Background())

	// Ensure batch completed before proceeding
	require.NotNil(t, results)

	// Verify the tracker picked up the new working directory
	assert.Equal(t, expectedNewCwd, trackerCmd.Cwd,
		"The CopyCwdToTemp command should have set the working directory for subsequent commands")

	// Verify the file was copied to the new directory
	content, err := afero.ReadFile(memFs, filepath.Join(expectedNewCwd, "testfile.txt"))
	require.NoError(t, err, "The file should have been copied to the new directory")
	assert.Equal(t, "test content", string(content), "The file content should match")
}
