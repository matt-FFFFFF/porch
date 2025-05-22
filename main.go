package main

import (
	"context"
	"os"

	"github.com/matt-FFFFFF/avmtool/internal/ctxlog"
	"github.com/matt-FFFFFF/avmtool/internal/signalbroker"
	"github.com/urfave/cli/v3"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)
	defer cancel()

	sigCh := signalbroker.New(ctx)

	go func() {
		<-sigCh
		cancel()
	}()

	(&cli.Command{}).Run(ctx, os.Args)
}
