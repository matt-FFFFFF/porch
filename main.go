package main

import (
	"context"

	"github.com/matt-FFFFFF/avmtool/internal/ctxlog"
	"github.com/matt-FFFFFF/avmtool/internal/signalbroker"
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

}
