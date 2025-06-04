// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package parallelcommand

import "github.com/matt-FFFFFF/porch/internal/commandregistry"

const commandType = "parallel"

// init registers the parallel command type.
func init() {
	commandregistry.Register(commandType, NewCommander())
}
