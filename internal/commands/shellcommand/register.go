// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package shellcommand

import "github.com/matt-FFFFFF/porch/internal/commandregistry"

const commandType = "shell"

// Register registers the command in the given registry.
func Register(r commandregistry.Registry) {
	r.Register(commandType, &Commander{})
}
