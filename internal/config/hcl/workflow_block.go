// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package hcl

import (
	"strings"

	"github.com/Azure/golden"
	"github.com/zclconf/go-cty/cty"
)

const (
	workflowBlockAddressLength = 2
	workflowBlockName          = "workflow"
)

var _ golden.ApplyBlock = (*WorkflowBlock)(nil)

// WorkflowBlock represents a workflow block in the Porch configuration.
type WorkflowBlock struct {
	*golden.BaseBlock
	WorkflowName string          `hcl:"name"`
	Description  string          `hcl:"description,optional"`
	Source       string          `hcl:"source,optional"`
	Commands     []*CommandBlock `hcl:"command,block"`
}

// Type returns the type of the block.
func (b *WorkflowBlock) Type() string {
	return ""
}

// BlockType returns the type of the block, which is "workflow" for WorkflowBlock.
func (b *WorkflowBlock) BlockType() string {
	return workflowBlockName
}

// AddressLength returns the length of the address for the block.
func (b *WorkflowBlock) AddressLength() int {
	return workflowBlockAddressLength
}

// CanExecutePrePlan checks if the block can be executed before the plan is applied.
func (b *WorkflowBlock) CanExecutePrePlan() bool {
	return false
}

// Apply applies the workflow block, executing its commands and handling any dependencies.
func (b *WorkflowBlock) Apply() error {
	// Implement the logic to apply the workflow block
	// This is a placeholder for actual implementation
	return nil
}

// Address returns the address of the workflow block, which is prefixed with "workflow." followed by the workflow name.
func (b *WorkflowBlock) Address() string {
	return strings.Join([]string{workflowBlockName, b.WorkflowName}, ".")
}

// CommandBlock represents a command block within a workflow.
type CommandBlock struct {
	Type             string            `hcl:"type"`
	Name             string            `hcl:"name,optional"`
	WorkingDirectory string            `hcl:"working_directory,optional"`
	RunsOnCondition  string            `hcl:"runs_on_condition,optional"`
	RunsOnExitCodes  []int             `hcl:"runs_on_exit_codes,optional"`
	Enabled          *bool             `hcl:"enabled,optional"`
	Env              map[string]string `hcl:"env,optional"`

	// Shell/PowerShell specific attributes
	CommandLine      string `hcl:"command_line,optional"`
	Script           string `hcl:"script,optional"`
	ScriptFile       string `hcl:"script_file,optional"`
	SuccessExitCodes []int  `hcl:"success_exit_codes,optional"`
	SkipExitCodes    []int  `hcl:"skip_exit_codes,optional"`

	// Foreachdirectory specific attributes
	Mode                     string `hcl:"mode,optional"`
	WorkingDirectoryStrategy string `hcl:"working_directory_strategy,optional"`
	Depth                    int    `hcl:"depth,optional"`
	IncludeHidden            bool   `hcl:"include_hidden,optional"`
	SkipOnNotExist           bool   `hcl:"skip_on_not_exist,optional"`

	// Copy command specific
	CWD string `hcl:"cwd,optional"`

	// Nested commands (for serial, parallel, foreachdirectory)
	Commands []*CommandBlock `hcl:"command,block"`
}

func commandBlockCtyType(depth int) cty.Type {
	if depth == 0 {
		return cty.ObjectWithOptionalAttrs(map[string]cty.Type{
			"type":                       cty.String,
			"name":                       cty.String,
			"working_directory":          cty.String,
			"runs_on_condition":          cty.String,
			"runs_on_exit_codes":         cty.List(cty.Number),
			"enabled":                    cty.Bool,
			"env":                        cty.Map(cty.String),
			"command_line":               cty.String,
			"script":                     cty.String,
			"script_file":                cty.String,
			"success_exit_codes":         cty.List(cty.Number),
			"skip_exit_codes":            cty.List(cty.Number),
			"mode":                       cty.String,
			"working_directory_strategy": cty.String,
			"depth":                      cty.Number,
			"cwd":                        cty.String,
		}, []string{
			"name",
			"working_directory",
			"runs_on_condition",
			"runs_on_exit_codes",
			"enabled",
			"env",
			"command_line",
			"script",
			"script_file",
			"success_exit_codes",
			"skip_exit_codes",
			"mode",
			"working_directory_strategy",
			"depth",
			"cwd",
		})
	}

	return cty.ObjectWithOptionalAttrs(map[string]cty.Type{
		"type":                       cty.String,
		"name":                       cty.String,
		"working_directory":          cty.String,
		"runs_on_condition":          cty.String,
		"runs_on_exit_codes":         cty.List(cty.Number),
		"enabled":                    cty.Bool,
		"env":                        cty.Map(cty.String),
		"command_line":               cty.String,
		"script":                     cty.String,
		"script_file":                cty.String,
		"success_exit_codes":         cty.List(cty.Number),
		"skip_exit_codes":            cty.List(cty.Number),
		"mode":                       cty.String,
		"working_directory_strategy": cty.String,
		"depth":                      cty.Number,
		"cwd":                        cty.String,
		"command":                    cty.List(commandBlockCtyType(depth - 1)),
	}, []string{
		"name",
		"working_directory",
		"runs_on_condition",
		"runs_on_exit_codes",
		"enabled",
		"env",
		"command_line",
		"script",
		"script_file",
		"success_exit_codes",
		"skip_exit_codes",
		"mode",
		"working_directory_strategy",
		"depth",
		"cwd",
		"command",
	})
}
