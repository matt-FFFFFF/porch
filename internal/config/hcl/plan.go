// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package hcl

import (
	"errors"
	"sync"

	"github.com/Azure/golden"
)

var (
	// ErrRunPlan is returned when there is an error executing the Porch plan.
	ErrRunPlan = errors.New("failed to execute Porch plan")
)

// RunPorchPlan executes the Porch plan by running the configuration and collecting workflow blocks.
func RunPorchPlan(c *PorchConfig) (*PorchPlan, error) {
	err := c.RunPlan()
	if err != nil {
		return nil, errors.Join(ErrRunPlan, err)
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

// PorchPlan represents a plan in the Porch configuration, containing workflow blocks.
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
