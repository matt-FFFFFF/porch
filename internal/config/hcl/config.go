// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package hcl

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Azure/golden"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/afero"
)

const (
	porchConfigFileExt = ".porch.hcl"
)

var _ golden.Config = &PorchConfig{}

var (
	// ErrInitConfig is returned when the Porch configuration cannot be initialized.
	ErrInitConfig = errors.New("failed to initialize Porch configuration")
	// ErrNoPorchConfigFile is returned when no `.porch.hcl` file is found in the specified directory.
	ErrNoPorchConfigFile = errors.New("no `.porch.hcl` file found in the specified directory")
	// ErrParsePorchConfigFile is returned when there is an error parsing the `.porch.hcl` file.
	ErrParsePorchConfigFile = errors.New("failed to parse blocks in the configuration file")
)

// PorchConfig represents the configuration for the Porch system.
type PorchConfig struct {
	*golden.BaseConfig
}

// ErrInvalidBlockType represents an error for an invalid block type in the Porch configuration.
type ErrInvalidBlockType struct {
	BlockType string
	Range     hcl.Range
}

// NewErrInvalidBlockType creates a new ErrInvalidBlockType with the specified block type and range.
func NewErrInvalidBlockType(blockType string, r hcl.Range) *ErrInvalidBlockType {
	return &ErrInvalidBlockType{
		BlockType: blockType,
		Range:     r,
	}
}

// Error implements the error interface for ErrInvalidBlockType.
func (e *ErrInvalidBlockType) Error() string {
	return fmt.Sprintf("invalid block type: %s at %s", e.BlockType, e.Range.String())
}

// NewPorchConfig creates a new PorchConfig instance with the provided base directory,
// CLI flag assigned variables, context, and HCL blocks.
func NewPorchConfig(
	ctx context.Context,
	baseDir string,
	cliFlagAssignedVariables []golden.CliFlagAssignedVariables,
	hclBlocks []*golden.HclBlock,
) (*PorchConfig, error) {
	cfg := &PorchConfig{
		BaseConfig: golden.NewBasicConfig(baseDir, "porch", "porch", nil, cliFlagAssignedVariables, ctx),
	}

	err := golden.InitConfig(cfg, hclBlocks)

	if err != nil {
		err = errors.Join(ErrInitConfig, err)
	}

	return cfg, err
}

// BuildPorchConfig constructs a PorchConfig instance by loading HCL blocks
// from the specified configuration directory.
func BuildPorchConfig(
	ctx context.Context,
	baseDir, cfgDir string,
	cliFlagAssignedVariables []golden.CliFlagAssignedVariables,
) (*PorchConfig, error) {
	var err error

	hclBlocks, err := loadPorchHclBlocks(false, cfgDir)
	if err != nil {
		return nil, err
	}

	c, err := NewPorchConfig(ctx, baseDir, cliFlagAssignedVariables, hclBlocks)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func loadPorchHclBlocks(ignoreUnsupportedBlock bool, dir string) ([]*golden.HclBlock, error) {
	fs := FsFactory()

	matches, err := afero.Glob(fs, filepath.Join(dir, "*"+porchConfigFileExt))
	if err != nil {
		// the only error we expect here is ErrBadPattern, which should never happen as it is a constant.
		panic(err)
	}

	if len(matches) == 0 {
		return nil, ErrNoPorchConfigFile
	}

	var blocks []*golden.HclBlock

	for _, filename := range matches {
		content, fsErr := afero.ReadFile(fs, filename)
		if fsErr != nil {
			err = multierror.Append(err, fsErr)
			continue
		}

		readFile, diag := hclsyntax.ParseConfig(content, filename, hcl.InitialPos)
		if diag.HasErrors() {
			err = multierror.Append(err, diag.Errs()...)
			continue
		}

		writeFile, _ := hclwrite.ParseConfig(content, filename, hcl.InitialPos)
		readBody := readFile.Body.(*hclsyntax.Body)
		writeBody := writeFile.Body()
		blocks = append(blocks, golden.AsHclBlocks(readBody.Blocks, writeBody.Blocks())...)
	}

	if err != nil {
		return nil, errors.Join(ErrParsePorchConfigFile, err)
	}

	var r []*golden.HclBlock

	// First loop: parse all rule blocks
	for _, b := range blocks {
		if golden.IsBlockTypeWanted(b.Type) {
			r = append(r, b)
			continue
		}

		if !ignoreUnsupportedBlock {
			err = multierror.Append(errors.Join(NewErrInvalidBlockType(b.Type, b.Range())), err)
		}
	}

	if err != nil {
		err = errors.Join(ErrParsePorchConfigFile, err)
	}

	return r, err
}
