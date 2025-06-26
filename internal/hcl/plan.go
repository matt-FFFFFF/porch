package hcl

import (
	"github.com/Azure/golden"
	"sync"
)

func RunPorchPlan(c *PorchConfig) (*PorchPlan, error) {
	err := c.RunPlan()
	if err != nil {
		return nil, err
	}

	plan := newPlan(c)
	for _, rb := range golden.Blocks[*WorkflowBlock](c) {
		plan.addWorkflow(rb)
	}

	return plan, nil
}

func newPlan(c *PorchConfig) *PorchPlan {
	return &PorchPlan{
		c: c,
	}
}

type PorchPlan struct {
	Workflows []*WorkflowBlock
	c         *PorchConfig
	mu        sync.Mutex
}

func (p *PorchPlan) addWorkflow(c *WorkflowBlock) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Workflows = append(p.Workflows, c)

}
