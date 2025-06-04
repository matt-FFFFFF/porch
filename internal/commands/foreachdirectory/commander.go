// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/foreachproviders"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

var _ commands.Commander = (*Commander)(nil)
var _ commands.SchemaProvider = (*Commander)(nil)
var _ commands.SchemaWriter = (*Commander)(nil)

// Commander implements the commands.Commander interface for the foreachdirectory command.
type Commander struct {
	schemaGenerator *commands.BaseSchemaGenerator
}

// NewCommander creates a new foreachdirectory Commander.
func NewCommander() *Commander {
	c := &Commander{}
	c.schemaGenerator = commands.NewBaseSchemaGenerator(c)
	return c
}

// Create creates a new runnable command based on the provided YAML payload.
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

	mode, err := runbatch.ParseForEachMode(def.Mode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse foreach mode: %q %w", def.Mode, err)
	}

	if def.WorkingDirectoryStrategy == "" {
		def.WorkingDirectoryStrategy = runbatch.CwdStrategyNone.String()
	}

	strat, err := runbatch.ParseCwdStrategy(def.WorkingDirectoryStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to parse working directory strategy: %q %w", def.WorkingDirectoryStrategy, err)
	}

	forEachCommand := &runbatch.ForEachCommand{
		BaseCommand:   base,
		ItemsProvider: foreachproviders.ListDirectoriesDepth(def.Depth, foreachproviders.IncludeHidden(def.IncludeHidden)),
		Mode:          mode,
		CwdStrategy:   strat,
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

		runnable.SetParent(forEachCommand)

		runnables = append(runnables, runnable)
	}

	forEachCommand.Commands = runnables

	return forEachCommand, nil
}

// GetSchemaFields returns the schema fields for the foreachdirectory type.
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
	return "Executes commands in each directory found by traversing the filesystem"
}

// GetExampleDefinition returns an example definition for YAML generation.
func (c *Commander) GetExampleDefinition() interface{} {
	return &Definition{
		BaseDefinition: commands.BaseDefinition{
			Type: commandType,
			Name: "example-foreach-directory",
		},
		Mode:                     "parallel",
		Depth:                    2,
		IncludeHidden:            false,
		WorkingDirectoryStrategy: "item_relative",
		Commands: []any{
			map[string]any{
				"type":         "shellcommand",
				"name":         "directory-command",
				"command_line": "pwd && ls -la",
			},
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
