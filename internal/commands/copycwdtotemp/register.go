// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import "github.com/matt-FFFFFF/porch/internal/commandregistry"

const commandType = "copycwdtotemp"

// init registers the copycwdtotemp command type.
func init() {
	commandregistry.Register(commandType, &Commander{})
}
