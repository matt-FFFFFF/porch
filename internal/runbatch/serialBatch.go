// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"slices"
)

var _ Runnable = (*SerialBatch)(nil)

// SerialBatch represents a collection of commands, which are run serially.
type SerialBatch struct {
	*BaseCommand
	Commands []Runnable // The commands or nested batches to run
}

// Run implements the Runnable interface for SerialBatch.
func (b *SerialBatch) Run(ctx context.Context) Results {
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
			cmd.InheritEnv(b.Env)
			if err := cmd.SetCwd(b.Cwd); err != nil {
				results = append(results, &Result{
					Label:  cmd.GetLabel(),
					Status: ResultStatusError,
					Error:  err,
				})
				continue OuterLoop
			}

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
