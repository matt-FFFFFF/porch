// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import "github.com/matt-FFFFFF/porch/internal/commandregistry"

const commandType = "foreachdirectory"

// init registers the foreachdirectory command type.
func init() {
	commandregistry.Register(commandType, NewCommander())
}
