// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package serialcommand provides a command type for running commands in serial.
package serialcommand

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/matt-FFFFFF/porch/internal/schema"
)

var _ commands.Commander = (*Commander)(nil)
var _ schema.Writer = (*Commander)(nil)
var _ schema.Provider = (*Commander)(nil)

// Commander is a struct that implements the commands.Commander interface.
type Commander struct {
	schemaGenerator *schema.BaseSchemaGenerator
}

// NewCommander creates a new serialcommand Commander.
func NewCommander() *Commander {
	c := &Commander{}
	c.schemaGenerator = schema.NewBaseSchemaGenerator()

	return c
}

// Create creates a new runnable command and implements the commands.Commander interface.
func (c *Commander) Create(
	ctx context.Context,
	factory commands.CommanderFactory,
	payload []byte,
) (runbatch.Runnable, error) {
	def := new(Definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	if err := def.Validate(); err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate(commandType), err)
	}

	var runnables []runbatch.Runnable

	base, err := def.ToBaseCommand()
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate(commandType), err)
	}

	serialBatch := &runbatch.SerialBatch{
		BaseCommand: base,
	}

	// Determine which commands to use
	var commandsToProcess []any

	switch {
	case def.CommandGroup != "":
		commands, err := factory.ResolveCommandGroup(def.CommandGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve command group %q: %w", def.CommandGroup, err)
		}

		commandsToProcess = commands
	default:
		commandsToProcess = def.Commands
	}

	for i, cmd := range commandsToProcess {
		// Check for context cancellation during command processing
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("serial command creation cancelled while processing command %d: %w", i, ctx.Err())
		default:
		}

		cmdYAML, err := yaml.Marshal(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command %d: %w", i, err)
		}

		runnable, err := factory.CreateRunnableFromYAML(ctx, cmdYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to create runnable for command %d: %w", i, err)
		}

		runnable.SetParent(serialBatch)

		runnables = append(runnables, runnable)
	}

	serialBatch.Commands = runnables

	return serialBatch, nil
}

// GetSchemaFields returns the schema fields for the serialcommand type.
func (c *Commander) GetSchemaFields() []schema.Field {
	def := &Definition{}
	generator := schema.NewGenerator()

	schemaObj, err := generator.Generate(commandType, def)
	if err != nil {
		return []schema.Field{}
	}

	return schemaObj.Fields
}

// GetCommandType returns the command type string.
func (c *Commander) GetCommandType() string {
	return commandType
}

// GetCommandDescription returns a description of what this command does.
func (c *Commander) GetCommandDescription() string {
	return "Executes a list of commands sequentially (one after another)"
}

// GetExampleDefinition returns an example definition for YAML generation.
func (c *Commander) GetExampleDefinition() interface{} {
	return &Definition{
		BaseDefinition: commands.BaseDefinition{
			Type: commandType,
			Name: "example-serial-command",
		},
		Commands: []any{
			map[string]any{
				"type":         "shellcommand",
				"name":         "first-command",
				"command_line": "echo 'First command'",
			},
			map[string]any{
				"type":         "shellcommand",
				"name":         "second-command",
				"command_line": "echo 'Second command'",
			},
		},
	}
}

// WriteYAMLExample writes the YAML schema documentation to the provided writer.
func (c *Commander) WriteYAMLExample(w io.Writer) error {
	return c.schemaGenerator.WriteYAMLExample(w, c.GetExampleDefinition()) //nolint:wrapcheck
}

// WriteMarkdownDoc writes the Markdown schema documentation to the provided writer.
func (c *Commander) WriteMarkdownDoc(w io.Writer) error {
	return c.schemaGenerator.WriteMarkdownExample( //nolint:wrapcheck
		w,
		c.GetCommandType(),
		c.GetExampleDefinition(),
		c.GetCommandDescription(),
	)
}

// WriteJSONSchema writes the JSON schema to the provided writer.
func (c *Commander) WriteJSONSchema(w io.Writer, f commands.CommanderFactory) error {
	return c.schemaGenerator.WriteJSONSchema(w, f) //nolint:wrapcheck
}
