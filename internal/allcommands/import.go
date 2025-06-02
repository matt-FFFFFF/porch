// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package allcommands imports all command packages to ensure their registration.
package allcommands

import (
	// Import all command packages to trigger their init() functions.
	_ "github.com/matt-FFFFFF/porch/internal/commands/copycwdtotemp"
	_ "github.com/matt-FFFFFF/porch/internal/commands/parallelcommand"
	_ "github.com/matt-FFFFFF/porch/internal/commands/serialcommand"
	_ "github.com/matt-FFFFFF/porch/internal/commands/shellcommand"
)
