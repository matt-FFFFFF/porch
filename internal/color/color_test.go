// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package color

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsColorEnabled(t *testing.T) {
	assert.False(t, isColorEnabled(), "Expected color output to be disabled")

	os.Setenv("FORCE_COLOR", "1")
	assert.False(t, isColorEnabled(), "Expected color output to be enabled as NO_COLOR is still set")

	os.Unsetenv("NO_COLOR")
	assert.True(t, isColorEnabled(), "Expected color output to be enabled as FORCE_COLOR is set and NO_COLOR is unset")
}
