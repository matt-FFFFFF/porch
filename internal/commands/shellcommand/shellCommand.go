// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package commandinpath provides a way to create an OSCommand that searches for a command in the system PATH.
package shellcommand

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/matt-FFFFFF/avmtool/internal/commands"
	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
)

const (
	// GOOSWindows is the string constant for Windows OS from the runtime package.
	GOOSWindows          = "windows"    // GOOSWindows is the string constant for Windows OS from the runtime package.
	commandSwitchWindows = "/C"         // Command switch for Windows cmd.exe
	commandSwitchUnix    = "-c"         // Command switch for Unix-like shells
	winSystem32          = "System32"   // System32 is the directory where cmd.exe is located on Windows.
	cmdExe               = "cmd.exe"    // cmdExe is the name of the command interpreter executable on Windows.
	defaut               = "/bin/sh"    // Default shell for Unix-like systems.
	winSystemRootEnv     = "SystemRoot" // Environment variable for Windows system root directory.
)

var (
	ErrCommandNotFound = errors.New("command not found")
)

type Definition struct {
	commands.BaseDefinition `yaml:",inline"`
	Exec                    string   `yaml:"exec"` // The command to execute, can be a path or a command name.
	Args                    []string `yaml:"args"` // Arguments to pass to the command.
}

// New creates a new runbatch.OSCommand. It will search for the command in the system PATH.
// It returns nil if the command is not found or if the command is empty.
// On Windows, there is no need to add .exe to the command name.
func New(ctx context.Context, label, command, cwd string, args []string) (*runbatch.OSCommand, error) {
	if command == "" {
		return nil, ErrCommandNotFound
	}
	cmdPath, err := resolveCommandPath(command)
	if err != nil {
		return nil, err //nolint:err113
	}
	osCommandArgs := make([]string, len(args)+2)

	if runtime.GOOS == GOOSWindows {
		osCommandArgs[0] = commandSwitchWindows
	} else {
		osCommandArgs[0] = commandSwitchUnix
	}

	osCommandArgs[1] = cmdPath
	copy(osCommandArgs[2:], args)

	return &runbatch.OSCommand{
		Label: label,
		Path:  defaultShell(ctx),
		Cwd:   cwd,
		Args:  osCommandArgs,
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
		return fmt.Sprintf("%s", shell)
	}
	return defaut
}

func resolveCommandPath(command string) (string, error) {
	if filepath.IsAbs(command) {
		return command, nil
	}
	// If the command is empty and we're on Windows, add .exe
	if runtime.GOOS == GOOSWindows && filepath.Ext(command) == "" {
		command += ".exe"
	}

	path := os.Getenv("PATH")
	paths := strings.Split(path, string(os.PathListSeparator))

	for _, p := range paths {
		fullPath := filepath.Join(p, command)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			if runtime.GOOS != GOOSWindows && info.Mode()&0111 == 0 {
				continue // Not executable
			}
			return fullPath, nil
		}
	}

	return "", ErrCommandNotFound
}
