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
	// ErrUnknownCommandGroup is returned when a command group is not found.
	ErrUnknownCommandGroup = errors.New("unknown command group")
)

// Registry holds the mapping between command types and their commanders.
type Registry struct {
	commands      map[string]commands.Commander
	commandGroups map[string][]any
}

// RegistrationFunc is a function that registers commands in a registry.
type RegistrationFunc func(Registry)

// New creates a new registry with the given registration functions.
func New(registrations ...RegistrationFunc) *Registry {
	registry := Registry{
		commands:      make(map[string]commands.Commander),
		commandGroups: make(map[string][]any),
	}
	for _, register := range registrations {
		register(registry)
	}

	return &registry
}

// Register adds a command type and its commander to the registry.
// It should never fail but the error return is kept for compatibility with the CommanderFactory interface.
func (r *Registry) Register(commandType string, commander commands.Commander) error {
	r.commands[commandType] = commander
	return nil
}

// Get retrieves a commander for the given command type.
func (r *Registry) Get(commandType string) (commands.Commander, bool) {
	commander, exists := r.commands[commandType]
	return commander, exists
}

// CreateRunnableFromYAML creates a runnable from YAML data using this registry.
func (r *Registry) CreateRunnableFromYAML(ctx context.Context, yamlData []byte) (runbatch.Runnable, error) {
	var cmdType commandType
	if err := yaml.Unmarshal(yamlData, &cmdType); err != nil {
		return nil, errors.Join(ErrCommandUnmarshal, err)
	}

	commander, exists := r.commands[cmdType.Type]
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
	return maps.All(r.commands)
}

// ResolveCommandGroup resolves a command group by name to a list of command definitions.
func (r *Registry) ResolveCommandGroup(groupName string) ([]any, error) {
	commands, exists := r.commandGroups[groupName]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCommandGroup, groupName)
	}

	return commands, nil
}

// AddCommandGroup adds a command group to this registry.
func (r *Registry) AddCommandGroup(name string, data []any) {
	r.commandGroups[name] = data
}

// commandType represents a command with its type and raw YAML data.
type commandType struct {
	Type string `yaml:"type"`
	Data any    `yaml:",inline"`
}
