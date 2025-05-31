// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package commands provides a standard interface for creating a command registry.
package commands

import (
	"context"

	"github.com/matt-FFFFFF/pporch/internal/runbatch"
)

// Commander is an interface for converting commands into runnables.
type Commander interface {
	// Create creates a runnable command from the provided payload.
	// The payload is the YAML command in bytes.
	Create(ctx context.Context, payload []byte) (runbatch.Runnable, error)
}
