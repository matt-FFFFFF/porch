// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package parallelcommand

import "github.com/matt-FFFFFF/porch/internal/commands"

// Definition represents the YAML configuration for the parallel command.
type Definition struct {
	commands.BaseDefinition `yaml:",inline"`
	Commands                []any `yaml:"commands" docdesc:"List of commands to execute in parallel"`
}
