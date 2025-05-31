// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package shellcommand

import "github.com/matt-FFFFFF/pporch/internal/commandregistry"

const commandType = "shell"

// init registers the shell command type.
func init() {
	commandregistry.Register(commandType, &Commander{})
}
