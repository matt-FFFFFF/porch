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

	var state RunState

OuterLoop:
	for i, cmd := range slices.All(b.Commands) {
		select {
		case <-ctx.Done():
			break OuterLoop
		default:
			// Inherit env and cwd from the batch if not already set
			cmd.InheritEnv(b.Env)
			cmd.SetCwd(b.Cwd, false)

			if !cmd.ShouldRun(state) {
				results = append(results, &Result{
					Label:   cmd.GetLabel(),
					Skipped: true,
				})
				continue OuterLoop
			}
			childResults := cmd.Run(ctx)

			state.ExitCode = childResults[len(childResults)-1].ExitCode
			state.Err = childResults[len(childResults)-1].Error

			// Update cwd for future commands if the current command has changed it
			if len(childResults) != 1 {
				newCwd = ""
			}
			if len(childResults) == 1 && !childResults.HasError() {
				newCwd = childResults[0].newCwd
			}
			if newCwd != "" && i < len(b.Commands)-1 {
				// set the newCwd for the remaining commands in the batch
				for rb := range slices.Values(b.Commands[i+1:]) {
					rb.SetCwd(newCwd, true)
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
	}}
	if results.HasError() {
		res[0].ExitCode = -1
		res[0].Error = ErrResultChildrenHasError
	}

	return res
}
