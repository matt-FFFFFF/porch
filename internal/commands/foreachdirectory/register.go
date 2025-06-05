// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import "github.com/matt-FFFFFF/porch/internal/commandregistry"

const commandType = "foreachdirectory"

// Register registers the command in the given registry.
func Register(r commandregistry.Registry) {
	r.Register(commandType, &Commander{})
}
