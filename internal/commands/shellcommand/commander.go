// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package shellcommand

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
func (c *Commander) Create(ctx context.Context, payload []byte) (runbatch.Runnable, error) {
	def := new(definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	base, err := def.ToBaseCommand()
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate("shellcommand"), err)
	}

	return New(ctx, base, def.CommandLine, def.SuccessExitCodes, def.SkipExitCodes)
}
