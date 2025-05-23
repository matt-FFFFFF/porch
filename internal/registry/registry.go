// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package registry

import (
	"github.com/matt-FFFFFF/avmtool/internal/commands"
	"github.com/matt-FFFFFF/avmtool/internal/commands/copycwdtotemp"
)

// Registry is a map of YAML command names to their respective Commander implementations.
type Registry map[string]commands.Commander

var DefaultRegistry = Registry{
	"copycwdtotemp": &copycwdtotemp.Commander{},
}
