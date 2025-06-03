// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package cmd contains the command-line interface (CLI) for the module.
package cmd

import (
	"github.com/matt-FFFFFF/porch/cmd/run"
	"github.com/matt-FFFFFF/porch/cmd/show"
	"github.com/urfave/cli/v3"
)

// RootCmd is the root command for the CLI.
var RootCmd = &cli.Command{
	Commands: []*cli.Command{
		run.RunCmd,
		show.ShowCmd,
	},
	Name: "porch",
}
