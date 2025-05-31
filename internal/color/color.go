// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package color

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const (
	sbPadding = 16 // padding for the strings.Builder
)

// Code represents an ANSI control code for text formatting.
type Code int

// ControlString generates a string with ANSI control codes for text formatting.
func ControlString(c ...Code) string {
	sb := strings.Builder{}
	sb.Grow(len(prefix) + len(suffix) + sbPadding)
	sb.WriteString(prefix)

	for i, code := range c {
		if i > 0 && i < len(c) {
			sb.WriteString(";")
		}

		sb.WriteString(strconv.Itoa(int(code)))
	}

	sb.WriteString(suffix)

	return sb.String()
}

const (
	// NoColor is the environment variable that disables color output.
	NoColor = "NO_COLOR"
	// ForceColor is the environment variable that forces color output.
	ForceColor = "FORCE_COLOR"
	reset      = "\033[0m"
	prefix     = "\033["
	suffix     = "m"
)

// Control codes for text formatting.
const (
	Reset Code = iota
	Bold
	Faint
	Italic
	Underline
	BlinkSlow
	BlinkRapid
	ReverseVideo
	Concealed
	CrossedOut
)

// Foreground text colors.
const (
	FgBlack Code = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite

	// used internally for 256 and 24-bit coloring.
	foreground //nolint:unused
)

// Foreground Hi-Intensity text colors.
const (
	FgHiBlack Code = iota + 90
	FgHiRed
	FgHiGreen
	FgHiYellow
	FgHiBlue
	FgHiMagenta
	FgHiCyan
	FgHiWhite
)

// Background text colors.
const (
	BgBlack Code = iota + 40
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite

	// used internally for 256 and 24-bit coloring.
	background //nolint:unused
)

// Background Hi-Intensity text colors.
const (
	BgHiBlack Code = iota + 100
	BgHiRed
	BgHiGreen
	BgHiYellow
	BgHiBlue
	BgHiMagenta
	BgHiCyan
	BgHiWhite
)

var enabled bool

func init() {
	enabled = isColorEnabled()
}

// Colorize returns a string with ANSI color codes applied.
// It appends the reset code at the end of the string to reset the color.
func Colorize(str string, colorCodes ...Code) string {
	// If color output is not enabled, return the string as is
	if !enabled {
		return str
	}

	sb := strings.Builder{}
	sb.Grow(len(str) + len(prefix) + len(suffix) + len(reset) + sbPadding)
	sb.WriteString(prefix)

	for i, code := range colorCodes {
		if i > 0 && i < len(colorCodes) {
			sb.WriteString(";")
		}

		sb.WriteString(strconv.Itoa(int(code)))
	}

	sb.WriteString(suffix)
	sb.WriteString(str)
	sb.WriteString(reset)

	return sb.String()
}

// ColorizeNoReset returns a string with ANSI color codes applied.
// It does not append the reset code at the end of the string.
func ColorizeNoReset(str string, colorCodes ...Code) string {
	// If color output is not enabled, return the string as is
	if !enabled {
		return str
	}

	sb := strings.Builder{}
	sb.Grow(len(str) + len(prefix) + len(suffix) + sbPadding)
	sb.WriteString(prefix)

	for i, code := range colorCodes {
		if i > 0 && i < len(colorCodes) {
			sb.WriteString(";")
		}

		sb.WriteString(strconv.Itoa(int(code)))
	}

	sb.WriteString(suffix)
	sb.WriteString(str)

	return sb.String()
}

// Enabled is a function that indicates whether color output is enabled.
// It is initialized in package init().
//
// It is set to true if either the NO_COLOR environment variable is not set,
// and the FORCE_COLOR environment variable is set, or if the output is a terminal.
// Terminal detection is done using the golang.org/x/term package.
//
// It is set to false if the NO_COLOR environment variable is set, or if the
// output is not a terminal.
func Enabled() bool {
	return enabled
}

func isColorEnabled() bool {
	if nc := os.Getenv(NoColor); nc != "" {
		return false
	}

	if fc := os.Getenv(ForceColor); fc != "" {
		return true
	}

	return term.IsTerminal(int(os.Stdout.Fd()))
}
