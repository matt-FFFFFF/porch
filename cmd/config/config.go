// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"context"
	"fmt"
	"maps"
	"os"

	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/schema"
	"github.com/urfave/cli/v3"
)

var ConfigCmd = &cli.Command{
	Name:     "config",
	Usage:    "Get info on configuration format and commands",
	Action:   actionFunc,
	Commands: []*cli.Command{},
}

func init() {
	for k, v := range maps.All(commandregistry.DefaultRegistry) {
		p, ok := v.(schema.Provider)
		if !ok {
			continue
		}
		w, ok := v.(schema.Writer)
		if !ok {
			continue
		}

		ConfigCmd.Commands = append(ConfigCmd.Commands, &cli.Command{
			Name:        k,
			Description: p.GetCommandDescription(),
			Action: func(ctx context.Context, cmd *cli.Command) error {
				// This command is a placeholder for future implementation.
				// Currently, it does not perform any actions.
				fmt.Println("Command:", k)
				fmt.Println("Description:", p.GetCommandDescription())
				fmt.Println()
				fmt.Println("Schema:")
				w.WriteYAMLExample(os.Stdout)
				return nil
			},
		})
	}
}

func actionFunc(ctx context.Context, cmd *cli.Command) error {
	for k := range maps.Keys(commandregistry.DefaultRegistry) {
		fmt.Println(k)
	}
	return nil
}
