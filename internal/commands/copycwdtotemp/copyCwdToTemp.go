package copycwdtotemp

import (
	"math/rand"
	"os"
	"path/filepath"

	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
	"github.com/spf13/afero"
)

var FS = afero.NewOsFs()

// TempDirPath returns the temporary directory to use
var TempDirPath = os.TempDir

// RandomName generates a random string with the given prefix and length
var RandomName = func(prefix string, n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return prefix + string(b)
}

func New(cwd string) *runbatch.FunctionCommand {
	ret := &runbatch.FunctionCommand{
		Label: "Copy current working directory to temporary directory",
		Func: func(cwd string) runbatch.FunctionCommandReturn {
			tmpDir := filepath.Join(TempDirPath(), RandomName("avmtool_", 8))
			// Create a temporary directory in the OS temp directory
			err := FS.MkdirAll(tmpDir, 0755)
			if err != nil {
				return runbatch.FunctionCommandReturn{
					Err: err,
				}
			}

			// Use afero.Walk to copy files from the current directory to the temp directory
			err = afero.Walk(FS, cwd, func(path string, info os.FileInfo, err error) error {
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
					return err
				}

				// Create the destination path
				dstPath := filepath.Clean(filepath.Join(tmpDir, relPath))

				// If it's a directory, create it
				if info.IsDir() {
					return FS.MkdirAll(dstPath, 0755)
				}

				// If it's a file, copy it
				srcFile, err := afero.ReadFile(FS, path)
				if err != nil {
					return err
				}

				return afero.WriteFile(FS, dstPath, srcFile, 0644)
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
	ret.SetCwd(cwd)
	return ret
}
