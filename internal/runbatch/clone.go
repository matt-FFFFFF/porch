package runbatch

import (
	"maps"
	"slices"
)

// cloneRunnable creates a deep copy of a Runnable to avoid shared state between foreach iterations
func cloneRunnable(r Runnable) Runnable {
	switch cmd := r.(type) {
	case *OSCommand:
		return &OSCommand{
			BaseCommand: cloneBaseCommand(cmd.BaseCommand),
			Args:        slices.Clone(cmd.Args),
			Path:        cmd.Path,
			// sigCh is left nil - it will be initialized during run if needed
		}
	case *FunctionCommand:
		return &FunctionCommand{
			BaseCommand: cloneBaseCommand(cmd.BaseCommand),
			Func:        cmd.Func, // function pointers can be shared
		}
	case *SerialBatch:
		clonedCommands := make([]Runnable, len(cmd.Commands))
		for i, subCmd := range cmd.Commands {
			clonedCommands[i] = cloneRunnable(subCmd)
		}
		return &SerialBatch{
			BaseCommand: cloneBaseCommand(cmd.BaseCommand),
			Commands:    clonedCommands,
		}
	case *ParallelBatch:
		clonedCommands := make([]Runnable, len(cmd.Commands))
		for i, subCmd := range cmd.Commands {
			clonedCommands[i] = cloneRunnable(subCmd)
		}
		return &ParallelBatch{
			BaseCommand: cloneBaseCommand(cmd.BaseCommand),
			Commands:    clonedCommands,
		}
	case *ForEachCommand:
		clonedCommands := make([]Runnable, len(cmd.Commands))
		for i, subCmd := range cmd.Commands {
			clonedCommands[i] = cloneRunnable(subCmd)
		}
		return &ForEachCommand{
			BaseCommand:   cloneBaseCommand(cmd.BaseCommand),
			ItemsProvider: cmd.ItemsProvider,
			Commands:      clonedCommands,
			Mode:          cmd.Mode,
		}
	default:
		// For unknown types, return the original - this should not happen in normal usage
		return r
	}
}

// cloneBaseCommand creates a deep copy of a BaseCommand
func cloneBaseCommand(base *BaseCommand) *BaseCommand {
	return &BaseCommand{
		Label:           base.Label,
		Cwd:             base.Cwd,
		RunsOnCondition: base.RunsOnCondition,
		RunsOnExitCodes: slices.Clone(base.RunsOnExitCodes),
		Env:             maps.Clone(base.Env),
		// parent is intentionally not copied - it will be set later
	}
}
