package run

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/matt-FFFFFF/avmtool/internal/config"
	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
	"github.com/urfave/cli/v3"
)

const (
	fileArg = "file"
)

var (
	ErrReadFile    = fmt.Errorf("failed to read file")
	ErrBuildConfig = fmt.Errorf("failed to build config")
)

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
		rb, err := config.BuildFromYAML(bytes)
		if err != nil {
			return errors.Join(ErrBuildConfig, err)
		}
		res := rb.Run(ctx)
		opts := runbatch.DefaultOutputOptions()
		opts.IncludeStdOut = true
		runbatch.WriteResults(cmd.Writer, res, opts)
		return nil
	},
}
