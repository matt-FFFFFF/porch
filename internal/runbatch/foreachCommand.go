// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"errors"
	"fmt"
	"maps"
)

var _ Runnable = (*ForEachCommand)(nil)

const (
	// ItemEnvVar is the environment variable name used to store the current item in the iteration.
	ItemEnvVar = "ITEM"
)

// ItemsProviderFunc is a function that returns a list of items to iterate over.
// It takes a context and the current working directory, and returns a list of items and an error.
type ItemsProviderFunc func(ctx context.Context, workingDirectory string) ([]string, error)

// ErrItemsProviderFailed is returned when the items provider function fails.
var ErrItemsProviderFailed = errors.New("items provider function failed")

// ForEachMode determines whether the commands are executed in serial or parallel.
type ForEachMode int

const (
	// ForEachSerial executes commands in series for each item.
	ForEachSerial ForEachMode = iota
	// ForEachParallel executes commands in parallel for each item.
	ForEachParallel
)

// ForEachCommand executes a list of commands for each item returned by an items provider function.
type ForEachCommand struct {
	Label         string
	Cwd           string
	ItemsProvider ItemsProviderFunc
	Commands      []Runnable
	Mode          ForEachMode
	Env           map[string]string
}

// SetCwd sets the working directory for the command.
func (f *ForEachCommand) SetCwd(cwd string) {
	f.Cwd = cwd
	// Also set the working directory for all child commands
	for _, cmd := range f.Commands {
		cmd.SetCwd(cwd)
	}
}

// InheritEnv sets the environment variables for the batch.
func (f *ForEachCommand) InheritEnv(env map[string]string) {
	if len(f.Env) == 0 {
		f.Env = maps.Clone(env)
		return
	}

	for k, v := range maps.All(env) {
		if _, ok := f.Env[k]; !ok {
			f.Env[k] = v
		}
	}
}

// Run implements the Runnable interface for ForEachCommand.
func (f *ForEachCommand) Run(ctx context.Context) Results {
	result := &Result{
		Label:    f.Label,
		ExitCode: 0,
		Children: Results{},
	}

	// Get the items to iterate over
	items, err := f.ItemsProvider(ctx, f.Cwd)
	if err != nil {
		return Results{{
			Label:    f.Label,
			ExitCode: -1,
			Error:    fmt.Errorf("%w: %v", ErrItemsProviderFailed, err),
		}}
	}

	// Handle empty list
	if len(items) == 0 {
		// Not an error, just an empty list - return success
		return Results{result}
	}

	// the child command of a foreach must be a single batch, or a single command
	foreachCommands := make([]Runnable, len(items))

	for i, item := range items {
		// Clone the current environment for each item
		// and set the ITEM environment variable to the current item.
		newEnv := maps.Clone(f.Env)
		if newEnv == nil {
			newEnv = make(map[string]string)
		}

		newEnv[ItemEnvVar] = item

		foreachCommands[i] = &SerialBatch{
			Label:    fmt.Sprintf("%s: %s", f.Label, item),
			Commands: f.Commands,
			Env:      newEnv,
		}
	}

	// This is the main runnable that will be executed.
	// We use an interface type here to allow for different implementations (e.g., ParallelBatch).
	var run Runnable

	// Handle different execution modes
	if f.Mode == ForEachParallel {
		run = &ParallelBatch{
			Label:    fmt.Sprintf("%s (parallel results)", f.Label),
			Commands: foreachCommands,
			Env:      maps.Clone(f.Env),
		}
	}

	if f.Mode == ForEachSerial {
		run = &SerialBatch{
			Label:    fmt.Sprintf("%s (serial results)", f.Label),
			Commands: foreachCommands,
			Env:      maps.Clone(f.Env),
		}
	}

	results := run.Run(ctx)
	result.Children = results

	// If any child has an error, set the error on the parent
	if results.HasError() {
		result.Error = ErrResultChildrenHasError
		result.ExitCode = -1
	}

	return Results{result}
}

// NewForEachCommand creates a new ForEachCommand.
func NewForEachCommand(label string, provider ItemsProviderFunc, mode ForEachMode, commands []Runnable) *ForEachCommand {
	return &ForEachCommand{
		Label:         label,
		ItemsProvider: provider,
		Commands:      commands,
		Mode:          mode,
	}
}
