// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package signalbroker

import (
	"context"
	"fmt"
	"os"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
)

// Watch monitors the signal channel and handles signals.
// It will cancel the context on the second signal of a given type that received.
func Watch(ctx context.Context, sigCh chan os.Signal, cancel context.CancelFunc) {
	sigMap := make(map[os.Signal]struct{})
	for sig := range sigCh {
		if _, ok := sigMap[sig]; ok {
			ctxlog.Logger(ctx).Warn(
				fmt.Sprintf("Received second signal of type %s, forcefully terminating", sig.String()))
			close(sigCh)
			cancel()

			return
		}

		ctxlog.Logger(ctx).Warn(
			fmt.Sprintf(
				"Received signal of type %s, attempting to gracefully terminate. "+
					"Send the same signal again to forcefully terminate.",
				sig.String()))

		sigMap[sig] = struct{}{}
	}
}
