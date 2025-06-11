// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package run

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/hashicorp/go-getter/v2"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/config"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/urfave/cli/v3"
)

const (
	getterArg                = "url"
	outFlag                  = "out"
	noOutputStdErrFlag       = "no-output-stderr"
	outputStdOutFlag         = "output-stdout"
	outputSuccessDetailsFlag = "output-success-details"
	parallelismFlag          = "parallelism"
	writeFlag                = "write"
	configTimeoutSeconds     = 30
)

var (
	// ErrGetConfigFile is returned when the file cannot be read.
	ErrGetConfigFile = fmt.Errorf("failed to get config file")
	// ErrBuildConfig is returned when the configuration cannot be built from the YAML file.
	ErrBuildConfig = fmt.Errorf("failed to build config")
)

// RunCmd is the command that runs a batch of commands defined in a YAML file.
var RunCmd = &cli.Command{
	Name: "run",
	Description: `Run a command or batch of commands defined in a YAML file.
This command executes the commands defined in the specified YAML file and outputs the results.
The YAML file should be structured according to the Porch configuration format,
which allows for defining commands, their parameters, and execution flow.

Config file URLs use Hashicorp's go-getter syntax, which allows for fetching files from various sources.
See https://github.com/hashicorp/go-getter.

To save the results to a file, specify the output file name as an argument.
`,
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:      getterArg,
			UsageText: "e.g. git::https://github.com/matt-FFFFFF/porch//examples/avm-test-examples.yaml, or ./path/to/local/file.yaml",
		},
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:      outFlag,
			Usage:     "Specify the output file name",
			TakesFile: true,
			Value:     "",
		},
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
		&cli.IntFlag{
			Name:    parallelismFlag,
			Aliases: []string{"p"},
			Usage: "Set the maximum number of concurrent commands to run. " +
				"Defaults to the number of CPU cores available.",
			Value: 0,
		},
	},
	Action: actionFunc,
}

func actionFunc(ctx context.Context, cmd *cli.Command) error {
	if cmd.Int(parallelismFlag) > 0 {
		runtime.GOMAXPROCS(cmd.Int(parallelismFlag))
	}

	url := cmd.StringArg(getterArg)
	bytes, err := getUrl(ctx, url)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to get config file from %s: %s", url, err.Error()), 1)
	}

	factory := ctx.Value(commands.FactoryContextKey{}).(commands.CommanderFactory)

	// Create a timeout context for configuration building
	configCtx, configCancel := context.WithTimeout(ctx, configTimeoutSeconds*time.Second)
	defer configCancel()

	rb, err := config.BuildFromYAML(configCtx, factory, bytes)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to build config from file %s: %s", url, err.Error()), 1)
	}

	res := rb.Run(ctx)

	fmt.Fprint(cmd.Writer, "\n\n") //nolint:errcheck

	outFileName := cmd.String(outFlag)
	if outFileName != "" {
		f, err := os.Create(outFileName) // Create the output file if it doesn't exist
		if err != nil {
			return cli.Exit(fmt.Sprintf("Failed to create output file %s: %s", outFileName, err.Error()), 1)
		}

		defer f.Close() //nolint:errcheck

		if err := res.WriteBinary(f); err != nil {
			return cli.Exit(fmt.Sprintf("Failed to write results to file %s: %s", outFileName, err.Error()), 1)
		}
		fmt.Fprintf(cmd.Writer, "Results written to %s\n\n", outFileName)
	}

	opts := runbatch.DefaultOutputOptions()
	opts.IncludeStdErr = !cmd.Bool(noOutputStdErrFlag)
	opts.IncludeStdOut = cmd.Bool(outputStdOutFlag)
	opts.ShowSuccessDetails = cmd.Bool(outputSuccessDetailsFlag)

	if err := res.WriteTextWithOptions(cmd.Writer, opts); err != nil {
		return cli.Exit("Failed to write results: "+err.Error(), 1)
	}
	if res.HasError() {
		return cli.Exit("Some commands failed. See above for details.", 1)
	}
	return nil
}

// getUrl retrieves the content from the specified URL using Hashicorp's go-getter.
// It removes the temporary file after reading its content.
func getUrl(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, ErrGetConfigFile
	}

	tmpFile, err := os.CreateTemp("", "porch-getter-*.yml")
	if err != nil {
		return nil, errors.Join(err, ErrGetConfigFile)
	}
	dst := tmpFile.Name()
	defer os.Remove(tmpFile.Name()) //nolint:errcheck
	tmpFile.Close()                 //nolint:errcheck

	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.Join(err, ErrGetConfigFile)
	}

	cli := getter.Client{
		DisableSymlinks: true,
	}

	req := &getter.Request{
		Src:     url,
		Dst:     dst,
		Pwd:     wd,
		GetMode: getter.ModeFile,
	}

	_, err = cli.Get(ctx, req)
	if err != nil {
		return nil, errors.Join(err, ErrGetConfigFile)
	}

	f, err := os.Open(tmpFile.Name()) //nolint:errcheck
	if err != nil {
		return nil, errors.Join(err, ErrGetConfigFile)
	}
	defer f.Close() //nolint:errcheck

	bytes, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, errors.Join(err, ErrGetConfigFile)
	}

	return bytes, nil
}
