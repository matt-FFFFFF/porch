// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package commandinpath provides a way to create an OSCommand that searches for a command in the system PATH.
package commandinpath

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
)

const (
	// GOOSWindows is the string constant for Windows OS from the runtime package.
	GOOSWindows = "windows"
)

var (
	ErrCommandNotFound = errors.New("command not found")
)

// New creates a new runbatch.OSCommand. It will search for the command in the system PATH.
// It returns nil if the command is not found or if the command is empty.
// On Windows, there is no need to add .exe to the command name.
func New(label, command, cwd string, args []string) (*runbatch.OSCommand, error) {
	if command == "" {
		return nil, ErrCommandNotFound
	}

	// If the command is empty and we're on Windows, add .exe
	if runtime.GOOS == GOOSWindows && filepath.Ext(command) == "" {
		command += ".exe"
	}

	path := os.Getenv("PATH")

	paths := strings.Split(path, string(os.PathListSeparator))
	for _, p := range paths {
		// check if the command exists in the path
		if info, err := os.Stat(filepath.Join(p, command)); err == nil {
			if info.IsDir() {
				continue
			}
			// check if the command is executable if not Windows
			if runtime.GOOS != GOOSWindows && info.Mode()&0111 == 0 {
				continue
			}

			return &runbatch.OSCommand{
				Label: label,
				Path:  filepath.Join(p, command),
				Cwd:   cwd,
				Args:  args,
			}, nil
		}
	}

	return nil, ErrCommandNotFound
}
