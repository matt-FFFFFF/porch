// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package registry

import (
	"context"

	"github.com/matt-FFFFFF/avmtool/internal/providers"
	"github.com/matt-FFFFFF/avmtool/internal/runbatch"
)

// ItemProvider is a function that returns a list of items to iterate over.
type ItemProvider = runbatch.ItemsProviderFunc

// ItemProviderRegistry is a map of item provider names to their implementations.
type ItemProviderRegistry map[string]ItemProvider

// DefaultItemProviderRegistry is the default registry for item providers.
var DefaultItemProviderRegistry = ItemProviderRegistry{
	// Fixed list example
	"example": func(ctx context.Context, _ string) ([]string, error) {
		return []string{"item1", "item2", "item3"}, nil
	},

	// File-based providers
	"list-go-files":    providers.ListFiles("*.go"),
	"list-yaml-files":  providers.ListFiles("*.yaml"),
	"list-directories": providers.ListDirectories("."),

	// String-based provider
	"comma-separated": providers.SplitString("item1,item2,item3", ","),
}

// Register adds a new item provider to the registry.
func (r ItemProviderRegistry) Register(name string, provider ItemProvider) {
	r[name] = provider
}
