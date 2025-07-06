// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import (
	"context"
	"errors"
	"io"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/config/hcl"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/matt-FFFFFF/porch/internal/schema"
)

var _ commands.Commander = (*Commander)(nil)
var _ schema.Provider = (*Commander)(nil)
var _ schema.Writer = (*Commander)(nil)

// Commander is a struct that implements the commands.Commander interface.
type Commander struct {
	schemaGenerator *schema.BaseSchemaGenerator
}

// NewCommander creates a new copycwdtotemp Commander.
func NewCommander() *Commander {
	c := &Commander{}
	c.schemaGenerator = schema.NewBaseSchemaGenerator()

	return c
}

// CreateFromYaml creates a new runnable command and implements the commands.Commander interface.
func (c *Commander) CreateFromYaml(
	ctx context.Context,
	_ commands.CommanderFactory,
	payload []byte,
	parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	def := new(Definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	if def.WorkingDirectory == "" {
		def.WorkingDirectory = "."
	}

	base, err := def.ToBaseCommand(ctx, parent)
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate(commandType), err)
	}

	return New(base), nil
}

// CreateFromHcl creates a new runnable command from an HCL command block and
// implements the commands.Commander interface.
func (c *Commander) CreateFromHcl(
	ctx context.Context,
	_ commands.CommanderFactory,
	hclCommand *hcl.CommandBlock,
	parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	if hclCommand.WorkingDirectory == "" {
		hclCommand.WorkingDirectory = "."
	}

	base, err := commands.HclCommandToBaseCommand(ctx, hclCommand, parent)
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate(commandType), err)
	}

	return New(base), nil
}

// GetSchemaFields returns the schema fields for the copycwdtotemp type.
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
	return "Copies the current working directory to a temporary directory. " +
		"Future working directories will be set to the temporary directory."
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
