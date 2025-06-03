// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package main is the entry point for the pporch command-line application.
package main

import (
	"context"
	"os"

	"github.com/matt-FFFFFF/porch/cmd"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/signalbroker"
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

	err := cmd.RootCmd.Run(ctx, os.Args)
	if err != nil {
		ctxlog.Logger(ctx).Error("command failed", "error", err)
		os.Exit(1)
	}

	ctxlog.Logger(ctx).Info("command completed successfully")
	os.Exit(0)
}
