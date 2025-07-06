// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package pwshcommand

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

const (
	// goOSWindows is the constant for Windows operating system.
	goOSWindows         = "windows"
	pwshExecName        = "pwsh"
	osCommandArgsLength = 6 // Number of arguments for the PowerShell command
)

var (
	// 	ErrCannotFindPwsh is returned when the pwsh executable cannot be found in the system PATH.
	ErrCannotFindPwsh = errors.New("cannot find pwsh executable in PATH")
	// ErrBothScriptAndScriptFileSpecified is returned when both Script and
	// ScriptFile are specified in the command definition.
	ErrBothScriptAndScriptFileSpecified = errors.New("cannot specify both script and scriptFile in the same command")
	// ErrCannotCreateTempFile is returned when a temporary file cannot be created for the script.
	ErrCannotCreateTempFile = errors.New("cannot create temporary file for script")
	// ErrCannotWriteTempFile is returned when the script cannot be written to the temporary file.
	ErrCannotWriteTempFile = errors.New("cannot write script to temporary file")
)

// New creates a new runbatch.OSCommand for PowerShell scripts.
func New(_ context.Context,
	base *runbatch.BaseCommand,
	script, scriptFile string,
	successExitCodes, skipExitCodes []int,
) (runbatch.Runnable, error) {
	// Check for conflicting script definitions first
	if script != "" && scriptFile != "" {
		return nil, ErrBothScriptAndScriptFileSpecified
	}

	execName := pwshExecName
	if runtime.GOOS == goOSWindows {
		execName = pwshExecName + ".exe" // On Windows, pwsh is typically pwsh.exe
	}

	execPath, err := exec.LookPath(execName)
	if err != nil && !errors.Is(err, exec.ErrDot) {
		return nil, errors.Join(ErrCannotFindPwsh, err)
	}

	cmd := &runbatch.OSCommand{
		BaseCommand:      base,
		Path:             execPath,
		Args:             nil, // Arguments will be set below
		SuccessExitCodes: successExitCodes,
		SkipExitCodes:    skipExitCodes,
	}

	// If script is specified, write it to a temporary file and use that as the script file.
	if script != "" {
		tmpFile, err := os.CreateTemp("", "script-*.ps1")

		if err != nil {
			return nil, errors.Join(ErrCannotCreateTempFile, err)
		}

		defer tmpFile.Close() //nolint:errcheck

		if _, err := tmpFile.Write([]byte(script)); err != nil {
			return nil, errors.Join(ErrCannotWriteTempFile, err)
		}

		scriptFile = tmpFile.Name()

		// Set the cleanup function to remove the temporary script file after execution
		cmd.SetCleanup(func(ctx context.Context) {
			ctxlog.Logger(ctx).Debug("cleaning up temporary script file", "file", scriptFile)
			os.Remove(scriptFile) //nolint:errcheck
		})
	}

	osCommmandArgs := make([]string, osCommandArgsLength)

	osCommmandArgs[0] = "-NonInteractive"
	osCommmandArgs[1] = "-NoProfile"
	osCommmandArgs[2] = "-ExecutionPolicy"
	osCommmandArgs[3] = "Bypass" // Bypass execution policy for the script, Windows only
	osCommmandArgs[4] = "-File"
	osCommmandArgs[5] = scriptFile

	cmd.Args = osCommmandArgs

	return cmd, nil
}
