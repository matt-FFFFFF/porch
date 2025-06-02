// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package run

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/matt-FFFFFF/porch/internal/config"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
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
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "write",
			Aliases:     []string{"w"},
			DefaultText: "Write results to a file",
			TakesFile:   true,
			Usage:       "The file to write results to",
			OnlyOnce:    true,
			Required:    false,
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
		outputFile := cmd.String("write")
		res := rb.Run(ctx)
		if outputFile != "" {
			f, err := os.Create(outputFile) // Create the output file if it doesn't exist
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer f.Close() //nolint:errcheck
			if err := res.WriteBinary(f); err != nil {
				return fmt.Errorf("failed to write results to file: %w", err)
			}
			return nil
		}
		opts := runbatch.DefaultOutputOptions()
		opts.IncludeStdOut = true
		opts.ShowSuccessDetails = true
		if err := res.WriteTextWithOptions(cmd.Writer, opts); err != nil {
			return fmt.Errorf("failed to write results: %w", err)
		}
		return nil
	},
}
