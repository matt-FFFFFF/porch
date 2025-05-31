// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package parallelcommand provides a command type for running commands in parallel.
package parallelcommand

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/pporch/internal/commandregistry"
	"github.com/matt-FFFFFF/pporch/internal/commands"
	"github.com/matt-FFFFFF/pporch/internal/runbatch"
)

var _ commands.Commander = (*Commander)(nil)

// Definition represents the YAML configuration for the parallel command.
type Definition struct {
	commands.BaseDefinition `yaml:",inline"`
	Commands                []interface{} `yaml:"commands"`
}

// Commander is a struct that implements the commands.Commander interface.
type Commander struct{}

// Create creates a new runnable command and implements the commands.Commander interface.
func (c *Commander) Create(ctx context.Context, payload []byte) (runbatch.Runnable, error) {
	def := new(Definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parallel command definition: %w", err)
	}

	var runnables []runbatch.Runnable

	for i, cmd := range def.Commands {
		cmdYAML, err := yaml.Marshal(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command %d: %w", i, err)
		}

		runnable, err := commandregistry.CreateRunnableFromYAML(ctx, cmdYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to create runnable for command %d: %w", i, err)
		}

		runnables = append(runnables, runnable)
	}

	return &runbatch.ParallelBatch{
		Label:    def.Name,
		Commands: runnables,
	}, nil
}
