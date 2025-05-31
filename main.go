// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package main is the entry point for the avmtool command-line application.
package main

import (
	"context"
	"os"

	"github.com/matt-FFFFFF/pporch/cmd"
	"github.com/matt-FFFFFF/pporch/internal/ctxlog"
	"github.com/matt-FFFFFF/pporch/internal/signalbroker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)
	defer cancel()

	sigCh := signalbroker.New(ctx)

	go signalbroker.Watch(ctx, sigCh, cancel)

	err := cmd.RootCmd.Run(ctx, os.Args)
	if err != nil {
		ctxlog.Logger(ctx).Error("main", "detail", "command failed", "error", err)
		os.Exit(1)
	}

	ctxlog.Logger(ctx).Info("main", "detail", "command completed successfully")
	os.Exit(0)
}
