package runbatch

import (
	"context"
	"os"
)

// Runnable is an interface for something that can be run as part of a batch (either a Command or a nested Batch).
//
// Run() executes the command or batch and returns the results. It should handle context cancellation and passing signals to the spawned process.
//
// GetLabel() returns the label of the command or batch. This is used for logging and error reporting.
type Runnable interface {
	Run(context.Context, <-chan os.Signal) Results
	GetLabel() string
	SetCwd(string)
}
