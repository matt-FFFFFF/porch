// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package serialcommand

import "github.com/matt-FFFFFF/porch/internal/commandregistry"

const commandType = "serial"

// init registers the serial command type.
func init() {
	commandregistry.Register(commandType, &Commander{})
}
