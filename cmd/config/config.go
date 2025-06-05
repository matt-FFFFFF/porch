// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"context"
	"fmt"

	"github.com/matt-FFFFFF/porch/internal/commands"

	"github.com/urfave/cli/v3"
)

var ConfigCmd = &cli.Command{
	Name:     "config",
	Usage:    "Get info on configuration format and commands",
	Action:   actionFunc,
	Commands: []*cli.Command{},
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name: "command",
		},
	},
}

func actionFunc(ctx context.Context, cmd *cli.Command) error {
	val, ok := ctx.Value(commands.FactoryContextKey{}).(commands.CommanderFactory)
	if !ok {
		return cli.Exit("failed to get command factory from context", 1)
	}

	if cmd.StringArg("command") == "" {
		fmt.Printf("Available commands:\n\n")
		for k := range val.Iter() {
			fmt.Printf("- %s\n", k)
		}
	}

	return nil
}
