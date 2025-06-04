// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

// ShouldRunAction defines the action to take based on the result of a command's pre-check.
type ShouldRunAction int

const (
	// ShouldRunActionRun means run the command.
	ShouldRunActionRun ShouldRunAction = iota
	// ShouldRunActionSkip means skip the command.
	ShouldRunActionSkip
	// ShouldRunActionError means an error occurred, do not run the command.
	ShouldRunActionError
)
