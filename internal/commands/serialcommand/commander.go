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
	"github.com/matt-FFFFFF/porch/internal/commandregistry"
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
func (c *Commander) Create(ctx context.Context, payload []byte) (runbatch.Runnable, error) {
	def := new(Definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	var runnables []runbatch.Runnable

	base, err := def.ToBaseCommand()
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate(commandType), err)
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
	return c.schemaGenerator.WriteYAMLSchema(w)
}

// WriteMarkdownDoc writes the Markdown schema documentation to the provided writer.
func (c *Commander) WriteMarkdownDoc(w io.Writer) error {
	return c.schemaGenerator.WriteMarkdownSchema(w)
}

// WriteJSONSchema writes the JSON schema to the provided writer.
func (c *Commander) WriteJSONSchema(w io.Writer) error {
	return c.schemaGenerator.WriteJSONSchema(w)
}
