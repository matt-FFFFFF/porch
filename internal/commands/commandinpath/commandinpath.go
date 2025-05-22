// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// package commandinpath provides a way to create an OSCommand that searches for a command in the system PATH.
package commandinpath

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
)

const (
	GOOSWindows = "windows"
)

func New(label, command, cwd string, args []string) *runbatch.OSCommand {
	if command == "" {
		return nil
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
			}
		}
	}

	return nil
}
