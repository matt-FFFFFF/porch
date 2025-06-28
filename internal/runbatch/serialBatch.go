// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"slices"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
)

var _ Runnable = (*SerialBatch)(nil)

// SerialBatch represents a collection of commands, which are run serially.
type SerialBatch struct {
	*BaseCommand
	Commands []Runnable // The commands or nested batches to run
}

// Run implements the Runnable interface for SerialBatch.
func (b *SerialBatch) Run(ctx context.Context) Results {
	label := FullLabel(b)
	logger := ctxlog.Logger(ctx).
		With("label", label).
		With("runnableType", "SerialBatch")

	results := make(Results, 0, len(b.Commands))
	newCwd := ""

	prevState := PreviousCommandStatus{
		State:    ResultStatusSuccess,
		ExitCode: 0,
		Err:      nil,
	}

OuterLoop:
	for i, cmd := range slices.All(b.Commands) {
		select {
		case <-ctx.Done():
			break OuterLoop
		default:
			// Inherit env and cwd from the batch if not already set
			logger.Debug("setting environment for child commands",
				"commandLabel", cmd.GetLabel(),
				"env", b.Env)
			cmd.InheritEnv(b.Env)

			switch cmd.ShouldRun(prevState) {
			case ShouldRunActionSkip:
				results = append(results, &Result{
					Label:  cmd.GetLabel(),
					Status: ResultStatusSkipped,
					Error:  ErrSkipIntentional,
				})
				continue OuterLoop
			case ShouldRunActionError:
				results = append(results, &Result{
					Label:  cmd.GetLabel(),
					Status: ResultStatusSkipped,
					Error:  ErrSkipOnError,
				})
				continue OuterLoop
			}

			childResults := cmd.Run(ctx)

			prevState.State = childResults[0].Status
			prevState.ExitCode = childResults[0].ExitCode
			prevState.Err = childResults[0].Error

			newCwd = childResults[0].newCwd

			if newCwd != "" && i < len(b.Commands)-1 {
				logger.Debug("newCwd is set, updating working directory for next commands",
					"newCwd", newCwd,
				)
				// set the newCwd for the remaining commands in the batch
				for rb := range slices.Values(b.Commands[i+1:]) {
					if err := rb.SetCwd(newCwd); err != nil {
						results = append(results, &Result{
							Label:  rb.GetLabel(),
							Status: ResultStatusError,
							Error:  err,
						})
						continue OuterLoop
					}
					logger.Debug("newCwd resultant working directory",
						"commandLabel", rb.GetLabel(),
						"cwd", rb.GetCwd(),
					)
				}
			}

			results = slices.Concat(results, childResults)
		}
	}

	res := Results{&Result{
		Label:    b.Label,
		ExitCode: 0,
		Error:    nil,
		StdOut:   nil,
		StdErr:   nil,
		Children: results,
		Status:   ResultStatusSuccess,
	}}
	if results.HasError() {
		res[0].ExitCode = -1
		res[0].Error = ErrResultChildrenHasError
		res[0].Status = ResultStatusError
	}

	return res
}

// SetCwd sets the current working directory for the batch and all its sub-commands.
func (b *SerialBatch) SetCwd(cwd string) error {
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
