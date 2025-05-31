// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package serialcommand

import "github.com/matt-FFFFFF/pporch/internal/commandregistry"

const CommandType = "serial"

// init registers the serial command type.
func init() {
	commandregistry.Register(CommandType, &Commander{})
}
