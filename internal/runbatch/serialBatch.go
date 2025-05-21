package runbatch

import (
	"context"
	"os"
	"slices"
)

var _ Runnable = (*SerialBatch)(nil)

// SerialBatch represents a collection of commands, which are run serially.
type SerialBatch struct {
	Commands []Runnable // The commands or nested batches to run
	Label    string     // Optional label for the batch
}

// GetLabel returns the label of the batch (to satisfy Runnable interface)
func (b *SerialBatch) GetLabel() string {
	return b.Label
}

// SetCwd sets the working directory for the batch
func (b *SerialBatch) SetCwd(cwd string) {
	for _, cmd := range b.Commands {
		cmd.SetCwd(cwd)
	}
}

func (b *SerialBatch) Run(ctx context.Context, sig <-chan os.Signal) Results {
	children := make(Results, 0, len(b.Commands))
	newCwd := ""
	for i, cmd := range slices.All(b.Commands) {
		childResults := cmd.Run(ctx, sig)
		if len(childResults) != 1 {
			newCwd = ""
		}
		if len(childResults) == 1 && !childResults.HasError() {
			newCwd = childResults[0].newCwd
		}
		if newCwd != "" && i < len(b.Commands)-1 {
			// set the newCwd for the remaining commands
			for rb := range slices.Values(b.Commands[i+1:]) {
				rb.SetCwd(newCwd)
			}
		}
		children = slices.Concat(children, childResults)
	}
	res := Results{&Result{
		Label:    b.Label,
		ExitCode: 0,
		Error:    nil,
		StdOut:   nil,
		StdErr:   nil,
		Children: children,
	}}
	if children.HasError() {
		res[0].ExitCode = -1
		res[0].Error = ErrResultChildrenHasError
	}
	return res
}
