package hcl

import (
	"github.com/Azure/golden"
	"github.com/zclconf/go-cty/cty"
)

var _ golden.ApplyBlock = (*WorkflowBlock)(nil)

type WorkflowBlock struct {
	*golden.BaseBlock
	WorkflowName string          `hcl:"name"`
	Description  string          `hcl:"description,optional"`
	Source       string          `hcl:"source,optional"`
	Commands     []*CommandBlock `hcl:"command,block"`
}

func (b *WorkflowBlock) Type() string {
	return ""
}

func (b *WorkflowBlock) BlockType() string {
	return "workflow"
}

func (b *WorkflowBlock) AddressLength() int {
	return 2
}

func (b *WorkflowBlock) CanExecutePrePlan() bool {
	return false
}

func (b *WorkflowBlock) Apply() error {
	// Implement the logic to apply the workflow block
	// This is a placeholder for actual implementation
	return nil
}

func (b *WorkflowBlock) Address() string {
	return "workflow." + b.WorkflowName
}

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
