// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/avmtool/internal/commandregistry"
	"github.com/matt-FFFFFF/avmtool/internal/runbatch"

	_ "github.com/matt-FFFFFF/avmtool/internal/commands/copycwdtotemp"
	_ "github.com/matt-FFFFFF/avmtool/internal/commands/parallelcommand"
	_ "github.com/matt-FFFFFF/avmtool/internal/commands/serialcommand"
	_ "github.com/matt-FFFFFF/avmtool/internal/commands/shellcommand"
)

var (
	ErrTooManyRootCommands = errors.New("too many root commands")
	ErrInvalidYaml         = errors.New("invalid YAML")
	ErrNoCommands          = errors.New("no commands specified")
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

	if len(def.Commands) > 1 {
		return nil, ErrTooManyRootCommands
	}

	// Convert the command to YAML and then process it
	cmdYAML, err := yaml.Marshal(def.Commands[0])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	runnable, err := commandregistry.CreateRunnableFromYAML(ctx, cmdYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to create runnable: %w", err)
	}

	// Wrap in a serial batch with the definition's metadata
	result := &runbatch.SerialBatch{
		Label:    def.Name,
		Commands: []runbatch.Runnable{runnable},
	}

	return result, nil
}
