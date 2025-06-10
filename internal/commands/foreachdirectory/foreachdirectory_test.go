// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package foreachdirectory

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain is used to run the goleak verification before and after tests.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
