// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package main contains the porch command-line interface (CLI).
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matt-FFFFFF/porch"
	"github.com/matt-FFFFFF/porch/cmd/porch/config"
	"github.com/matt-FFFFFF/porch/cmd/porch/run"
	"github.com/matt-FFFFFF/porch/cmd/porch/show"
	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/commands/copycwdtotemp"
	"github.com/matt-FFFFFF/porch/internal/commands/foreachdirectory"
	"github.com/matt-FFFFFF/porch/internal/commands/parallelcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/pwshcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/serialcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/shellcommand"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/signalbroker"
	"github.com/urfave/cli/v3"
)

// rootCmd is the root command for the CLI.
var rootCmd = &cli.Command{
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

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)
	defer cancel()

	sigCh := signalbroker.New(ctx)

	go signalbroker.Watch(ctx, sigCh, cancel)

	rootCmd.Version = fmt.Sprintf("%s (commit: %s)", porch.Version, porch.Commit)

	factory := commandregistry.New(
		serialcommand.Register,
		parallelcommand.Register,
		foreachdirectory.Register,
		copycwdtotemp.Register,
		shellcommand.Register,
		pwshcommand.Register,
	)

	ctx = context.WithValue(ctx, commands.FactoryContextKey{}, factory)

	err := rootCmd.Run(ctx, os.Args) // Err is handled by cli framework

	// Check if the context was cancelled (e.g., due to signals)
	if ctx.Err() != nil {
		ctxlog.Logger(ctx).Error("command terminated due to cancellation", "error", ctx.Err())
		os.Exit(1)
	}

	if err != nil {
		ctxlog.Logger(ctx).Error("command execution failed", "error", err)
		os.Exit(1)
	}

	ctxlog.Logger(ctx).Info("command completed successfully")
}
