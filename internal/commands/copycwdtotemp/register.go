// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import "github.com/matt-FFFFFF/pporch/internal/commandregistry"

const CommandType = "copycwdtotemp"

// init registers the copycwdtotemp command type.
func init() {
	commandregistry.Register(CommandType, &Commander{})
}
