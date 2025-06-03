// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
)

var _ Runnable = (*ForEachCommand)(nil)

const (
	// ItemEnvVar is the environment variable name used to store the current item in the iteration.
	ItemEnvVar = "ITEM"
)

// ItemsProviderFunc is a function that returns a list of items to iterate over.
// It takes a context and the current working directory, and returns a list of items and an error.
type ItemsProviderFunc func(ctx context.Context, workingDirectory string) ([]string, error)

var (
	// ErrItemsProviderFailed is returned when the items provider function fails.
	ErrItemsProviderFailed = errors.New("items provider function failed")
	// ErrInvalidForEachMode is returned when an invalid foreach mode is specified.
	ErrInvalidForEachMode = errors.New("invalid foreach mode specified, must be 'serial' or 'parallel'")
)

// ForEachMode determines whether the commands are executed in serial or parallel.
type ForEachMode int

const (
	// ForEachSerial executes commands in series for each item.
	ForEachSerial ForEachMode = iota
	// ForEachParallel executes commands in parallel for each item.
	ForEachParallel
)

func (m ForEachMode) String() string {
	switch m {
	case ForEachSerial:
		return "serial"
	case ForEachParallel:
		return "parallel"
	default:
		return "unknown"
	}
}

// ForEachCwdStrategy determines how the current working directory (cwd) is modified for each item.
type ForEachCwdStrategy int

const (
	// No cwd modification
	CwdStrategyNone ForEachCwdStrategy = iota
	// CwdItemRelative modifies the cwd to be relative to the item and the working directory of the foreach command.
	CwdStrategyItemRelative
	// CwdItemAbsolute modifies the cwd to be absolute based on the item.
	CwdStrategyItemAbsolute
)

// String implements the Stringer interface for ForEachCwdStrategy.
func (s ForEachCwdStrategy) String() string {
	switch s {
	case CwdStrategyNone:
		return "none"
	case CwdStrategyItemRelative:
		return "item_relative"
	case CwdStrategyItemAbsolute:
		return "item_absolute"
	default:
		return "unknown"
	}
}

// ParseCwdStrategy converts a string to a ForEachCwdStrategy.
func ParseCwdStrategy(strategy string) (ForEachCwdStrategy, error) {
	switch strategy {
	case "none":
		return CwdStrategyNone, nil
	case "item_relative":
		return CwdStrategyItemRelative, nil
	case "item_absolute":
		return CwdStrategyItemAbsolute, nil
	default:
		return -1, fmt.Errorf("invalid cwd strategy: %s", strategy)
	}
}

// ForEachCommand executes a list of commands for each item returned by an items provider function.
type ForEachCommand struct {
	*BaseCommand
	// ItemsProvider is a function that returns a list of items to iterate over.
	ItemsProvider ItemsProviderFunc
	// Commands is the list of commands to execute for each item.
	Commands []Runnable
	// Mode determines how the commands are executed for each item.
	Mode ForEachMode
	// CwdStrategy is for modifying the current working directory for each item
	CwdStrategy ForEachCwdStrategy
}

// ParseForEachMode converts a string to a ForEachMode.
// If the string is not valid, it returns an ErrInvalidForEachMode error.
func ParseForEachMode(mode string) (ForEachMode, error) {
	switch mode {
	case "serial":
		return ForEachSerial, nil
	case "parallel":
		return ForEachParallel, nil
	default:
		return -1, ErrInvalidForEachMode
	}
}

// Run implements the Runnable interface for ForEachCommand.
func (f *ForEachCommand) Run(ctx context.Context) Results {
	result := &Result{
		Label:    f.Label,
		ExitCode: 0,
		Children: Results{},
		Status:   ResultStatusSuccess,
	}

	// Get the items to iterate over
	items, err := f.ItemsProvider(ctx, f.Cwd)
	if err != nil {
		return Results{{
			Label:    f.Label,
			ExitCode: -1,
			Error:    fmt.Errorf("%w: %v", ErrItemsProviderFailed, err),
			Status:   ResultStatusError,
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

		var cwd string
		switch f.CwdStrategy {
		case CwdStrategyItemRelative:
			cwd = filepath.Join(f.Cwd, item)
		case CwdStrategyItemAbsolute:
			cwd = item
		default:
			cwd = f.Cwd // No modification to cwd
		}

		newEnv[ItemEnvVar] = item
		base := NewBaseCommand(
			fmt.Sprintf("[%s]", item),
			cwd,
			f.RunsOnCondition,
			f.RunsOnExitCodes,
			newEnv,
		)

		// Create the serial batch for this item
		serialBatch := &SerialBatch{
			BaseCommand: base,
		}

		// Clone the commands for each iteration to avoid shared state
		clonedCommands := make([]Runnable, len(f.Commands))
		for j, cmd := range f.Commands {
			clonedCommands[j] = cloneRunnable(cmd)
			// Set the parent of each cloned command to this serial batch
			clonedCommands[j].SetParent(serialBatch)
		}

		serialBatch.Commands = clonedCommands
		foreachCommands[i] = serialBatch
	}

	base := NewBaseCommand(f.Label, f.Cwd, f.RunsOnCondition, f.RunsOnExitCodes, maps.Clone(f.Env))
	base.SetParent(f.GetParent())

	// Handle different execution modes
	if f.Mode == ForEachParallel {
		base.Label = f.Label + " (parallel)"
		run = &ParallelBatch{
			BaseCommand: base,
			Commands:    foreachCommands,
		}
	}

	if f.Mode == ForEachSerial {
		base.Label = f.Label + " (serial)"
		run = &SerialBatch{
			BaseCommand: base,
			Commands:    foreachCommands,
		}
	}

	// Now set the parent for each foreach command to the run batch
	for _, foreachCmd := range foreachCommands {
		foreachCmd.SetParent(run)
	}

	results := run.Run(ctx)

	// If any child has an error, set the error on the parent
	if results.HasError() {
		result.Error = ErrResultChildrenHasError
		result.ExitCode = -1
		result.Status = ResultStatusError
	}

	return results
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
