// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package run

import (
	"context"
	"fmt"
	"os"

	"github.com/matt-FFFFFF/porch/cmd/cmdstate"
	"github.com/matt-FFFFFF/porch/internal/config"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/urfave/cli/v3"
)

const (
	fileArg                  = "file"
	outArg                   = "out"
	noOutputStdErrFlag       = "no-output-stderr"
	outputStdOutFlag         = "output-stdout"
	outputSuccessDetailsFlag = "output-success-details"
	writeFlag                = "write"
)

var (
	// ErrReadFile is returned when the file cannot be read.
	ErrReadFile = fmt.Errorf("failed to read file")
	// ErrBuildConfig is returned when the configuration cannot be built from the YAML file.
	ErrBuildConfig = fmt.Errorf("failed to build config")
)

// RunCmd is the command that runs a batch of commands defined in a YAML file.
var RunCmd = &cli.Command{
	Name: "run",
	Description: `Run a command or batch of commands defined in a YAML file.
This command executes the commands defined in the specified YAML file and outputs the results.
The YAML file should be structured according to the Porch configuration format, which allows for defining commands, their parameters, and execution flow.

To save the results to a file, specify the output file name as an argument.
`,
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:      fileArg,
			UsageText: "commands.porch.yaml",
		},
		&cli.StringArg{
			Name:      outArg,
			UsageText: " [output]",
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
		&cli.BoolFlag{
			Name:        noOutputStdErrFlag,
			Aliases:     []string{"no-stderr"},
			Usage:       "Exclude stderr output in the results",
			Value:       false,
			DefaultText: "false",
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
	Action: actionFunc,
}

func actionFunc(ctx context.Context, cmd *cli.Command) error {
	yamlFileName := cmd.StringArg(fileArg)
	bytes, err := os.ReadFile(yamlFileName)

	if yamlFileName == "" {
		return cli.Exit("Please provide a YAML file to run", 1)
	}

	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to read file %s: %s", yamlFileName, err.Error()), 1)
	}

	rb, err := config.BuildFromYAML(ctx, cmdstate.Factory, bytes)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to build config from file %s: %s", yamlFileName, err.Error()), 1)
	}

	res := rb.Run(ctx)

	outFileName := cmd.StringArg(outArg)
	if outFileName != "" {
		f, err := os.Create(outFileName) // Create the output file if it doesn't exist
		if err != nil {
			return cli.Exit(fmt.Sprintf("Failed to create output file %s: %s", outFileName, err.Error()), 1)
		}

		defer f.Close() //nolint:errcheck

		if err := res.WriteBinary(f); err != nil {
			return cli.Exit(fmt.Sprintf("Failed to write results to file %s: %s", outFileName, err.Error()), 1)
		}

		return nil
	}

	opts := runbatch.DefaultOutputOptions()
	opts.IncludeStdErr = !cmd.Bool(noOutputStdErrFlag)
	opts.IncludeStdOut = cmd.Bool(outputStdOutFlag)
	opts.ShowSuccessDetails = cmd.Bool(outputSuccessDetailsFlag)

	if err := res.WriteTextWithOptions(cmd.Writer, opts); err != nil {
		return cli.Exit("Failed to write results: "+err.Error(), 1)
	}

	return nil
}
