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
	*BaseCommand
	ItemsProvider ItemsProviderFunc
	Commands      []Runnable
	Mode          ForEachMode
}

// GetLabel returns the label of the batch.
func (f *ForEachCommand) GetLabel() string {
	if f.Label == "" {
		return "ForEach Command"
	}

	return f.Label
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

	// This is the main runnable that will be executed.
	// We use an interface type here to allow for different implementations (e.g., ParallelBatch).
	var run Runnable

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
		base := NewBaseCommand(
			fmt.Sprintf("[%s]", f.Label),
			f.Cwd,
			f.RunsOnCondition,
			f.RunsOnExitCodes,
			newEnv,
		)
		foreachCommands[i] = &SerialBatch{
			BaseCommand: base,
			Commands:    f.Commands,
		}
		foreachCommands[i].SetParent(run)
	}

	base := NewBaseCommand(f.Label, f.Cwd, f.RunsOnCondition, f.RunsOnExitCodes, maps.Clone(f.Env))
	base.SetParent(f.GetParent())

	// Handle different execution modes
	if f.Mode == ForEachParallel {
		run = &ParallelBatch{
			BaseCommand: base,
			Commands:    foreachCommands,
		}
	}

	if f.Mode == ForEachSerial {
		run = &SerialBatch{
			BaseCommand: base,
			Commands:    foreachCommands,
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
func NewForEachCommand(
	base *BaseCommand,
	provider ItemsProviderFunc,
	mode ForEachMode,
	commands []Runnable) *ForEachCommand {
	return &ForEachCommand{
		BaseCommand:   base,
		ItemsProvider: provider,
		Commands:      commands,
		Mode:          mode,
	}
}
