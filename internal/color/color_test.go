// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package color

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsColorEnabled(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	assert.False(t, isColorCapable(), "Expected color output to be disabled")

	t.Setenv("FORCE_COLOR", "1")
	assert.False(t, isColorCapable(), "Expected color output to be disabled as NO_COLOR is still set")

	t.Setenv("NO_COLOR", "")
	assert.True(t, isColorCapable(), "Expected color output to be enabled as FORCE_COLOR is set and NO_COLOR is unset")
}
