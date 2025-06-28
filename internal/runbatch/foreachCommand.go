// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/matt-FFFFFF/porch/internal/ctxlog"
)

var _ Runnable = (*ForEachCommand)(nil)

const (
	// ItemEnvVar is the environment variable name used to store the current item in the iteration.
	ItemEnvVar = "ITEM"
)

const (
	forEachSerialString    = "serial"
	forEachParallelString  = "parallel"
	cwdStrategyNoneStr     = "none"
	cwdStrategyRelativeStr = "item_relative"
	unknownValue           = "unknown"
)

// ItemsProviderFunc is a function that returns a list of items to iterate over.
// It takes a context and the current working directory,
// and returns a list of items and an error.
type ItemsProviderFunc func(ctx context.Context, workingDirectory string) ([]string, error)

var (
	// ErrItemsProviderFailed is returned when the items provider function fails.
	ErrItemsProviderFailed = errors.New("items provider function failed")
	// ErrInvalidForEachMode is returned when an invalid foreach mode is specified.
	ErrInvalidForEachMode = errors.New("invalid foreach mode specified, must be 'serial' or 'parallel'")
	// ErrInvalidCwdStrategy is returned when an invalid cwd strategy is specified.
	ErrInvalidCwdStrategy = errors.New(
		"invalid cwd strategy specified, must be 'none', 'item_relative', or 'item_absolute'",
	)
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
		return forEachSerialString
	case ForEachParallel:
		return forEachParallelString
	default:
		return unknownValue
	}
}

// ForEachCwdStrategy determines how the current working directory (cwd) is modified for each item.
type ForEachCwdStrategy int

const (
	// CwdStrategyNone means no cwd modification.
	CwdStrategyNone ForEachCwdStrategy = iota
	// CwdStrategyItemRelative modifies the cwd to be relative to the item and
	// the working directory of the foreach command.
	CwdStrategyItemRelative
)

// String implements the Stringer interface for ForEachCwdStrategy.
func (s ForEachCwdStrategy) String() string {
	switch s {
	case CwdStrategyNone:
		return cwdStrategyNoneStr
	case CwdStrategyItemRelative:
		return cwdStrategyRelativeStr
	default:
		return unknownValue
	}
}

// ParseCwdStrategy converts a string to a ForEachCwdStrategy.
func ParseCwdStrategy(strategy string) (ForEachCwdStrategy, error) {
	switch strategy {
	case cwdStrategyNoneStr:
		return CwdStrategyNone, nil
	case cwdStrategyRelativeStr:
		return CwdStrategyItemRelative, nil
	default:
		return -1, ErrInvalidCwdStrategy
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
	// ItemsSkipOnErrors is a list of errors that will not cause the foreach items provider to fail.
	// Must be a list of errors that can be used with errors.Is.
	ItemsSkipOnErrors []error
}

// ParseForEachMode converts a string to a ForEachMode.
// If the string is not valid, it returns an ErrInvalidForEachMode error.
func ParseForEachMode(mode string) (ForEachMode, error) {
	switch mode {
	case forEachSerialString:
		return ForEachSerial, nil
	case forEachParallelString:
		return ForEachParallel, nil
	default:
		return -1, ErrInvalidForEachMode
	}
}

// Run implements the Runnable interface for ForEachCommand.
func (f *ForEachCommand) Run(ctx context.Context) Results {
	label := FullLabel(f)
	logger := ctxlog.Logger(ctx).
		With("label", label).
		With("runnableType", "ForEachCommand")

	result := &Result{
		Label:    f.Label,
		ExitCode: 0,
		Children: Results{},
		Status:   ResultStatusSuccess,
	}

	// Get the items to iterate over
	items, err := f.ItemsProvider(ctx, f.Cwd)
	if err != nil {
		for _, skipErr := range f.ItemsSkipOnErrors {
			// If the error is in the skip list, treat it as a skipped result.
			if errors.Is(err, skipErr) {
				result.Status = ResultStatusSkipped
				return Results{result}
			}
		}

		// If the error is not in the skip list, return an error result.
		return Results{{
			Label:    f.Label,
			ExitCode: -1,
			Error:    fmt.Errorf("%w: %v", ErrItemsProviderFailed, err),
			Status:   ResultStatusError,
		}}
	}

	logger.Debug("items to iterate over",
		"count", len(items),
		"items", items)

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
			fmt.Sprintf("[%s]", item),
			f.Cwd,
			f.CwdRel,
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

		switch f.CwdStrategy {
		case CwdStrategyItemRelative:
			if err := serialBatch.SetCwd(item); err != nil {
				return Results{{
					Label:    f.Label,
					ExitCode: -1,
					Error:    fmt.Errorf("%w: %v", ErrSetCwd, err),
					Status:   ResultStatusError,
				}}
			}
		}

		foreachCommands[i] = serialBatch
	}

	base := NewBaseCommand(
		f.Label, f.Cwd, f.CwdRel, f.RunsOnCondition, f.RunsOnExitCodes, maps.Clone(f.Env),
	)
	base.SetParent(f.GetParent())

	switch f.Mode {
	case ForEachParallel:
		base.Label = f.Label + " (parallel)"
		run = &ParallelBatch{
			BaseCommand: base,
			Commands:    foreachCommands,
		}
	case ForEachSerial:
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

// SetCwd sets the current working directory for the batch and all its sub-commands.
func (f *ForEachCommand) SetCwd(cwd string) error {
	if err := f.BaseCommand.SetCwd(cwd); err != nil {
		return err //nolint:err113,wrapcheck
	}

	for _, cmd := range f.Commands {
		if err := cmd.SetCwd(cwd); err != nil {
			return err //nolint:err113,wrapcheck
		}
	}

	return nil
}
