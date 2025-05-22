// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package signalbroker provides a way to listen for OS signals and handle them gracefully.
// By default it listens for os.Interrupt, syscall.SIGINT, syscall.SIGTERM, and syscall.SIGQUIT signals.
//
// It also contains a watchdog function that can be used to watch for signals
// and cancel a context when two signals of the same type are received.
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
