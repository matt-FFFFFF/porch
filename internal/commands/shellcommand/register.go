// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package shellcommand

import "github.com/matt-FFFFFF/pporch/internal/commandregistry"

const CommandType = "shell"

// init registers the shell command type.
func init() {
	commandregistry.Register(CommandType, &Commander{})
}
