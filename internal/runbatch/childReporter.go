// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"github.com/matt-FFFFFF/porch/internal/progress"
)

// ChildReporter creates a child progress reporter that prefixes command paths.
// This is used by progressive implementations to create nested progress reporting.
type ChildReporter struct {
	parent progress.Reporter
	prefix []string
}

// NewChildReporter creates a new child reporter with the given path prefix.
func NewChildReporter(parent progress.Reporter, prefix []string) *ChildReporter {
	return &ChildReporter{
		parent: parent,
		prefix: prefix,
	}
}

// Report implements ProgressReporter by correctly handling nested command paths.
func (cr *ChildReporter) Report(event progress.Event) {
	// If the event has a non-empty command path, it represents the relative path
	// from our context. We need to prepend our prefix to create the full path.
	if len(event.CommandPath) > 0 {
		// Create new slice to avoid modifying the original event
		fullPath := make([]string, 0, len(cr.prefix)+len(event.CommandPath))
		fullPath = append(fullPath, cr.prefix...)
		fullPath = append(fullPath, event.CommandPath...)
		event.CommandPath = fullPath
	} else {
		// If event has no path, use our prefix
		event.CommandPath = cr.prefix
	}

	cr.parent.Report(event)
}

// Close implements Reporter by delegating to the parent.
func (cr *ChildReporter) Close() {
	// Don't close the parent reporter as it might be used by other children
}

// TransparentReporter is a reporter that passes events through without modifying the command path.
// This is useful for intermediate commands that should not appear in the progress hierarchy,
// such as ForEachCommand which creates a batch internally.
type TransparentReporter struct {
	parent progress.Reporter
}

// NewTransparentReporter creates a new transparent reporter that passes events through
// to the parent without adding any command path prefixes.
func NewTransparentReporter(parent progress.Reporter) *TransparentReporter {
	return &TransparentReporter{
		parent: parent,
	}
}

// Report implements ProgressReporter by passing the event through unchanged.
func (tr *TransparentReporter) Report(event progress.Event) {
	tr.parent.Report(event)
}

// Close implements Reporter by delegating to the parent.
func (tr *TransparentReporter) Close() {
	// Don't close the parent reporter as it might be used by other children
}
