// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package color provides functions to determine if color output is enabled.
// It also provides a function to colorize strings with ANSI escape codes.
// The package checks the environment variables NO_COLOR and FORCE_COLOR to determine
// if color output should be enabled or disabled. It also checks if the output is a
// terminal using the golang.org/x/term package. If the output is not a terminal,
// color output is disabled. The package provides constants for various ANSI color codes
// for foreground and background colors.
package color
