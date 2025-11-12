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

// NewProgressive creates a new shell command. This is now just an alias for New()
// since all commands support optional progress reporting via dependency injection.
// Kept for backward compatibility.
func NewProgressive(
	ctx context.Context,
	base *runbatch.BaseCommand,
	command string,
	successExitCodes []int,
	skipExitCodes []int) (*runbatch.OSCommand, error,
) {
	// Just delegate to the regular New function - no difference anymore
	return New(ctx, base, command, successExitCodes, skipExitCodes)
}

// ProgressiveCommander extends Commander with progress reporting capabilities.
// This is now just a wrapper around Commander since all commands support progress reporting.
// Kept for backward compatibility.
type ProgressiveCommander struct {
	*Commander
}

// NewProgressiveCommander creates a new progressive shell command commander.
func NewProgressiveCommander() *ProgressiveCommander {
	return &ProgressiveCommander{
		Commander: NewCommander(),
	}
}

// CreateProgressive creates a new runnable command. Since all commands now support
// progress reporting via SetProgressReporter(), this just returns a regular Runnable.
func (pc *ProgressiveCommander) CreateProgressive(
	ctx context.Context,
	_ commands.CommanderFactory,
	payload []byte,
	parent runbatch.Runnable,
) (runbatch.Runnable, error) {
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
