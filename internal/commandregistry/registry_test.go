// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package commandregistry

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircularDependencyDetection(t *testing.T) {
	t.Run("detect simple circular dependency", func(t *testing.T) {
		registry := New()

		// Add command groups with circular dependency
		registry.AddCommandGroup("group_a", []any{
			map[string]any{
				"type":          "serial",
				"name":          "Reference B",
				"command_group": "group_b",
			},
		})

		registry.AddCommandGroup("group_b", []any{
			map[string]any{
				"type":          "serial",
				"name":          "Reference A",
				"command_group": "group_a",
			},
		})

		// Try to resolve group_a, should detect circular dependency
		_, err := registry.ResolveCommandGroup("group_a")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency")
		assert.Contains(t, err.Error(), "group_a")
		assert.Contains(t, err.Error(), "group_b")
	})

	t.Run("detect self-referencing group", func(t *testing.T) {
		registry := New()

		// Add a self-referencing command group
		registry.AddCommandGroup("self_ref", []any{
			map[string]any{
				"type":         "shell",
				"name":         "First Command",
				"command_line": "echo 'first'",
			},
			map[string]any{
				"type":          "serial",
				"name":          "Self Reference",
				"command_group": "self_ref",
			},
		})

		// Try to resolve self_ref, should detect circular dependency
		_, err := registry.ResolveCommandGroup("self_ref")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency")
		assert.Contains(t, err.Error(), "self_ref")
	})

	t.Run("allow valid nested groups", func(t *testing.T) {
		registry := New()

		// Add valid nested command groups (no circular dependency)
		registry.AddCommandGroup("group_a", []any{
			map[string]any{
				"type":         "shell",
				"name":         "First Command",
				"command_line": "echo 'first'",
			},
			map[string]any{
				"type":          "serial",
				"name":          "Reference B",
				"command_group": "group_b",
			},
		})

		registry.AddCommandGroup("group_b", []any{
			map[string]any{
				"type":         "shell",
				"name":         "Second Command",
				"command_line": "echo 'second'",
			},
		})

		// Should resolve without error
		commands, err := registry.ResolveCommandGroup("group_a")

		require.NoError(t, err)
		assert.Len(t, commands, 2)
	})

	t.Run("detect three-way circular dependency", func(t *testing.T) {
		registry := New()

		// Add command groups with three-way circular dependency
		registry.AddCommandGroup("group_a", []any{
			map[string]any{
				"type":          "serial",
				"name":          "Reference B",
				"command_group": "group_b",
			},
		})

		registry.AddCommandGroup("group_b", []any{
			map[string]any{
				"type":          "serial",
				"name":          "Reference C",
				"command_group": "group_c",
			},
		})

		registry.AddCommandGroup("group_c", []any{
			map[string]any{
				"type":          "serial",
				"name":          "Reference A",
				"command_group": "group_a",
			},
		})

		// Try to resolve group_a, should detect circular dependency
		_, err := registry.ResolveCommandGroup("group_a")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency")
	})

	t.Run("detect unknown command group", func(t *testing.T) {
		registry := New()

		// Try to resolve a non-existent group
		_, err := registry.ResolveCommandGroup("nonexistent")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command group")
		assert.Contains(t, err.Error(), "nonexistent")
	})

	t.Run("command referencing unknown group", func(t *testing.T) {
		registry := New()

		// Add a command group that references a non-existent group
		registry.AddCommandGroup("group_a", []any{
			map[string]any{
				"type":          "serial",
				"name":          "Reference Unknown",
				"command_group": "unknown_group",
			},
		})

		// Try to resolve group_a, should fail because of unknown reference
		_, err := registry.ResolveCommandGroup("group_a")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command group")
		assert.Contains(t, err.Error(), "unknown_group")
	})
}

func TestMaxRecursionDepth(t *testing.T) {
	t.Run("detect excessive recursion depth", func(t *testing.T) {
		registry := New()

		// Create a very deep chain that exceeds MaxRecursionDepth
		for i := 0; i < 150; i++ {
			groupName := fmt.Sprintf("group_%d", i)
			nextGroupName := fmt.Sprintf("group_%d", i+1)

			if i == 149 {
				// Last group doesn't reference another
				registry.AddCommandGroup(groupName, []any{
					map[string]any{
						"type":         "shell",
						"name":         "End Command",
						"command_line": "echo 'end'",
					},
				})
			} else {
				registry.AddCommandGroup(groupName, []any{
					map[string]any{
						"type":          "serial",
						"name":          "Reference Next",
						"command_group": nextGroupName,
					},
				})
			}
		}

		// Try to resolve the first group, should fail due to max recursion depth
		_, err := registry.ResolveCommandGroup("group_0")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "recursion depth")
	})
}

func TestFormatCircularDependencyPath(t *testing.T) {
	tests := []struct {
		name     string
		path     []string
		expected string
	}{
		{
			name:     "empty path",
			path:     []string{},
			expected: "unknown path",
		},
		{
			name:     "single element",
			path:     []string{"a"},
			expected: "a → a",
		},
		{
			name:     "two elements",
			path:     []string{"a", "b"},
			expected: "a → b → a",
		},
		{
			name:     "three elements",
			path:     []string{"a", "b", "c"},
			expected: "a → b → c → a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCircularDependencyPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
