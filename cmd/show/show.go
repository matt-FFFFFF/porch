// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package show

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"

	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/urfave/cli/v3"
)

const (
	fileArg = "file"
)

var (
	// ErrReadFile is returned when the file cannot be read.
	ErrReadFile = errors.New("failed to read file")
	// ErrDecodeResults is returned when the results cannot be decoded from the file.
	ErrDecodeResults = errors.New("failed to decode results")
	// ErrWriteResults is returned when the results cannot be written to stdout.
	ErrWriteResults          = errors.New("failed to write results to stdout")
	outputStdErrFlag         = "output-stderr"
	outputStdOutFlag         = "output-stdout"
	outputSuccessDetailsFlag = "output-success-details"
)

// ShowCmd is the command that shows the results of a batch of commands defined in a YAML file.
var ShowCmd = &cli.Command{
	Name:        "show",
	Description: "Show previously saved results.",
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name: fileArg,
		},
	},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:        outputSuccessDetailsFlag,
			Aliases:     []string{"success"},
			Usage:       "Include successful results in the output",
			TakesFile:   false,
			DefaultText: "false",
			Value:       false,
		},
		&cli.BoolWithInverseFlag{
			Name:        outputStdErrFlag,
			Aliases:     []string{"stderr"},
			Usage:       "Include stderr output in the results",
			Value:       true,
			DefaultText: "true",
			TakesFile:   false,
		},
		&cli.BoolFlag{
			Name:        outputStdOutFlag,
			Aliases:     []string{"stdout"},
			Usage:       "Include stdout output in the results",
			TakesFile:   false,
			DefaultText: "false",
			Value:       false,
		},
	},
	Action: func(_ context.Context, cmd *cli.Command) error {
		file, err := os.Open(cmd.StringArg(fileArg))
		if err != nil {
			return cli.Exit(fmt.Sprintf("%s: %v", ErrReadFile.Error(), err), 1)
		}
		defer file.Close() // nolint:errcheck
		var results runbatch.Results
		if err := gob.NewDecoder(file).Decode(&results); err != nil {
			return cli.Exit(fmt.Sprintf("%s: %v", ErrDecodeResults.Error(), err), 1)
		}
		opts := runbatch.DefaultOutputOptions()
		opts.IncludeStdErr = cmd.Bool(outputStdErrFlag)
		opts.IncludeStdOut = cmd.Bool(outputStdOutFlag)
		opts.ShowSuccessDetails = cmd.Bool(outputSuccessDetailsFlag)

		if err := results.WriteTextWithOptions(os.Stdout, opts); err != nil { // Write the results to stdout
			return cli.Exit(fmt.Sprintf("%s: %v", ErrWriteResults.Error(), err), 1)
		}
		return nil
	},
}
