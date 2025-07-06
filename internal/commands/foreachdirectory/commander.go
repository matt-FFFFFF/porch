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
	"github.com/matt-FFFFFF/porch/internal/config/hcl"
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

// CreateFromYaml creates a new runnable command based on the provided YAML payload.
func (c *Commander) CreateFromYaml(
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

	forEachCommand, err := New(
		ctx,
		base,
		def.Depth,
		def.IncludeHidden,
		def.Mode,
		def.WorkingDirectoryStrategy,
		def.SkipOnNotExist,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create foreach command: %w", err)
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

		runnables = append(runnables, runnable)
	}

	forEachCommand.Commands = runnables

	return forEachCommand, nil
}

// CreateFromHcl creates a new runnable command from an HCL command block.
func (c *Commander) CreateFromHcl(
	ctx context.Context,
	factory commands.CommanderFactory,
	hclCommand *hcl.CommandBlock,
	parent runbatch.Runnable,
) (runbatch.Runnable, error) {
	base, err := commands.HclCommandToBaseCommand(ctx, hclCommand, parent)
	if err != nil {
		return nil, errors.Join(
			commands.NewErrCommandCreate(commandType),
			commands.ErrFailedToCreateRunnable,
			err,
		)
	}

	forEachCommand, err := New(
		ctx,
		base,
		hclCommand.Depth,
		hclCommand.IncludeHidden,
		hclCommand.Mode,
		hclCommand.WorkingDirectoryStrategy,
		hclCommand.SkipOnNotExist,
	)
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate(commandType), err)
	}

	for _, cmd := range hclCommand.Commands {
		cmd := cmd
		// Check for context cancellation during command processing
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("foreach command creation cancelled while processing command: %w", ctx.Err())
		default:
		}

		runnable, err := factory.CreateRunnableFromHCL(ctx, cmd, forEachCommand)
		if err != nil {
			return nil, errors.Join(
				commands.NewErrCommandCreate(commandType),
				commands.ErrFailedToCreateRunnable,
				err,
			)
		}

		forEachCommand.Commands = append(forEachCommand.Commands, runnable)
	}

	return forEachCommand, nil
}

// New creates a new ForEachCommand for iterating over directories.
func New(
	_ context.Context,
	base *runbatch.BaseCommand,
	depth int,
	includeHidden bool,
	mode, workingDirectoryStrategy string,
	skipOnNotExist bool,
) (*runbatch.ForEachCommand, error) {
	if base == nil {
		return nil, commands.ErrNilParent
	}

	forEachMode, err := runbatch.ParseForEachMode(mode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse foreach mode: %q %w", mode, err)
	}

	itemsSkipOnErrors := []error{}
	if skipOnNotExist {
		itemsSkipOnErrors = append(itemsSkipOnErrors, os.ErrNotExist)
	}

	if workingDirectoryStrategy == "" {
		workingDirectoryStrategy = "none"
	}

	strat, err := runbatch.ParseCwdStrategy(workingDirectoryStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to parse working directory strategy: %q %w", workingDirectoryStrategy, err)
	}

	return &runbatch.ForEachCommand{
		BaseCommand:       base,
		ItemsProvider:     foreachproviders.ListDirectoriesDepth(depth, foreachproviders.IncludeHidden(includeHidden)),
		Mode:              forEachMode,
		CwdStrategy:       strat,
		ItemsSkipOnErrors: itemsSkipOnErrors,
		Commands:          []runbatch.Runnable{},
	}, nil
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
