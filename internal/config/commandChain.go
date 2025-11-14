// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

var (
	// ErrInvalidYaml is returned when the YAML configuration is invalid.
	ErrInvalidYaml = errors.New("invalid YAML")
	// ErrNoCommands is returned when no commands are specified in the configuration.
	ErrNoCommands = errors.New("no commands specified")
	// ErrConfigurationTimeout is returned when configuration building times out.
	ErrConfigurationTimeout = errors.New("configuration building timed out")
	// ErrConfigBuild is returned when there is an error building the configuration.
	ErrConfigBuild = errors.New("error building configuration")
)

// Definition represents the root configuration structure.
type Definition struct {
	Name          string         `yaml:"name" json:"name" docdesc:"Name of the configuration"`                                 //nolint:lll
	Description   string         `yaml:"description" json:"description" docdesc:"Description of what this configuration does"` //nolint:lll
	Commands      []any          `yaml:"commands" json:"commands" docdesc:"List of commands to execute"`                       //nolint:lll
	CommandGroups []CommandGroup `yaml:"command_groups" json:"command_groups" docdesc:"List of command groups"`                //nolint:lll
}

// CommandGroup represents a named collection of commands that can be referenced by container commands.
type CommandGroup struct {
	Name        string `yaml:"name" json:"name" docdesc:"Name of the command group"`
	Description string `yaml:"description" json:"description" docdesc:"Description of the command group"`
	Commands    []any  `yaml:"commands" json:"commands" docdesc:"List of commands in this group"`
}

// BuildFromYAML creates a runnable from YAML configuration.
func BuildFromYAML(ctx context.Context, factory commands.CommanderFactory, yamlData []byte) (runbatch.Runnable, error) {
	var def Definition
	if err := yaml.Unmarshal(yamlData, &def); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidYaml, err)
	}

	if len(def.Commands) == 0 {
		return nil, ErrNoCommands
	}

	// Add command groups to the factory
	for _, group := range def.CommandGroups {
		factory.AddCommandGroup(group.Name, group.Commands)
	}

	// Validate command groups for circular dependencies
	if err := validateCommandGroups(ctx, factory, def.CommandGroups); err != nil {
		return nil, err
	}

	runnables := make([]runbatch.Runnable, 0, len(def.Commands))

	// Wrap in a serial batch with the definition's metadata
	topLevelCommand := &runbatch.SerialBatch{
		BaseCommand: runbatch.NewBaseCommand(
			def.Name,
			".",
			runbatch.RunOnAlways,
			nil,
			nil,
		),
	}

	for i, cmd := range def.Commands {
		// Check for context cancellation during command processing
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: cancelled while processing command %d", ErrConfigurationTimeout, i)
		default:
		}

		// Convert the command to YAML and then process it
		cmdYAML, err := yaml.Marshal(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command %d: %w", i, err)
		}

		runnable, err := factory.CreateRunnableFromYAML(ctx, cmdYAML, topLevelCommand)
		if err != nil {
			return nil, fmt.Errorf("failed to create runnable for command %d: %w", i, err)
		}

		runnable.SetParent(topLevelCommand)
		runnables = append(runnables, runnable)
	}

	// Assign the runnables to the top-level command
	topLevelCommand.Commands = runnables

	return topLevelCommand, nil
}

// validateCommandGroups validates all command groups for circular dependencies.
func validateCommandGroups(ctx context.Context, factory commands.CommanderFactory, groups []CommandGroup) error {
	// Validate each command group for circular dependencies
	for _, group := range groups {
		// Check for context cancellation early
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %v", ErrConfigurationTimeout, ctx.Err())
		default:
		}

		_, err := factory.ResolveCommandGroup(group.Name)
		if err != nil {
			return fmt.Errorf("invalid command group '%s': %w", group.Name, err)
		}
	}

	return nil
}
