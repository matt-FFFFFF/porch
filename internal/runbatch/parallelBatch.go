// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"slices"
	"sync"
)

var _ Runnable = (*ParallelBatch)(nil)

// ParallelBatch represents a collection of commands, which can be run in parallel.
type ParallelBatch struct {
	Commands []Runnable // The commands or nested batches to run
	Label    string     // Optional label for the batch
}

// GetLabel returns the label of the batch (to satisfy Runnable interface).
func (b *ParallelBatch) GetLabel() string {
	return b.Label
}

// SetCwd sets the working directory for the batch.
func (b *ParallelBatch) SetCwd(cwd string) {
	for _, cmd := range b.Commands {
		cmd.SetCwd(cwd)
	}
}

func (b *ParallelBatch) Run(ctx context.Context) Results {
	children := make(Results, 0, len(b.Commands))
	wg := &sync.WaitGroup{}
	resChan := make(chan Results, len(b.Commands))

	for _, cmd := range b.Commands {
		wg.Add(1)

		go func(c Runnable) {
			defer wg.Done()
			resChan <- c.Run(ctx)
		}(cmd)
	}

	wg.Wait()
	close(resChan)

	for r := range resChan {
		children = slices.Concat(children, r)
	}

	res := Results{&Result{
		Label:    b.Label,
		ExitCode: 0,
		Error:    nil,
		StdOut:   nil,
		StdErr:   nil,
		Children: children,
	}}
	if children.HasError() {
		res[0].ExitCode = -1
		res[0].Error = ErrResultChildrenHasError
	}

	return res
}
