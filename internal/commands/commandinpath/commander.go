// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commandinpath

import (
	"github.com/matt-FFFFFF/avmtool/internal/commands"
	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
)

var _ commands.Commander = (*Commander)(nil)

// Type commander is a struct that implements the commands.Commander interface.
type Commander struct{}

// Create creates a new runnable command and implements the commands.Commander interface.
func (c *Commander) Create(name, exec, cwd string, args ...string) (runbatch.Runnable, error) {
	return New(name, exec, cwd, args)
}
