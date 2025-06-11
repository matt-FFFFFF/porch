// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package signalbroker

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/matt-FFFFFF/porch/internal/color"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
)

const (
	separator = "================================================"
)

// Watch monitors the signal channel and handles signals.
// It will cancel the context on the second signal of a given type that received.
func Watch(ctx context.Context, sigCh chan os.Signal, cancel context.CancelFunc) {
	sigMap := make(map[os.Signal]struct{})
	for sig := range sigCh {
		if _, ok := sigMap[sig]; ok {
			ctxlog.Logger(ctx).Info(
				"watchdog",
				"detail", "received second signal of type, forcefully terminating",
				"signal", sig.String())
			close(sigCh)
			cancel()

			return
		}

		ctxlog.Logger(ctx).Info("watchdog", "detail", "received first signal of type, no-op", "signal", sig.String())

		sb := strings.Builder{}
		sb.WriteString(color.Colorize(separator, color.FgHiRed))
		sb.WriteString("\n")
		sb.WriteString(color.Colorize(`=  Received signal and attempting graceful termination: `, color.FgHiRed))
		sb.WriteString(color.Colorize(sig.String(), color.FgHiYellow))
		sb.WriteString("\n")
		sb.WriteString(color.Colorize(`=  Send the same signal again to forcefully terminate`, color.FgHiRed))
		sb.WriteString("\n")
		sb.WriteString(color.Colorize(separator, color.FgHiRed))
		sb.WriteString("\n")
		fmt.Print(sb.String())

		sigMap[sig] = struct{}{}
	}
}
