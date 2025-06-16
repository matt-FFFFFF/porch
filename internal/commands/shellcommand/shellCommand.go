// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package shellcommand provides a way to create an OSCommand that searches for a command in the system PATH.
package shellcommand

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

const (
	// GOOSWindows is the string constant for Windows OS from the runtime package.
	GOOSWindows          = "windows"    // GOOSWindows is the string constant for Windows OS from the runtime package.
	commandSwitchWindows = "/C"         // Command switch for Windows cmd.exe
	commandSwitchUnix    = "-c"         // Command switch for Unix-like shells
	winSystem32          = "System32"   // System32 is the directory where cmd.exe is located on Windows.
	cmdExe               = "cmd.exe"    // cmdExe is the name of the command interpreter executable on Windows.
	binSh                = "/bin/sh"    // Default shell for Unix-like systems.
	winSystemRootEnv     = "SystemRoot" // Environment variable for Windows system root directory.
)

var (
	// ErrCommandNotFound is returned when the command is not found in the system PATH or if the command is empty.
	ErrCommandNotFound = errors.New("command not found")
)

// New creates a new runbatch.OSCommand. It will search for the command in the system PATH.
// It returns nil if the command is not found or if the command is empty.
// On Windows, there is no need to add .exe to the command name.
func New(
	ctx context.Context,
	base *runbatch.BaseCommand,
	command string,
	successExitCodes []int,
	skipExitCodes []int) (*runbatch.OSCommand, error,
) {
	if command == "" {
		return nil, ErrCommandNotFound
	}

	var osCommandArgs []string

	switch runtime.GOOS {
	case GOOSWindows:
		osCommandArgs = []string{commandSwitchWindows, command}
	default:
		osCommandArgs = []string{commandSwitchUnix, command}
	}

	return &runbatch.OSCommand{
		BaseCommand:      base,
		Path:             defaultShell(ctx),
		Args:             osCommandArgs,
		SuccessExitCodes: successExitCodes,
		SkipExitCodes:    skipExitCodes,
	}, nil
}

func defaultShell(ctx context.Context) string {
	if runtime.GOOS == GOOSWindows {
		systemRoot := os.Getenv(winSystemRootEnv)
		if systemRoot == "" {
			systemRoot = `C:\Windows`
		}

		return fmt.Sprintf(`%s\%s\%s`, systemRoot, winSystem32, cmdExe)
	}

	if shell := os.Getenv("SHELL"); shell != "" {
		ctxlog.Debug(ctx, "Using SHELL environment variable", "shell", shell)
		return shell
	}

	return binSh
}
