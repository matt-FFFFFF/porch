// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package registry

import (
	"context"
)

// ItemsProviderFunc is a function that returns a list of items to iterate over.
// It takes a context and the current working directory, and returns a list of items and an error.
type ItemsProviderFunc func(ctx context.Context, workingDirectory string) ([]string, error)

// ItemProviderRegistry is a map of item provider names to their implementations
type ItemProviderRegistry map[string]ItemsProviderFunc

// DefaultItemProviderRegistry is the default registry for item providers
var DefaultItemProviderRegistry = ItemProviderRegistry{
	// Example provider that returns a fixed list
	"example": func(ctx context.Context, _ string) ([]string, error) {
		return []string{"item1", "item2", "item3"}, nil
	},
}

// Register adds a new item provider to the registry
func (r ItemProviderRegistry) Register(name string, provider ItemsProviderFunc) {
	r[name] = provider
}
