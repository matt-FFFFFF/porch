// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"maps"
	"slices"
)

var _ Runnable = (*SerialBatch)(nil)

// SerialBatch represents a collection of commands, which are run serially.
type SerialBatch struct {
	Commands []Runnable        // The commands or nested batches to run
	Label    string            // Optional label for the batch
	Env      map[string]string // Environment variables to be passed to each command.
}

// SetCwd sets the working directory for the batch.
func (b *SerialBatch) SetCwd(cwd string) {
	for _, cmd := range b.Commands {
		cmd.SetCwd(cwd)
	}
}

// InheritEnv sets the environment variables for the batch.
func (b *SerialBatch) InheritEnv(env map[string]string) {
	if len(b.Env) == 0 {
		b.Env = make(map[string]string)
		return
	}
	for k, v := range maps.All(env) {
		if _, ok := b.Env[k]; !ok {
			b.Env[k] = v
		}
	}
}

// Run implements the Runnable interface for SerialBatch.
func (b *SerialBatch) Run(ctx context.Context) Results {
	results := make(Results, 0, len(b.Commands))
	newCwd := ""

OuterLoop:
	for i, cmd := range slices.All(b.Commands) {
		select {
		case <-ctx.Done():
			break OuterLoop
		default:
			cmd.InheritEnv(b.Env)
			childResults := cmd.Run(ctx)
			if len(childResults) != 1 {
				newCwd = ""
			}
			if len(childResults) == 1 && !childResults.HasError() {
				newCwd = childResults[0].newCwd
			}
			if newCwd != "" && i < len(b.Commands)-1 {
				// set the newCwd for the remaining commands in the batch
				for rb := range slices.Values(b.Commands[i+1:]) {
					rb.SetCwd(newCwd)
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
