// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package commands provides a standard interface for creating a command registry.
package commands

import (
	"context"
	"iter"

	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

// FactoryContextKey is the context key for the command factory.
type FactoryContextKey struct{}

// Commander is an interface for converting commands into runnables.
type Commander interface {
	// Create creates a runnable command from the provided payload.
	// The payload is the YAML command in bytes.
	Create(ctx context.Context, registry CommanderFactory, payload []byte, parent runbatch.Runnable) (runbatch.Runnable, error)
}

// CommanderFactory is an interface for creating a Commander.
type CommanderFactory interface {
	// Get retrieves a Commander by its command type.
	// It must have been registered in the command registry.
	Get(commandType string) (Commander, bool)
	// CreateRunnableFromYAML creates a runnable from the provided YAML payload.
	// This method is tasked with determining the command type from the payload.
	CreateRunnableFromYAML(ctx context.Context, payload []byte, parent runbatch.Runnable) (runbatch.Runnable, error)
	// Register registers a Commander for a specific command type.
	Register(cmdtype string, commander Commander) error
	// Iter returns an iterator over all registered command types.
	Iter() iter.Seq2[string, Commander]
	// ResolveCommandGroup resolves a command group by name to a list of command definitions.
	ResolveCommandGroup(groupName string) ([]any, error)
	// AddCommandGroup adds a command group to the factory.
	AddCommandGroup(name string, commands []any)
}
