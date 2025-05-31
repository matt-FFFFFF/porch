// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package commandregistry provides a registry for command types and their commanders.
package commandregistry

import (
	"context"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/pporch/internal/commands"
	"github.com/matt-FFFFFF/pporch/internal/runbatch"
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
type Registry map[string]commands.Commander

// DefaultRegistry is the default registry for command types.
var DefaultRegistry = make(Registry)

// Register registers a new command type with its commander.
func Register(commandType string, commander commands.Commander) {
	DefaultRegistry[commandType] = commander
}

// RawCommand represents a command with its type and raw YAML data.
type RawCommand struct {
	Type string      `yaml:"type"`
	Data interface{} `yaml:",inline"`
}

// CreateRunnableFromYAML creates a runnable from YAML data using the registered commanders.
func CreateRunnableFromYAML(ctx context.Context, yamlData []byte) (runbatch.Runnable, error) {
	var rawCmd RawCommand
	if err := yaml.Unmarshal(yamlData, &rawCmd); err != nil {
		return nil, errors.Join(ErrCommandUnmarshal, err)
	}

	commander, exists := DefaultRegistry[rawCmd.Type]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCommandType, rawCmd.Type)
	}

	// Re-marshal the original data to pass to the commander
	runnable, err := commander.Create(ctx, yamlData)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCommandCreation, rawCmd.Type, err)
	}

	return runnable, nil
}
