package hcl

import "github.com/Azure/golden"

func init() {
	golden.RegisterBlock(new(WorkflowBlock))
}
