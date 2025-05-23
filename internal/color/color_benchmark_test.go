// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package color

import (
	"math/rand"
	"testing"
)

func BenchmarkColorizeSB(b *testing.B) {
	s := randStringRunes(10)

	b.ResetTimer()

	for b.Loop() {
		Colorize(s, FgRed)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}
