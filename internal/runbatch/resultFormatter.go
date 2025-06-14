// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/matt-FFFFFF/porch/internal/color"
)

// OutputOptions controls what is included in the output.
type OutputOptions struct {
	IncludeStdOut      bool // Whether to include stdout in the output
	IncludeStdErr      bool // Whether to include stderr in the output
	ShowSuccessDetails bool // Whether to show details for successful commands
}

// DefaultOutputOptions returns a default set of output options.
func DefaultOutputOptions() *OutputOptions {
	return &OutputOptions{
		IncludeStdOut:      false,
		IncludeStdErr:      true,
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

	switch r.Status {
	case ResultStatusSkipped:
		statusStr = color.Colorize("~", color.FgYellow)               // Yellow tilde
		labelPrefix = color.ControlString(color.Bold, color.FgYellow) // Bold yellow
	case ResultStatusError:
		statusStr = color.Colorize("✗", color.FgRed)               // Red X
		labelPrefix = color.ControlString(color.Bold, color.FgRed) // Bold red
	case ResultStatusSuccess:
		statusStr = color.Colorize("✓", color.FgGreen)               // Green checkmark
		labelPrefix = color.ControlString(color.Bold, color.FgGreen) // Bold green
	default:
		statusStr = color.Colorize("?", color.FgWhite) // White question mark for unknown status
	}

	// Format the label
	label := r.Label
	if label == "" {
		label = "[unnamed]"
	}

	// Print the status line
	fmt.Fprintf( // nolint:errcheck
		w,
		"%s%s %s%s%s",
		indent,
		statusStr,
		labelPrefix,
		label,
		color.ControlString(color.Reset),
	)

	// Add exit code if non-zero
	if r.ExitCode != 0 {
		fmt.Fprintf(w, " (exit code: %d)", r.ExitCode) // nolint:errcheck
	}

	fmt.Fprintln(w) // nolint:errcheck

	// Add error message if there is one
	if r.Error != nil {
		var errColor color.Code

		switch r.Status {
		case ResultStatusSkipped:
			errColor = color.FgYellow // Yellow for skipped
		case ResultStatusError:
			errColor = color.FgRed // Red for error
		default:
			errColor = color.FgWhite // Default to white for unknown status
		}
		// Skip printing ErrResultChildrenHasError since it's redundant with the actual errors
		if !errors.Is(r.Error, ErrResultChildrenHasError) {
			errMsg := r.Error.Error()
			fmt.Fprintf( // nolint:errcheck
				w,
				"%s  %s %s%s\n",
				indent,
				color.ColorizeNoReset("➜ Error:", errColor),
				errMsg,
				color.ControlString(color.Reset),
			)
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
		fmt.Fprintf(w, "%s  %s\n", indent, color.Colorize("➜ Error Output:", color.FgHiRed)) // nolint:errcheck
		fmt.Fprintf(w, "%s", formatOutput(r.StdErr, indent+"     "))                         // nolint:errcheck
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
	sb := strings.Builder{}
	lines := strings.Split(string(output), "\n")
	sb.Grow(len(output) + len(lines)*len(indent)) // Preallocate enough space
	// Add indentation to each line and join them back together
	for _, line := range lines {
		if line == "" {
			sb.WriteString("\n") // Preserve empty lines
			continue             // Skip empty lines
		}

		sb.WriteString(indent)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}
