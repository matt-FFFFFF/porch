// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package commands provides a standard interface for creating a command registry.
package commands

import "github.com/matt-FFFFFF/avmtool/internal/runbatch"

// Commander is an interface for creating commands.
type Commander interface {
	Create(name, exec, cwd string, args ...string) (runbatch.Runnable, error)
}
