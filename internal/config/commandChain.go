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
)

// Definition represents the root configuration structure.
type Definition struct {
	Name          string         `yaml:"name" json:"name" docdesc:"Name of the configuration"`
	Description   string         `yaml:"description" json:"description" docdesc:"Description of what this configuration does"`
	Commands      []any          `yaml:"commands" json:"commands" docdesc:"List of commands to execute"`
	CommandGroups []CommandGroup `yaml:"command_groups" json:"command_groups" docdesc:"List of command groups"`
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

	runnables := make([]runbatch.Runnable, 0, len(def.Commands))

	// Wrap in a serial batch with the definition's metadata
	topLevelCommand := &runbatch.SerialBatch{
		BaseCommand: &runbatch.BaseCommand{
			Label: def.Name,
		},
	}

	for _, cmd := range def.Commands {
		// Convert the command to YAML and then process it
		cmdYAML, err := yaml.Marshal(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command: %w", err)
		}

		runnable, err := factory.CreateRunnableFromYAML(ctx, cmdYAML)

		if err != nil {
			return nil, fmt.Errorf("failed to create runnable: %w", err)
		}

		runnable.SetParent(topLevelCommand)
		runnables = append(runnables, runnable)
	}

	// Assign the runnables to the top-level command
	topLevelCommand.Commands = runnables

	return topLevelCommand, nil
}
