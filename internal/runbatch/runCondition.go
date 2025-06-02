// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"errors"
)

// RunCondition defines when a command should run based on the result of the previous command.
// It can be set to RunOnSuccess, RunOnError, or RunOnAlways.
type RunCondition int

const (
	// RunOnSuccess means the command runs only if the previous command succeeded (exit code 0).
	RunOnSuccess RunCondition = iota
	// RunOnError means the command runs only if the previous command failed (non-zero exit code) or error occurred.
	RunOnError
	// RunOnAlways means the command always runs regardless of the previous command's result.
	RunOnAlways
	// RunOnExitCodes means the command runs only if the previous command's exit code matches one of the specified codes.
	RunOnExitCodes
)

const (
	runOnSuccessStr = "success"
	runOnErrorStr   = "error"
	runOnAlwaysStr  = "always"
	runOnExitCodes  = "exit-codes"
	runOnUnknownStr = "unknown"
)

var (
	// ErrRunConditionUnknown is returned when an unknown RunCondition value is encountered.
	ErrRunConditionUnknown = errors.New("unknown RunCondition value")
)

// String returns the string representation of the RunCondition.
func (r RunCondition) String() string {
	switch r {
	case RunOnSuccess:
		return runOnSuccessStr
	case RunOnError:
		return runOnErrorStr
	case RunOnAlways:
		return runOnAlwaysStr
	case RunOnExitCodes:
		return runOnExitCodes
	default:
		return runOnUnknownStr
	}
}

// NewRunCondition creates a RunCondition from a string.
func NewRunCondition(s string) (RunCondition, error) {
	switch s {
	case runOnSuccessStr:
		return RunOnSuccess, nil
	case runOnErrorStr:
		return RunOnError, nil
	case runOnAlwaysStr:
		return RunOnAlways, nil
	case runOnExitCodes:
		return RunOnExitCodes, nil
	default:
		return RunCondition(-1), ErrRunConditionUnknown
	}
}
