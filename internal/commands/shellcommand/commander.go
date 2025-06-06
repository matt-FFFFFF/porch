// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package shellcommand

import (
	"context"
	"errors"
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

// NewCommander creates a new shellcommand Commander.
func NewCommander() *Commander {
	c := &Commander{}
	c.schemaGenerator = schema.NewBaseSchemaGenerator()

	return c
}

// Create creates a new runnable command and implements the commands.Commander interface.
func (c *Commander) Create(ctx context.Context, _ commands.CommanderFactory, payload []byte) (runbatch.Runnable, error) {
	def := new(Definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	base, err := def.ToBaseCommand()
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate(commandType), err)
	}

	return New(ctx, base, def.CommandLine, def.SuccessExitCodes, def.SkipExitCodes)
}

// GetSchemaFields returns the schema fields for the shellcommand type.
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
	return "Executes a shell command with configurable success and skip exit codes"
}

// GetExampleDefinition returns an example definition for YAML generation.
func (c *Commander) GetExampleDefinition() interface{} {
	return &Definition{
		BaseDefinition: commands.BaseDefinition{
			Type: commandType,
			Name: "example-shell-command",
		},
		CommandLine:      "echo 'Hello, World!'",
		SuccessExitCodes: []int{0},
		SkipExitCodes:    []int{2},
	}
}

// WriteYAMLExample writes the YAML schema documentation to the provided writer.
func (c *Commander) WriteYAMLExample(w io.Writer) error {
	return c.schemaGenerator.WriteYAMLExample(w, c.GetExampleDefinition())
}

// WriteMarkdownDoc writes the Markdown schema documentation to the provided writer.
func (c *Commander) WriteMarkdownDoc(w io.Writer) error {
	return c.schemaGenerator.WriteMarkdownExample(w, c.GetCommandType(), c.GetExampleDefinition(), c.GetCommandDescription())
}

// WriteJSONSchema writes the JSON schema to the provided writer.
func (c *Commander) WriteJSONSchema(w io.Writer, f commands.CommanderFactory) error {
	return c.schemaGenerator.WriteJSONSchema(w, f)
}
