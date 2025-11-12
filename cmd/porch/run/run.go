// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/go-getter/v2"
	"github.com/matt-FFFFFF/porch/internal/commands"
	"github.com/matt-FFFFFF/porch/internal/config"
	"github.com/matt-FFFFFF/porch/internal/ctxlog"
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/matt-FFFFFF/porch/internal/tui"
	"github.com/urfave/cli/v3"
)

const (
	fileFlag                    = "file"
	outFlag                     = "out"
	noOutputStdErrFlag          = "no-output-stderr"
	outputStdOutFlag            = "output-stdout"
	outputSuccessDetailsFlag    = "output-success-details"
	parallelismFlag             = "parallelism"
	tuiFlag                     = "tui"
	writeFlag                   = "write"
	configTimeoutFlag           = "config-timeout"
	configTimeoutSecondsDefault = 30
	cliExitStr                  = ""
	showDetails                 = "show-details"
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
	Arguments: []cli.Argument{},
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:    fileFlag,
			Aliases: []string{"f"},
			Usage: "Specify the URL of the YAML configuration file to run. " +
				"Supports Hashicorp's go-getter syntax for fetching files from various sources. " +
				"Specify multiple times to run multiple files.",
			OnlyOnce: false,
		},
		&cli.StringFlag{
			Name:      outFlag,
			Usage:     "Specify the output file name",
			TakesFile: true,
			Value:     "",
			OnlyOnce:  true,
		},
		&cli.BoolFlag{
			Name:        outputSuccessDetailsFlag,
			Aliases:     []string{"success"},
			Usage:       "Include successful results in the output",
			TakesFile:   false,
			DefaultText: "false",
			Value:       false,
			OnlyOnce:    true,
		},
		&cli.BoolFlag{
			Name:        noOutputStdErrFlag,
			Aliases:     []string{"no-stderr"},
			Usage:       "Exclude stderr output in the results",
			Value:       false,
			DefaultText: "false",
			TakesFile:   false,
			OnlyOnce:    true,
		},
		&cli.BoolFlag{
			Name:        outputStdOutFlag,
			Aliases:     []string{"stdout"},
			Usage:       "Include stdout output in the results",
			TakesFile:   false,
			DefaultText: "false",
			Value:       false,
			OnlyOnce:    true,
		},
		&cli.IntFlag{
			Name:    parallelismFlag,
			Aliases: []string{"p"},
			Usage: "Set the maximum number of concurrent commands to run. " +
				"Defaults to the number of CPU cores available.",
			Value: 0,
		},
		&cli.BoolFlag{
			Name:        tuiFlag,
			Aliases:     []string{"t", "interactive"},
			Usage:       "Run with interactive Terminal User Interface (TUI) showing real-time progress",
			Value:       false,
			DefaultText: "false",
			TakesFile:   false,
			OnlyOnce:    true,
		},
		&cli.IntFlag{
			Name:    configTimeoutFlag,
			Aliases: []string{"timeout"},
			Usage: "Set the maximum time in seconds to wait for configuration building. " +
				"Defaults to 30 seconds.",
			Value: configTimeoutSecondsDefault,
		},
		&cli.BoolFlag{
			Name:        showDetails,
			Aliases:     []string{"details"},
			Usage:       "Include the types and working directory in the output",
			Value:       false,
			DefaultText: "false",
			TakesFile:   false,
			OnlyOnce:    true,
		},
	},
	Action: actionFunc,
}

func actionFunc(ctx context.Context, cmd *cli.Command) error {
	logger := ctxlog.Logger(ctx).With("command", cmd.Name)
	logger.Debug("Running run command")

	if cmd.Int(parallelismFlag) > 0 {
		runtime.GOMAXPROCS(cmd.Int(parallelismFlag))
	}

	url := cmd.StringSlice(fileFlag)

	if len(url) == 0 {
		logger.Error("Please specify at least one URL for the configuration file using the --file or -f flag.")
		return cli.Exit(nil, 1)
	}

	factory := ctx.Value(commands.FactoryContextKey{}).(commands.CommanderFactory)

	// Create a timeout context for configuration building
	configCtx, configCancel := context.WithTimeout(ctx, time.Duration(cmd.Int(configTimeoutFlag))*time.Second)
	defer configCancel()

	runnables := make([]runbatch.Runnable, len(url))

	for i, u := range url {
		if u == "" {
			logger.Error(fmt.Sprintf("The URL at index %d is empty. Please provide a valid URL.", i))
			return cli.Exit(cliExitStr, 1)
		}

		bytes, err := getURL(ctx, u)
		if err != nil {
			return cli.Exit(err.Error(), 1)
		}

		rb, err := config.BuildFromYAML(configCtx, factory, bytes)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to build config from file %s: %s", u, err.Error()))
			return cli.Exit(cliExitStr, 1)
		}

		if rb == nil {
			continue
		}

		runnables[i] = rb
	}

	var topRunnable runbatch.Runnable

	switch l := len(runnables); l {
	case 0:
		logger.Error("No runnable commands found in the provided configuration files.")
		return cli.Exit(nil, 1)
	case 1:
		topRunnable = runnables[0]
	default:
		topRunnable = &runbatch.SerialBatch{
			BaseCommand: &runbatch.BaseCommand{
				Cwd:   ".",
				Label: "Aggregate",
			},
			Commands: runnables,
		}
	}

	// Execute with TUI or regular mode based on flag
	var res runbatch.Results

	var execErr error

	switch cmd.Bool(tuiFlag) {
	case true:
		// Run with TUI - use TUI-compatible logger that won't interfere with display
		logger.Info("Starting interactive TUI mode...")

		buf := new(bytes.Buffer)
		// Create a TUI-friendly context that suppresses log output
		tuiCtx := ctxlog.NewForTUI(ctx, buf)

		runner := tui.NewRunner(tuiCtx)

		res, execErr = runner.Run(tuiCtx, topRunnable)

		buf.WriteTo(cmd.Writer) //nolint:errcheck // Write any buffered log output to the command writer

		if execErr != nil {
			logger.Error(fmt.Sprintf("TUI execution error: %s", execErr.Error()), "error", execErr.Error())
		}
	default:
		// Run in standard mode
		res = topRunnable.Run(ctx)
	}

	outFileName := cmd.String(outFlag)
	if outFileName != "" {
		f, err := os.Create(outFileName) // Create the output file if it doesn't exist
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create output file %s: %s", outFileName, err.Error()))
			return cli.Exit(cliExitStr, 1)
		}

		defer f.Close() //nolint:errcheck

		if err := res.WriteBinary(f); err != nil {
			logger.Error(fmt.Sprintf("Failed to write results to file %s: %s", outFileName, err.Error()))
			return cli.Exit(cliExitStr, 1)
		}

		logger.Info(fmt.Sprintf("Results written to %s", outFileName))
	}

	opts := runbatch.DefaultOutputOptions()
	opts.IncludeStdErr = !cmd.Bool(noOutputStdErrFlag)
	opts.IncludeStdOut = cmd.Bool(outputStdOutFlag)
	opts.ShowSuccessDetails = cmd.Bool(outputSuccessDetailsFlag)
	opts.ShowDetals = cmd.Bool(showDetails)

	logger.Info("Displaying results...")

	if err := res.WriteTextWithOptions(cmd.Writer, opts); err != nil {
		logger.Error(fmt.Sprintf("Failed to write results: %s", err.Error()))
		return cli.Exit(nil, 1)
	}

	if res.HasError() {
		logger.Error("Some commands failed. See above for details.")
		return cli.Exit(cliExitStr, 1)
	}

	return nil
}

// getURL retrieves the content from the specified URL using Hashicorp's go-getter.
// It removes the temporary file after reading its content.
func getURL(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, ErrGetConfigFile
	}

	tmpDir, err := os.MkdirTemp("", "porch-getter-*")
	if err != nil {
		return nil, errors.Join(ErrGetConfigFile, err)
	}

	defer os.RemoveAll(tmpDir) //nolint:errcheck

	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.Join(ErrGetConfigFile, err)
	}

	cli := getter.Client{
		DisableSymlinks: true,
	}

	req := &getter.Request{
		Src:     url,
		Dst:     filepath.Join(tmpDir, "g"),
		Pwd:     wd,
		GetMode: getter.ModeDir,
	}

	var fileName string
	// If it's not a local file URL, we need to download the directory and read the file from there
	// https://github.com/hashicorp/go-getter/issues/98
	if ok, err := getter.Detect(req, &getter.FileGetter{}); !ok || err != nil {
		if err != nil {
			return nil, errors.Join(ErrGetConfigFile, err)
		}

		var newURL string

		newURL, fileName = splitFileNameFromGetterURL(url)
		if newURL == "" || fileName == "" {
			return nil, fmt.Errorf("%w: invalid URL format: %s", ErrGetConfigFile, url)
		}

		req.Src = newURL
	}

	if fileName == "" {
		req.Src = filepath.Dir(url)
		fileName = filepath.Base(url)
	}

	res, err := cli.Get(ctx, req)
	if err != nil {
		return nil, errors.Join(ErrGetConfigFile, err)
	}

	bytes, err := os.ReadFile(filepath.Join(res.Dst, fileName))
	if err != nil {
		return nil, errors.Join(ErrGetConfigFile, err)
	}

	return bytes, nil
}

const (
	goGetterPathSeparator = "//"
	goGetterRefSeparator  = "?"
	minimumGetterParts    = 3 // Minimum parts in a go-getter URL: scheme, host, and path
)

// splitFileNameFromGetterURL splits the URL into the directory and file name.
// It returns the new getter URL without the file name and the file name itself.
// It will append any ref query parameter to the new URL if it exists.
func splitFileNameFromGetterURL(url string) (string, string) {
	var ref, fileName string

	parts := strings.Split(url, goGetterPathSeparator)
	if len(parts) < minimumGetterParts {
		return "", ""
	}

	if strings.Contains(parts[len(parts)-1], goGetterRefSeparator) {
		refSplit := strings.Split(parts[len(parts)-1], goGetterRefSeparator)
		if len(refSplit) > 1 {
			ref = strings.Join(refSplit[1:], "")
		}

		parts[len(parts)-1] = refSplit[0]
	}

	if filepath.Clean(parts[len(parts)-1]) == filepath.Dir(parts[len(parts)-1]) {
		return "", ""
	}

	fileName = filepath.Base(parts[len(parts)-1])
	parts[len(parts)-1] = filepath.Dir(parts[len(parts)-1])

	if parts[len(parts)-1] == "." {
		parts = parts[:len(parts)-1] // Remove the last part which is the file name
	}

	newURL := strings.Join(parts, goGetterPathSeparator)

	if ref != "" {
		newURL += goGetterRefSeparator + ref
	}

	return newURL, fileName
}
