// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package cmdstate provides a context key for the command factory.
// We do this because we dynamically generate subcommands to display the examples.
// We have to use init for this, therefore global state is the only option.
package cmdstate

import "github.com/matt-FFFFFF/porch/internal/commands"

var Factory commands.CommanderFactory
