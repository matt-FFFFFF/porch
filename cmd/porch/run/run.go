// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package run

import (
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
	"github.com/matt-FFFFFF/porch/internal/runbatch"
	"github.com/urfave/cli/v3"
)

const (
	fileFlag                 = "file"
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
	},
	Action: actionFunc,
}

func actionFunc(ctx context.Context, cmd *cli.Command) error {
	if cmd.Int(parallelismFlag) > 0 {
		runtime.GOMAXPROCS(cmd.Int(parallelismFlag))
	}

	url := cmd.StringSlice(fileFlag)

	if len(url) == 0 {
		return cli.Exit("Please specify at least one URL for the configuration file using the --file or -f flag.", 1)
	}

	factory := ctx.Value(commands.FactoryContextKey{}).(commands.CommanderFactory)

	// Create a timeout context for configuration building
	configCtx, configCancel := context.WithTimeout(ctx, configTimeoutSeconds*time.Second)
	defer configCancel()

	runnables := make([]runbatch.Runnable, len(url))

	for i, u := range url {
		if u == "" {
			return cli.Exit(fmt.Sprintf("The URL at index %d is empty. Please provide a valid URL.", i), 1)
		}

		bytes, err := getURL(ctx, u)
		if err != nil {
			return cli.Exit(err.Error(), 1)
		}

		rb, err := config.BuildFromYAML(configCtx, factory, bytes)
		if err != nil {
			return cli.Exit(fmt.Sprintf("Failed to build config from file %s: %s", url, err.Error()), 1)
		}

		if rb == nil {
			continue
		}

		runnables[i] = rb
	}

	var topRunnable runbatch.Runnable

	switch l := len(runnables); l {
	case 0:
		return cli.Exit("No runnable commands found in the provided configuration files.", 1)
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

	res := topRunnable.Run(ctx)

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

		fmt.Fprintf(cmd.Writer, "Results written to %s\n\n", outFileName) //nolint:errcheck
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
