// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package runbatch

import (
	"slices"
	"strings"
)

const (
	fullLabelInitialSliceSize = 10 // Initial size for the labels slice in FullLabel
)

// FullLabel returns the full label of a Runnable, including its parent labels.
func FullLabel(r Runnable) string {
	if r == nil {
		return "Unknown"
	}

	sb := strings.Builder{}
	labels := make([]string, 0, fullLabelInitialSliceSize)

	parent := r.GetParent()
	if parent == nil {
		return r.GetLabel()
	}

	labels = append(labels, r.GetLabel())
	for parent != nil {
		labels = append(labels, parent.GetLabel())
		parent = parent.GetParent()
	}

	for _, v := range slices.Backward(labels) {
		if sb.Len() > 0 {
			sb.WriteString(" > ")
		}

		sb.WriteString(v)
	}

	return sb.String()
}
