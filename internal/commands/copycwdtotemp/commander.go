// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import (
	"context"
	"errors"
	"io"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

var _ commands.Commander = (*Commander)(nil)
var _ commands.SchemaProvider = (*Commander)(nil)
var _ commands.SchemaWriter = (*Commander)(nil)

// Commander is a struct that implements the commands.Commander interface.
type Commander struct {
	schemaGenerator *commands.BaseSchemaGenerator
}

// NewCommander creates a new copycwdtotemp Commander.
func NewCommander() *Commander {
	c := &Commander{}
	c.schemaGenerator = commands.NewBaseSchemaGenerator(c)
	return c
}

// Create creates a new runnable command and implements the commands.Commander interface.
func (c *Commander) Create(_ context.Context, payload []byte) (runbatch.Runnable, error) {
	def := new(Definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	if def.WorkingDirectory == "" {
		def.WorkingDirectory = "."
	}

	base, err := def.ToBaseCommand()
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate(commandType), err)
	}

	return New(base), nil
}

// GetSchemaFields returns the schema fields for the copycwdtotemp type.
func (c *Commander) GetSchemaFields() []commands.SchemaField {
	def := &Definition{}
	generator := commands.NewSchemaGenerator()
	schema, err := generator.GenerateSchema(commandType, def)
	if err != nil {
		return []commands.SchemaField{}
	}
	return schema.Fields
}

// GetCommandType returns the command type string.
func (c *Commander) GetCommandType() string {
	return commandType
}

// GetCommandDescription returns a description of what this command does.
func (c *Commander) GetCommandDescription() string {
	return "Copies the current working directory to a temporary directory"
}

// GetExampleDefinition returns an example definition for YAML generation.
func (c *Commander) GetExampleDefinition() interface{} {
	return &Definition{
		BaseDefinition: commands.BaseDefinition{
			Type: commandType,
			Name: "example-copy-cwd-to-temp",
		},
	}
}

// WriteYAMLSchema writes the YAML schema documentation to the provided writer.
func (c *Commander) WriteYAMLSchema(w io.Writer) error {
	return c.schemaGenerator.WriteYAMLSchema(w)
}

// WriteMarkdownSchema writes the Markdown schema documentation to the provided writer.
func (c *Commander) WriteMarkdownSchema(w io.Writer) error {
	return c.schemaGenerator.WriteMarkdownSchema(w)
}

// WriteJSONSchema writes the JSON schema to the provided writer.
func (c *Commander) WriteJSONSchema(w io.Writer) error {
	return c.schemaGenerator.WriteJSONSchema(w)
}
