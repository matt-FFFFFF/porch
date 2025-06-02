// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package copycwdtotemp

import (
	"context"
	"errors"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

var _ commands.Commander = (*Commander)(nil)

// Commander is a struct that implements the commands.Commander interface.
type Commander struct{}

// Create creates a new runnable command and implements the commands.Commander interface.
func (c *Commander) Create(_ context.Context, payload []byte) (runbatch.Runnable, error) {
	def := new(definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	if def.WorkingDirectory == "" {
		def.WorkingDirectory = "."
	}

	base, err := def.ToBaseCommand()
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate("copycwdtotemp"), err)
	}

	return New(base), nil
}
