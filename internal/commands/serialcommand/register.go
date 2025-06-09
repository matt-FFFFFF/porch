// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package serialcommand

import "github.com/matt-FFFFFF/porch/internal/commandregistry"

const commandType = "serial"

// Register registers the command in the given registry.
func Register(r commandregistry.Registry) {
	err := r.Register(commandType, &Commander{})
	if err != nil {
		panic(err)
	}
}
