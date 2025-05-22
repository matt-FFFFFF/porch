package signalbroker

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/matt-FFFFFF/avmtool/internal/ctxlog"
)

var termSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGQUIT,
	os.Interrupt,
}

// New creates a new signal broker that listens for OS signals that should terminate the process.
func New(ctx context.Context, sigs ...os.Signal) chan os.Signal {
	ch := make(chan os.Signal, 1)
	if len(sigs) == 0 {
		sigs = termSignals
	}
	ctxlog.Debug(ctx, "signalbroker", "detail", "creating signal broker", "signals", sigs)
	signal.Notify(ch, sigs...)
	return ch
}
