// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"context"
	"fmt"
	"os"

	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/schema"

	"github.com/urfave/cli/v3"
)

var ConfigCmd = &cli.Command{
	Name:   "config",
	Usage:  "Get info on configuration format and commands",
	Action: actionFunc,
	Commands: []*cli.Command{
		schemaCmd,
	},
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name: "command",
		},
	},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "markdown",
			Usage:   "Output the configuration example in Markdown format",
			Aliases: []string{"md"},
		},
	},
}

func actionFunc(ctx context.Context, cmd *cli.Command) error {
	factory, ok := ctx.Value(commands.FactoryContextKey{}).(commands.CommanderFactory)
	if !ok {
		return cli.Exit("failed to get command factory from context", 1)
	}

	cmdName := cmd.StringArg("command")
	if cmdName == "" {
		fmt.Printf("Available commands:\n\n")
		for k := range factory.Iter() {
			fmt.Printf("- %s\n", k)
		}
		fmt.Printf("\nUse `porch config <command>` to get examples for a specific command.\n")
		return nil
	}

	cmdr, ok := factory.Get(cmdName)
	if !ok {
		return cli.Exit(fmt.Sprintf("unknown command: %s", cmdName), 1)
	}
	sp, ok := cmdr.(schema.Provider)
	if !ok {
		return cli.Exit(fmt.Sprintf("command %s does not provide schema", cmdName), 1)
	}
	sw, ok := cmdr.(schema.Writer)
	if !ok {
		return cli.Exit(fmt.Sprintf("command %s does not provide schema writer", cmdName), 1)
	}

	fmt.Printf("%s\n\nSchema:\n\n", sp.GetCommandDescription())

	if cmd.Bool("markdown") {
		sw.WriteMarkdownDoc(os.Stdout)
	}
	sw.WriteYAMLExample(os.Stdout)

	return nil
}

var schemaCmd = &cli.Command{
	Name:   "schema",
	Usage:  "Output the JSON schema for the configuration",
	Action: schemaCmdActionFunc,
}

func schemaCmdActionFunc(ctx context.Context, cmd *cli.Command) error {
	factory, ok := ctx.Value(commands.FactoryContextKey{}).(commands.CommanderFactory)
	if !ok {
		return cli.Exit("failed to get command factory from context", 1)
	}

	sw := schema.NewBaseSchemaGenerator()
	sw.WriteJSONSchema(os.Stdout, factory)

	return nil
}
