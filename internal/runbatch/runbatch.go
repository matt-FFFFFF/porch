// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"strconv"
)

// BatchError aggregates errors from multiple commands and formats a detailed error message.
type BatchError struct {
	FailedResults Results
}

func (e *BatchError) Error() string {
	msg := "Batch execution failed:\n"
	for _, r := range e.FailedResults {
		msg += r.Label + ": " + r.Error.Error() + " (exit code: " + strconv.Itoa(r.ExitCode) + ")\n"
	}

	return msg
}
