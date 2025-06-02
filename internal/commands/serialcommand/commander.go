// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package serialcommand provides a command type for running commands in serial.
package serialcommand

import (
	"context"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

var _ commands.Commander = (*Commander)(nil)

// definition represents the YAML configuration for the serial command.
type definition struct {
	commands.BaseDefinition `yaml:",inline"`
	Commands                []any `yaml:"commands"`
}

// Commander is a struct that implements the commands.Commander interface.
type Commander struct{}

// Create creates a new runnable command and implements the commands.Commander interface.
func (c *Commander) Create(ctx context.Context, payload []byte) (runbatch.Runnable, error) {
	def := new(definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	var runnables []runbatch.Runnable

	base, err := def.ToBaseCommand()
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate("serialcommand"), err)
	}

	serialBatch := &runbatch.SerialBatch{
		BaseCommand: base,
	}

	for i, cmd := range def.Commands {
		cmdYAML, err := yaml.Marshal(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command %d: %w", i, err)
		}

		runnable, err := commandregistry.CreateRunnableFromYAML(ctx, cmdYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to create runnable for command %d: %w", i, err)
		}

		runnable.SetParent(serialBatch)

		runnables = append(runnables, runnable)
	}

	serialBatch.Commands = runnables

	return serialBatch, nil
}
