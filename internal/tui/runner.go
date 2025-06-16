// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package tui

import (
	"context"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matt-FFFFFF/porch/internal/progress"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

// Runner manages the TUI application and progress event integration.
type Runner struct {
	model    *Model
	program  *tea.Program
	reporter *Reporter
	mutex    sync.Mutex
}

// Reporter implements ProgressReporter and forwards events to the TUI.
type Reporter struct {
	program *tea.Program
	closed  bool
	mutex   sync.RWMutex
}

// NewTUIReporter creates a new TUI progress reporter.
func NewTUIReporter(program *tea.Program) *Reporter {
	return &Reporter{
		program: program,
	}
}

// Report implements ProgressReporter.Report.
func (tr *Reporter) Report(event progress.Event) {
	tr.mutex.RLock()
	defer tr.mutex.RUnlock()

	if tr.closed || tr.program == nil {
		return
	}

	// Send the event to the TUI program
	tr.program.Send(ProgressEventMsg{Event: event})
}

// Close implements ProgressReporter.Close.
func (tr *Reporter) Close() {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()
	tr.closed = true
}

// NewRunner creates a new TUI runner.
func NewRunner(ctx context.Context) *Runner {
	model := NewModel(ctx)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithoutSignalHandler())
	reporter := NewTUIReporter(program)

	model.SetReporter(reporter)

	return &Runner{
		model:    model,
		program:  program,
		reporter: reporter,
	}
}

// GetReporter returns the progress reporter for this TUI runner.
func (r *Runner) GetReporter() progress.Reporter {
	return r.reporter
}

// Run starts the TUI and executes the given runnable with progress reporting.
func (r *Runner) Run(ctx context.Context, runnable runbatch.Runnable) (runbatch.Results, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Channel to receive results from the command execution
	resultChan := make(chan runbatch.Results, 1)

	var cmdError error

	// Start the command execution in a goroutine
	go func() {
		defer close(resultChan)

		// If the runnable implements ProgressiveRunnable, use that method
		// Otherwise, fall back to the regular Run method.
		switch runnable := runnable.(type) {
		case runbatch.ProgressiveRunnable:
			result := runnable.RunWithProgress(ctx, r.reporter)
			resultChan <- result
		default:
			result := runnable.Run(ctx)
			resultChan <- result
		}
	}()

	// Start the TUI program in a goroutine
	tuiDone := make(chan error, 1)

	go func() {
		_, err := r.program.Run()
		tuiDone <- err
	}()

	// Wait for either the command to complete or context cancellation
	var result runbatch.Results
	select {
	case result = <-resultChan:
		// Command completed, notify TUI but don't quit yet
		r.program.Send(CommandCompletedMsg{Results: result})

		// Wait for user to manually exit the TUI
		err := <-tuiDone
		cmdError = err

		r.reporter.Close()

	case err := <-tuiDone:
		// TUI exited (user pressed 'q' or error occurred)
		cmdError = err

		r.reporter.Close()

		// Wait for command to complete or timeout
		select {
		case result = <-resultChan:
			// Command completed normally
		case <-ctx.Done():
			// Context cancelled, return what we have
			result = runbatch.Results{&runbatch.Result{
				Error: ctx.Err(),
			}}
		}

	case <-ctx.Done():
		// Context cancelled
		r.reporter.Close()
		r.program.Quit()

		select {
		case result = <-resultChan:
			// Command finished just as context was cancelled
		default:
			// Return cancellation error
			result = runbatch.Results{&runbatch.Result{
				Error: ctx.Err(),
			}}
		}

		<-tuiDone // Wait for TUI cleanup
	}

	return result, cmdError
}

// RunWithoutTUI runs a command with progress reporting but without the TUI.
// This is useful for headless environments or when TUI is not desired.
func RunWithoutTUI(
	ctx context.Context, runnable runbatch.Runnable, reporter progress.Reporter,
) runbatch.Results {
	// Check if the runnable supports progress reporting
	if progressive, ok := runnable.(runbatch.ProgressiveRunnable); ok {
		return progressive.RunWithProgress(ctx, reporter)
	}

	// Fallback to regular execution
	return runnable.Run(ctx)
}
