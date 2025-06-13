// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/spf13/afero"
)

// FS is a filesystem abstraction used for file operations.
// Default is the OS filesystem, but can be replaced with a mock for testing.
var FS = afero.NewOsFs()

var (
	// ErrFileCopy is returned when a file copy operation fails.
	ErrFileCopy = errors.New("file copy error")
	// ErrFilePath is returned when a file path operation fails.
	ErrFilePath = errors.New("file path error")
)

const (
	// sevenFiveFive is the file mode for directories created in the temporary directory.
	sevenFiveFive = 0o755
	// tempDirSuffixLength is the length of the random suffix for the temporary directory.
	tempDirSuffixLength = 8
)

// TempDirPath returns the temporary directory to use.
var TempDirPath = os.TempDir

// RandomName generates a random string with the given prefix and length.
var RandomName = func(prefix string, n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return prefix + string(b)
}

// New creates a new command that copies the current working directory to a temporary directory.
// It also sets the new working directory to the temporary directory for any subsequent
// serial batch commands.
func New(base *runbatch.BaseCommand) *runbatch.FunctionCommand {
	// Capture the global variables by value to avoid data races with tests
	fs := FS
	tempDirPath := TempDirPath
	randomName := RandomName

	ret := &runbatch.FunctionCommand{
		BaseCommand: base,
		Func: func(ctx context.Context, cwd string, _ ...string) runbatch.FunctionCommandReturn {
			tmpDir := filepath.Join(tempDirPath(), randomName("porch_", tempDirSuffixLength))
			// Create a temporary directory in the OS temp directory
			err := fs.MkdirAll(tmpDir, sevenFiveFive)
			if err != nil {
				return runbatch.FunctionCommandReturn{
					Err: err,
				}
			}

			// Use afero.Walk to copy files from the current directory to the temp directory
			err = afero.Walk(fs, cwd, func(path string, info os.FileInfo, err error) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					if err != nil {
						return err
					}

					// Skip the temporary directory itself to avoid infinite recursion
					if path == cwd {
						return nil
					}

					// Strip cwd from the path to get the relative path
					relPath, err := filepath.Rel(cwd, path)
					if err != nil {
						return errors.Join(ErrFilePath, err)
					}

					// Create the destination path
					dstPath := filepath.Clean(filepath.Join(tmpDir, relPath))

					// If it's a directory, create it
					if info.IsDir() {
						return fs.MkdirAll(dstPath, sevenFiveFive)
					}

					// If it's a file, copy it
					srcFile, err := afero.ReadFile(fs, path)
					if err != nil {
						return errors.Join(ErrFileCopy, err)
					}

					return afero.WriteFile(fs, dstPath, srcFile, info.Mode().Perm())
				}
			})

			if err != nil {
				return runbatch.FunctionCommandReturn{
					Err: err,
				}
			}

			// Return the newly created temp directory as the new working directory
			return runbatch.FunctionCommandReturn{
				NewCwd: tmpDir,
			}
		},
	}

	return ret
}
