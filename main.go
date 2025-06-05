// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package main is the entry point for the porch command-line application.
package main

import (
	"context"
	"os"

	"github.com/matt-FFFFFF/porch/cmd"
	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/commands/copycwdtotemp"
	"github.com/matt-FFFFFF/porch/internal/commands/foreachdirectory"
	"github.com/matt-FFFFFF/porch/internal/commands/parallelcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/serialcommand"
	"github.com/matt-FFFFFF/porch/internal/commands/shellcommand"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/signalbroker"
	"github.com/urfave/cli/v3"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)
	defer cancel()

	sigCh := signalbroker.New(ctx)

	go signalbroker.Watch(ctx, sigCh, cancel)

	cmd.RootCmd.Version = version
	cmd.RootCmd.ExtraInfo = func() map[string]string {
		return map[string]string{
			"commit": commit,
		}
	}

	factory := commandregistry.New(
		serialcommand.Register,
		parallelcommand.Register,
		foreachdirectory.Register,
		copycwdtotemp.Register,
		shellcommand.Register,
	)

	cmd.RootCmd.Before = cli.BeforeFunc(func(c context.Context, cmd *cli.Command) (context.Context, error) {
		ctx = context.WithValue(ctx, commands.FactoryContextKey{}, factory)
		return ctx, nil
	})

	_ = cmd.RootCmd.Run(ctx, os.Args) // Err is handled by cli framework

	ctxlog.Logger(ctx).Info("command completed successfully")
}
