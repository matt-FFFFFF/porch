package main

import (
	"context"
	"os"

	"github.com/matt-FFFFFF/avmtool/cmd"
	"github.com/matt-FFFFFF/avmtool/internal/ctxlog"
	"github.com/matt-FFFFFF/avmtool/internal/signalbroker"
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
