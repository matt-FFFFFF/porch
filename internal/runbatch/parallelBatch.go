// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"slices"
	"sync"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/progress"
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

	// Report that this batch is starting if we have a reporter
	if b.hasProgressReporter() {
		ReportBatchStarted(b.GetProgressReporter(), b.Label, "parallel")
	}

	// Propagate reporter to child commands
	PropagateReporterToChildren(b.GetProgressReporter(), b.Label, b.Commands)

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
		Cwd:      b.GetCwd(),
		Type:     b.GetType(),
	}}
	if children.HasError() {
		res[0].ExitCode = -1
		res[0].Error = ErrResultChildrenHasError
		res[0].Status = ResultStatusError
	}

	// Report completion based on results if we have a reporter
	if b.hasProgressReporter() {
		ReportExecutionComplete(ctx, b.GetProgressReporter(), b.Label, res,
			"Parallel batch completed successfully",
			"Parallel batch failed")
	}

	return res
}

// SetProgressReporter sets the progress reporter and propagates it to all child commands.
func (b *ParallelBatch) SetProgressReporter(reporter progress.Reporter) {
	b.BaseCommand.SetProgressReporter(reporter)
	// Note: We don't propagate here as it's done in Run() with a child reporter
}

// GetType returns the type of the runnable (e.g., "Command", "SerialBatch", "ParallelBatch", etc.).
func (b *ParallelBatch) GetType() string {
	return "ParallelBatch"
}
