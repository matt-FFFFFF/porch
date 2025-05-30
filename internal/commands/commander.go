// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

// Package commands provides a standard interface for creating a command registry.
package commands

import (
	"context"

	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
)

// Commander is an interface for converting commands into runnables.
type Commander interface {
	Create(ctx context.Context, payload []byte) (runbatch.Runnable, error)
}
