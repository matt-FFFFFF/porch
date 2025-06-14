// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package progress

import (
	"context"
	"sync"
)

// ChannelReporter implements ProgressReporter using a Go channel.
// It provides a thread-safe way to send progress events to listeners.
type ChannelReporter struct {
	ch     chan ProgressEvent
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once
}

// NewChannelReporter creates a new ChannelReporter with the specified buffer size.
// A larger buffer size reduces the chance of blocking when sending events.
func NewChannelReporter(ctx context.Context, bufferSize int) *ChannelReporter {
	reporterCtx, cancel := context.WithCancel(ctx)
	return &ChannelReporter{
		ch:     make(chan ProgressEvent, bufferSize),
		ctx:    reporterCtx,
		cancel: cancel,
	}
}

// Report implements ProgressReporter.Report.
// It sends the event to the channel in a non-blocking manner.
// If the channel is full or closed, the event is dropped.
func (cr *ChannelReporter) Report(event ProgressEvent) {
	select {
	case <-cr.ctx.Done():
		// Reporter is closed, drop the event
		return
	default:
	}

	select {
	case cr.ch <- event:
		// Event sent successfully
	case <-cr.ctx.Done():
		// Reporter is closed, drop the event
	default:
		// Channel is full, drop the event to avoid blocking
	}
}

// Close implements ProgressReporter.Close.
// It closes the channel and cancels the context.
func (cr *ChannelReporter) Close() {
	cr.once.Do(func() {
		cr.cancel()
		close(cr.ch)
		cr.wg.Wait()
	})
}

// Listen starts listening for events and forwards them to the provided listener.
// This method blocks until the reporter is closed or the context is cancelled.
func (cr *ChannelReporter) Listen(listener ProgressListener) {
	cr.wg.Add(1)
	go func() {
		defer cr.wg.Done()
		for {
			select {
			case event, ok := <-cr.ch:
				if !ok {
					// Channel closed
					return
				}
				listener.OnEvent(event)
			case <-cr.ctx.Done():
				// Context cancelled
				return
			}
		}
	}()
}

// Events returns a read-only channel of progress events.
// Useful when you want to handle events manually instead of using a listener.
func (cr *ChannelReporter) Events() <-chan ProgressEvent {
	return cr.ch
}

// Context returns the reporter's context.
// The context is cancelled when the reporter is closed.
func (cr *ChannelReporter) Context() context.Context {
	return cr.ctx
}
