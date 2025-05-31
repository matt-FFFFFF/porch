// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/pporch/internal/commands"
	"github.com/matt-FFFFFF/pporch/internal/runbatch"
)

var _ commands.Commander = (*Commander)(nil)

// Commander is a struct that implements the commands.Commander interface.
type Commander struct{}

// Create creates a new runnable command and implements the commands.Commander interface.
func (c *Commander) Create(_ context.Context, payload []byte) (runbatch.Runnable, error) {
	def := new(Definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, fmt.Errorf("failed to unmarshal copycwdtotemp command definition: %w", err)
	}

	if def.WorkingDirectory == "" {
		def.WorkingDirectory = "."
	}

	return New(def.WorkingDirectory), nil
}
