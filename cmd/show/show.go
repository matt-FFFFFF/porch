// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package show

import (
	"context"
	"encoding/gob"
	"errors"
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
	ErrWriteResults = errors.New("failed to write results to stdout")
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
	Action: func(_ context.Context, cmd *cli.Command) error {
		file, err := os.Open(cmd.StringArg(fileArg))
		if err != nil {
			return errors.Join(ErrReadFile, err)
		}
		defer file.Close() // nolint:errcheck
		var results runbatch.Results
		if err := gob.NewDecoder(file).Decode(&results); err != nil {
			return errors.Join(ErrDecodeResults, err)
		}
		if err := results.WriteText(os.Stdout); err != nil { // Write the results to stdout
			return errors.Join(ErrWriteResults, err)
		}
		return nil
	},
}
