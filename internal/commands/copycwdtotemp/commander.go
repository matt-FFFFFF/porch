// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import (
	"github.com/matt-FFFFFF/avmtool/internal/commands"
	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
)

var _ commands.Commander = (*Commander)(nil)

// Commander is a struct that implements the commands.Commander interface.
type Commander struct{}

// Create creates a new runnable command and implements the commands.Commander interface.
func (c *Commander) Create(_, _, cwd string, _ ...string) (runbatch.Runnable, error) {
	return New(cwd), nil
}
