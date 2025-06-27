// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/foreachproviders"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/matt-FFFFFF/porch/internal/schema"
)

var _ commands.Commander = (*Commander)(nil)
var _ schema.Writer = (*Commander)(nil)
var _ schema.Provider = (*Commander)(nil)

// Commander implements the commands.Commander interface for the foreachdirectory command.
type Commander struct {
	schemaGenerator *schema.BaseSchemaGenerator
}

// NewCommander creates a new foreachdirectory Commander.
func NewCommander() *Commander {
	c := &Commander{}
	c.schemaGenerator = schema.NewBaseSchemaGenerator()

	return c
}

// Create creates a new runnable command based on the provided YAML payload.
func (c *Commander) Create(
	ctx context.Context,
	factory commands.CommanderFactory,
	payload []byte,
	parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	def := new(Definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	if err := def.Validate(); err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate("foreachdirectory"), err)
	}

	var runnables []runbatch.Runnable

	base, err := def.ToBaseCommand(ctx, parent)
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate("foreachdirectory"), err)
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

	itemsSkipOnErrors := []error{}
	if def.SkipOnNotExist {
		itemsSkipOnErrors = append(itemsSkipOnErrors, os.ErrNotExist)
	}

	forEachCommand := &runbatch.ForEachCommand{
		BaseCommand: base,
		ItemsProvider: foreachproviders.ListDirectoriesDepth(
			def.Depth, foreachproviders.IncludeHidden(def.IncludeHidden),
		),
		Mode:              mode,
		CwdStrategy:       strat,
		ItemsSkipOnErrors: itemsSkipOnErrors,
	}

	// Determine which commands to use
	var commandsToProcess []any

	if def.CommandGroup != "" {
		commands, err := factory.ResolveCommandGroup(def.CommandGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve command group %q: %w", def.CommandGroup, err)
		}

		commandsToProcess = commands
	} else {
		commandsToProcess = def.Commands
	}

	for i, cmd := range commandsToProcess {
		cmdYAML, err := yaml.Marshal(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command %d: %w", i, err)
		}

		runnable, err := factory.CreateRunnableFromYAML(ctx, cmdYAML, forEachCommand)
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
	return `Executes commands in each directory found by traversing the filesystem.
Directories are found based on the specified depth and whether hidden directories are included.
Commands are executed in parallel or serially based on the specified mode,
and the working directory for each command can be set relative to the item being processed.

Set "working_directory_strategy: \"item_relative\"" to run commands in the directory of each item.

Additionally, an environment variable named "ITEM" is set to the current item being processed.`
}

// GetExampleDefinition returns an example definition for YAML generation.
func (c *Commander) GetExampleDefinition() interface{} {
	return &Definition{
		BaseDefinition: commands.BaseDefinition{
			Type: commandType,
			Name: "example-foreach-directory",
		},
		Mode:                     "parallel",
		Depth:                    2, //nolint:mnd
		IncludeHidden:            false,
		WorkingDirectoryStrategy: "item_relative",
		SkipOnNotExist:           false,
		Commands: []any{
			map[string]any{
				"type":         "shellcommand",
				"name":         "directory-command",
				"command_line": "pwd && ls -la",
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
