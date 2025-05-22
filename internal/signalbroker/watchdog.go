package signalbroker

import (
	"context"
	"os"

	"github.com/matt-FFFFFF/avmtool/internal/ctxlog"
)

// Watch monitors the signal channel and handles signals.
// It will cancel the context on the second signal of a given type that received.
func Watch(ctx context.Context, sigCh chan os.Signal, cancel context.CancelFunc) {
	sigMap := make(map[os.Signal]struct{})
	for sig := range sigCh {
		if _, ok := sigMap[sig]; ok {
			ctxlog.Logger(ctx).Info("watchdog", "detail", "received second signal of type, forcefully terminating", "signal", sig.String())
			close(sigCh)
			cancel()

			return
		}

		ctxlog.Logger(ctx).Info("watchdog", "detail", "received first signal of type, no-op", "signal", sig.String())

		sigMap[sig] = struct{}{}
	}
}
