// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import (
	"context"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/matt-FFFFFF/porch/internal/commandregistry"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/foreachproviders"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
)

var _ commands.Commander = (*Commander)(nil)

type Commander struct{}

func (c *Commander) Create(ctx context.Context, payload []byte) (runbatch.Runnable, error) {
	def := new(definition)
	if err := yaml.Unmarshal(payload, def); err != nil {
		return nil, errors.Join(commands.ErrYamlUnmarshal, err)
	}

	var runnables []runbatch.Runnable

	base, err := def.ToBaseCommand()
	if err != nil {
		return nil, errors.Join(commands.NewErrCommandCreate("foreachdirectory"), err)
	}

	mode, err := runbatch.ParseForEachMode(def.Mode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse foreach mode: %q %w", def.Mode, err)
	}

	if def.WorkingDirectoryStrategy == "" {
		def.WorkingDirectoryStrategy = runbatch.CwdStrategyNone.String()
	}

	strat, err := runbatch.ParseCwdStrategy(def.WorkingDirectoryStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to parse working directory strategy: %q %w", def.WorkingDirectoryStrategy, err)
	}

	forEachCommand := &runbatch.ForEachCommand{
		BaseCommand:   base,
		ItemsProvider: foreachproviders.ListDirectoriesDepth(def.Depth, foreachproviders.IncludeHidden(def.IncludeHidden)),
		Mode:          mode,
		CwdStrategy:   strat,
	}

	for i, cmd := range def.Commands {
		cmdYAML, err := yaml.Marshal(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal command %d: %w", i, err)
		}

		runnable, err := commandregistry.CreateRunnableFromYAML(ctx, cmdYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to create runnable for command %d: %w", i, err)
		}

		runnable.SetParent(forEachCommand)

		runnables = append(runnables, runnable)
	}

	forEachCommand.Commands = runnables

	return forEachCommand, nil
}
