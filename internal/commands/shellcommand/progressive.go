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

// NewProgressive creates a new progressive shell command that supports real-time progress reporting.
// It wraps the standard shell command with progress reporting capabilities.
func NewProgressive(
	ctx context.Context,
	base *runbatch.BaseCommand,
	command string,
	successExitCodes []int,
	skipExitCodes []int) (*runbatch.OSCommand, error,
) {
	// Create the underlying OSCommand using the existing function
	osCmd, err := New(ctx, base, command, successExitCodes, skipExitCodes)
	if err != nil {
		return nil, err
	}

	return osCmd, nil
}

// ProgressiveCommander extends Commander with progress reporting capabilities.
type ProgressiveCommander struct {
	*Commander
}

// NewProgressiveCommander creates a new progressive shell command commander.
func NewProgressiveCommander() *ProgressiveCommander {
	return &ProgressiveCommander{
		Commander: NewCommander(),
	}
}

// CreateProgressive creates a new progressive runnable command.
func (pc *ProgressiveCommander) CreateProgressive(
	ctx context.Context,
	_ commands.CommanderFactory,
	payload []byte,
	parent runbatch.Runnable,
) (runbatch.ProgressiveRunnable, error) {
	def := new(Definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	base, err := def.ToBaseCommand(ctx, parent)
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate(commandType), err)
	}

	return NewProgressive(ctx, base, def.CommandLine, def.SuccessExitCodes, def.SkipExitCodes)
}
