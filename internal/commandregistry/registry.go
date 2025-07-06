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
	"github.com/matt-FFFFFF/porch/internal/config/hcl"
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
	// ErrCircularDependency is returned when a circular dependency is detected.
	ErrCircularDependency = errors.New("circular dependency detected")
	// ErrMaxRecursionDepth is returned when maximum recursion depth is exceeded.
	ErrMaxRecursionDepth = errors.New("maximum recursion depth exceeded")
)

const (
	// MaxRecursionDepth is the maximum depth allowed for command group resolution.
	MaxRecursionDepth = 100
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
func (r *Registry) CreateRunnableFromYAML(
	ctx context.Context, yamlData []byte, parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("command creation cancelled: %w", ctx.Err())
	default:
	}

	var cmdType commandType
	if err := yaml.Unmarshal(yamlData, &cmdType); err != nil {
		return nil, errors.Join(ErrCommandUnmarshal, err)
	}

	commander, exists := r.commands[cmdType.Type]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCommandType, cmdType.Type)
	}

	runnable, err := commander.CreateFromYaml(ctx, r, yamlData, parent)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCommandCreation, cmdType.Type, err)
	}

	return runnable, nil
}

// CreateRunnableFromHCL creates a runnable from HCL data using this registry.
func (r *Registry) CreateRunnableFromHCL(
	ctx context.Context, hclCommand *hcl.CommandBlock, parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("command creation cancelled: %w", ctx.Err())
	default:
	}

	commander, exists := r.commands[hclCommand.Type]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCommandType, hclCommand.Type)
	}

	runnable, err := commander.CreateFromHcl(ctx, r, hclCommand, parent)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCommandCreation, hclCommand.Type, err)
	}

	return runnable, nil
}

// Iter returns an iterator over all registered command types and their commanders.
func (r *Registry) Iter() iter.Seq2[string, commands.Commander] {
	return maps.All(r.commands)
}

// ResolveCommandGroup resolves a command group by name to a list of command definitions.
func (r *Registry) ResolveCommandGroup(groupName string) ([]any, error) {
	return r.resolveCommandGroupWithDepth(groupName, make(map[string]bool), []string{}, 0)
}

// resolveCommandGroupWithDepth resolves a command group with circular dependency detection.
func (r *Registry) resolveCommandGroupWithDepth(
	groupName string,
	visiting map[string]bool,
	path []string,
	depth int,
) ([]any, error) {
	// Check maximum recursion depth
	if depth > MaxRecursionDepth {
		return nil, fmt.Errorf("%w: exceeded depth of %d while resolving command groups",
			ErrMaxRecursionDepth, MaxRecursionDepth)
	}

	// Check if we're currently visiting this group (circular dependency)
	if visiting[groupName] {
		cyclePath := append(path, groupName)
		return nil, fmt.Errorf("%w: %s", ErrCircularDependency,
			formatCircularDependencyPath(cyclePath))
	}

	// Check if the group exists
	commands, exists := r.commandGroups[groupName]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCommandGroup, groupName)
	}

	// Mark this group as being visited
	visiting[groupName] = true

	currentPath := append(path, groupName)

	// Check all commands in this group for nested command group references
	for i, cmd := range commands {
		if err := r.validateCommandForCircularDeps(cmd, visiting, currentPath, depth+1, i); err != nil {
			return nil, err
		}
	}

	// Unmark this group as being visited (backtrack)
	delete(visiting, groupName)

	return commands, nil
}

// validateCommandForCircularDeps validates a command for circular dependencies.
func (r *Registry) validateCommandForCircularDeps(
	cmd any,
	visiting map[string]bool,
	path []string,
	depth int,
	cmdIndex int,
) error {
	// Convert command to map to check for command_group field
	cmdMap, ok := cmd.(map[string]any)
	if !ok {
		return nil // Not a map, skip validation
	}

	// Check if this command references a command group
	if groupName, exists := cmdMap["command_group"]; exists {
		if groupNameStr, ok := groupName.(string); ok && groupNameStr != "" {
			// Recursively validate the referenced command group
			_, err := r.resolveCommandGroupWithDepth(groupNameStr, visiting, path, depth)
			if err != nil {
				return fmt.Errorf("in command %d of group %s: %w",
					cmdIndex, path[len(path)-1], err)
			}
		}
	}

	return nil
}

// formatCircularDependencyPath formats a circular dependency path for error messages.
func formatCircularDependencyPath(path []string) string {
	if len(path) == 0 {
		return "unknown path"
	}

	result := path[0]
	for i := 1; i < len(path); i++ {
		result += " → " + path[i]
	}

	// Show the cycle by adding the first element again
	result += " → " + path[0]

	return result
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
