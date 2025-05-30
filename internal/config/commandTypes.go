// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config

// CommandDefinition represents the base structure that all commands must implement.
type CommandDefinition interface {
	GetType() string
}
