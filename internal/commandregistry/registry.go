// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commandregistry

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

var (
	// ErrUnknownCommandType is returned when a command type is not registered.
	ErrUnknownCommandType = errors.New("unknown command type")
	// ErrCommandCreation is returned when a command cannot be created.
	ErrCommandCreation = errors.New("failed to create command")
	// ErrCommandUnmarshal is returned when a command cannot be unmarshaled.
	ErrCommandUnmarshal = errors.New("failed to unmarshal command definition")
)

// Registry holds the mapping between command types and their commanders.
type Registry struct {
	m map[string]commands.Commander
}

// RegistrationFunc is a function that registers commands in a registry.
type RegistrationFunc func(Registry)

// New creates a new registry with the given registration functions.
func New(registrations ...RegistrationFunc) *Registry {
	registry := Registry{
		m: make(map[string]commands.Commander),
	}
	for _, register := range registrations {
		register(registry)
	}

	return &registry
}

// Register adds a command type and its commander to the registry.
func (r *Registry) Register(commandType string, commander commands.Commander) error {
	r.m[commandType] = commander
	return nil
}

// Get retrieves a commander for the given command type.
func (r *Registry) Get(commandType string) (commands.Commander, bool) {
	commander, exists := r.m[commandType]
	return commander, exists
}

// CreateRunnableFromYAML creates a runnable from YAML data using this registry.
func (r *Registry) CreateRunnableFromYAML(ctx context.Context, yamlData []byte) (runbatch.Runnable, error) {
	var cmdType commandType
	if err := yaml.Unmarshal(yamlData, &cmdType); err != nil {
		return nil, errors.Join(ErrCommandUnmarshal, err)
	}

	commander, exists := r.m[cmdType.Type]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCommandType, cmdType.Type)
	}

	runnable, err := commander.Create(ctx, r, yamlData)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCommandCreation, cmdType.Type, err)
	}

	return runnable, nil
}

// Iter returns an iterator over all registered command types and their commanders.
func (r *Registry) Iter() iter.Seq2[string, commands.Commander] {
	return maps.All(r.m)
}

// commandType represents a command with its type and raw YAML data.
type commandType struct {
	Type string `yaml:"type"`
	Data any    `yaml:",inline"`
}
