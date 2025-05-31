// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commands

// BaseDefinition contains fields common to all command types.
type BaseDefinition struct {
	Type             string `yaml:"type"`
	Name             string `yaml:"name"`
	WorkingDirectory string `yaml:"working_directory,omitempty"`
}
