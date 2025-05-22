// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package signalbroker

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/matt-FFFFFF/avmtool/internal/ctxlog"
)

func TestWatch_FirstSignalNoCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)

	sigCh := make(chan os.Signal, 1)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		Watch(ctx, sigCh, cancel)
	}()
	sigCh <- os.Interrupt

	time.Sleep(50 * time.Millisecond)
	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled after first signal")
	default:
		// ok
	}
	close(sigCh)
	wg.Wait()
}

func TestWatch_SecondSignalCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)
	sigCh := make(chan os.Signal, 2)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		Watch(ctx, sigCh, cancel)
	}()
	sigCh <- os.Interrupt
	sigCh <- os.Interrupt

	time.Sleep(50 * time.Millisecond)
	select {
	case <-ctx.Done():
		// ok
	default:
		t.Fatal("context should be cancelled after second signal")
	}
	// Channel should be closed by Watch
	_, ok := <-sigCh
	if ok {
		t.Fatal("signal channel should be closed after second signal")
	}

	wg.Wait()
}

func TestWatch_DifferentSignalsNoCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = ctxlog.New(ctx, ctxlog.DefaultLogger)
	sigCh := make(chan os.Signal, 2)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		Watch(ctx, sigCh, cancel)
	}()
	sigCh <- os.Interrupt
	sigCh <- os.Kill

	time.Sleep(50 * time.Millisecond)
	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled for different signals")
	default:
		// ok
	}
	close(sigCh)
	wg.Wait()
}
