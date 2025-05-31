// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/pporch/internal/commandregistry"
	"github.com/matt-FFFFFF/pporch/internal/runbatch"

	// blank import used to run init functions of all commands to register them.
	_ "github.com/matt-FFFFFF/pporch/internal/allcommands"
)

var (
	// ErrInvalidYaml is returned when the YAML configuration is invalid.
	ErrInvalidYaml = errors.New("invalid YAML")
	// ErrNoCommands is returned when no commands are specified in the configuration.
	ErrNoCommands = errors.New("no commands specified")
)

// Definition represents the root configuration structure.
type Definition struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Commands    []interface{} `yaml:"commands"`
}

// BuildFromYAML creates a runnable from YAML configuration.
func BuildFromYAML(ctx context.Context, yamlData []byte) (runbatch.Runnable, error) {
	var def Definition
	if err := yaml.Unmarshal(yamlData, &def); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidYaml, err)
	}

	if len(def.Commands) == 0 {
		return nil, ErrNoCommands
	}

	runnables := make([]runbatch.Runnable, 0, len(def.Commands))

	for _, cmd := range def.Commands {
		// Convert the command to YAML and then process it
		cmdYAML, err := yaml.Marshal(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command: %w", err)
		}

		runnable, err := commandregistry.CreateRunnableFromYAML(ctx, cmdYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to create runnable: %w", err)
		}

		runnables = append(runnables, runnable)
	}

	// Wrap in a serial batch with the definition's metadata
	result := &runbatch.SerialBatch{
		Label:    def.Name,
		Commands: runnables,
	}

	return result, nil
}
