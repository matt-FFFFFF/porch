package hcl

import (
	"github.com/Azure/golden"
)

var _ golden.ApplyBlock = (*WorkflowBlock)(nil)

type WorkflowBlock struct {
	*golden.BaseBlock
	WorkflowName string         `hcl:"name"`
	Description  string         `hcl:"description,optional"`
	Source       string         `hcl:"source,optional"`
	Command      []CommandBlock `hcl:"command,block"`
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
	Commands []NestedCommandBlock `hcl:"command,block"`
}

type NestedCommandBlock struct {
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
}
