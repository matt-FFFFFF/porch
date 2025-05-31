// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package run

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/matt-FFFFFF/pporch/internal/config"
	"github.com/matt-FFFFFF/pporch/internal/runbatch"
	"github.com/urfave/cli/v3"
)

const (
	fileArg = "file"
)

var (
	// ErrReadFile is returned when the file cannot be read.
	ErrReadFile = fmt.Errorf("failed to read file")
	// ErrBuildConfig is returned when the configuration cannot be built from the YAML file.
	ErrBuildConfig = fmt.Errorf("failed to build config")
)

// RunCmd is the command that runs a batch of commands defined in a YAML file.
var RunCmd = &cli.Command{
	Name:        "run",
	Description: "Run a command or batch of commands defined in a YAML file.",
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name: fileArg,
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bytes, err := os.ReadFile(cmd.StringArg(fileArg))
		if err != nil {
			return errors.Join(ErrReadFile, err)
		}
		rb, err := config.BuildFromYAML(ctx, bytes)
		if err != nil {
			return errors.Join(ErrBuildConfig, err)
		}
		res := rb.Run(ctx)
		opts := runbatch.DefaultOutputOptions()
		opts.IncludeStdOut = true
		opts.ShowSuccessDetails = true
		if err := runbatch.WriteResults(cmd.Writer, res, opts); err != nil {
			return fmt.Errorf("failed to write results: %w", err)
		}
		return nil
	},
}
