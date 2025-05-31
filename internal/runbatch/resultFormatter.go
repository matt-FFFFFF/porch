// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/matt-FFFFFF/pporch/internal/color"
)

// OutputOptions controls what is included in the output.
type OutputOptions struct {
	IncludeStdOut      bool // Whether to include stdout in the output
	IncludeStdErr      bool // Whether to include stderr in the output
	ColorOutput        bool // Whether to use color in the output
	ShowSuccessDetails bool // Whether to show details for successful commands
}

// DefaultOutputOptions returns a default set of output options.
func DefaultOutputOptions() *OutputOptions {
	return &OutputOptions{
		IncludeStdOut:      false,
		IncludeStdErr:      true,
		ColorOutput:        color.Enabled(),
		ShowSuccessDetails: false,
	}
}

// writeTextResults formatted results to the provided writer.
func writeTextResults(w io.Writer, results Results, options *OutputOptions) error {
	if options == nil {
		options = DefaultOutputOptions()
	}

	// Process each top-level result
	for _, r := range results {
		if err := writeResultWithIndent(w, r, "", options); err != nil {
			return err
		}
	}

	return nil
}

func writeResultWithIndent(w io.Writer, r *Result, indent string, options *OutputOptions) error {
	// Format the status indicator
	var statusStr, labelPrefix string

	isError := r.Error != nil || r.ExitCode != 0

	switch {
	case options.ColorOutput && isError:
		statusStr = color.Colorize("✗", color.FgRed)               // Red X
		labelPrefix = color.ControlString(color.Bold, color.FgRed) // Bold red
	case options.ColorOutput && !isError:
		statusStr = color.Colorize("✓", color.FgGreen)               // Green checkmark
		labelPrefix = color.ControlString(color.Bold, color.FgGreen) // Bold green
	case !options.ColorOutput && isError:
		statusStr = "✗" // Plain X
	case !options.ColorOutput && !isError:
		statusStr = "✓" // Plain checkmark
	}

	// Format the label
	label := r.Label
	if label == "" {
		label = "[unnamed]"
	}

	// Print the status line
	if options.ColorOutput {
		fmt.Fprintf( // nolint:errcheck
			w,
			"%s%s %s%s%s",
			indent,
			statusStr,
			labelPrefix,
			label,
			color.ControlString(color.Reset),
		)
	} else {
		fmt.Fprintf(w, "%s%s %s", indent, statusStr, label) // nolint:errcheck
	}

	// Add exit code if non-zero
	if r.ExitCode != 0 {
		fmt.Fprintf(w, " (exit code: %d)", r.ExitCode) // nolint:errcheck
	}

	fmt.Fprintln(w) // nolint:errcheck

	// Add error message if there is one
	if r.Error != nil {
		// Skip printing ErrResultChildrenHasError since it's redundant with the actual errors
		if !errors.Is(r.Error, ErrResultChildrenHasError) {
			errMsg := r.Error.Error()
			if options.ColorOutput {
				fmt.Fprintf( // nolint:errcheck
					w,
					"%s  %s %s%s\n",
					indent,
					color.ColorizeNoReset("➜ Error:", color.FgRed),
					errMsg,
					color.ControlString(color.Reset),
				)
			} else {
				fmt.Fprintf(w, "%s  ➜ Error: %s\n", indent, errMsg) // nolint:errcheck
			}
		}
	}

	// Show details only for failed commands or if explicitly asked to show success details
	shouldShowDetails := (r.Error != nil || r.ExitCode != 0 || options.ShowSuccessDetails) &&
		len(r.Children) == 0

	// Add stdout if requested and exists
	if shouldShowDetails && options.IncludeStdOut && len(r.StdOut) > 0 {
		fmt.Fprintf(w, "%s  ➜ Output:\n", indent)                    // nolint:errcheck
		fmt.Fprintf(w, "%s", formatOutput(r.StdOut, indent+"     ")) // nolint:errcheck
	}

	// Add stderr if requested and exists
	if shouldShowDetails && options.IncludeStdErr && len(r.StdErr) > 0 {
		if options.ColorOutput {
			fmt.Fprintf(w, "%s  %s\n", indent, color.Colorize("➜ Error Output:", color.FgYellow)) // nolint:errcheck
		} else {
			fmt.Fprintf(w, "%s  ➜ Error Output:\n", indent) // nolint:errcheck
		}

		fmt.Fprintf(w, "%s", formatOutput(r.StdErr, indent+"     ")) // nolint:errcheck
	}

	// Process child results if any, with increased indentation
	if len(r.Children) > 0 {
		childIndent := indent + "  "
		for _, child := range r.Children {
			if err := writeResultWithIndent(w, child, childIndent, options); err != nil {
				return err
			}
		}
	}

	return nil
}

// formatOutput formats multi-line output with proper indentation.
func formatOutput(output []byte, indent string) string {
	lines := strings.Split(string(output), "\n")
	// Add indentation to each line and join them back together
	for i, line := range lines {
		lines[i] = indent + line
	}

	return strings.Join(lines, "\n") + "\n"
}
