// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"slices"
	"sync"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
)

var _ Runnable = (*ParallelBatch)(nil)

// ParallelBatch represents a collection of commands, which can be run in parallel.
type ParallelBatch struct {
	*BaseCommand
	Commands []Runnable // The commands or nested batches to run
}

// Run implements the Runnable interface for ParallelBatch.
func (b *ParallelBatch) Run(ctx context.Context) Results {
	label := FullLabel(b)
	logger := ctxlog.Logger(ctx).
		With("label", label).
		With("runnableType", "ParallelBatch")

	children := make(Results, 0, len(b.Commands))
	wg := &sync.WaitGroup{}
	resChan := make(chan Results, len(b.Commands))

	for _, cmd := range b.Commands {
		cmd.InheritEnv(b.Env)

		logger.Debug("setting environment for child commands",
			"commandLabel", cmd.GetLabel(),
			"env", b.Env)
	}

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
		Children: children,
		Status:   ResultStatusSuccess,
	}}
	if children.HasError() {
		res[0].ExitCode = -1
		res[0].Error = ErrResultChildrenHasError
		res[0].Status = ResultStatusError
	}

	return res
}

// SetCwd sets the current working directory for the batch and all its sub-commands.
func (b *ParallelBatch) SetCwd(cwd string) error {
	if err := b.BaseCommand.SetCwd(cwd); err != nil {
		return err //nolint:err113,wrapcheck
	}

	for _, cmd := range b.Commands {
		if err := cmd.SetCwd(cwd); err != nil {
			return err //nolint:err113,wrapcheck
		}
	}

	return nil
}
