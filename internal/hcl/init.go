// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package hcl

import "github.com/Azure/golden"

const (
	commandBlockCtyTypeDepth = 200
)

func init() {
	golden.RegisterBlock(new(WorkflowBlock))
	golden.AddCustomTypeMapping[*CommandBlock](commandBlockCtyType(commandBlockCtyTypeDepth))
}
