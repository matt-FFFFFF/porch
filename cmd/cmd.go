// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package cmd contains the command-line interface (CLI) for the module.
package cmd

import (
	"os"

	"github.com/matt-FFFFFF/porch/cmd/config"
	"github.com/matt-FFFFFF/porch/cmd/run"
	"github.com/matt-FFFFFF/porch/cmd/show"
	"github.com/urfave/cli/v3"
)

// RootCmd is the root command for the CLI.
var RootCmd = &cli.Command{
	Commands: []*cli.Command{
		config.ConfigCmd,
		run.RunCmd,
		show.ShowCmd,
	},
	Writer:    os.Stdout,
	ErrWriter: os.Stderr,
	Name:      "porch",
	Description: `Porch is a sophisticated Go-based process orchestration framework
designed for running and managing complex command workflows. It provides a flexible,
YAML-driven approach to define, compose, and execute command chains with advanced
flow control, parallel processing, and comprehensive error handling.`,
	Usage:     "porch run myfile.yaml",
	Copyright: "Copyright (c) matt-FFFFFF 2025. All rights reserved.",
	Authors: []any{
		"Matt White (matt-FFFFFF)",
	},
	EnableShellCompletion: true,
}
