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

// Close implements ProgressReporter by delegating to the parent.
func (cr *ChildReporter) Close() {
	// Don't close the parent reporter as it might be used by other children
}
