// Copyright (c) matt-FFFFFF 2025. All rights reserved.
// SPDX-License-Identifier: MIT

package config_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/matt-FFFFFF/porch/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircularDependencyDetection(t *testing.T) {
	t.Run("simple circular dependency", func(t *testing.T) {
		yamlData := `
name: "Test Circular Dependency"
description: "Test circular dependency detection"
command_groups:
  - name: "group_a"
    commands:
      - type: "serial"
        name: "Reference B"
        command_group: "group_b"
  - name: "group_b"
    commands:
      - type: "serial"
        name: "Reference A"
        command_group: "group_a"
commands:
  - type: "serial"
    name: "Start"
    command_group: "group_a"
`

		ctx := context.Background()
		_, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "circular dependency")
		assert.Contains(t, err.Error(), "group_a")
		assert.Contains(t, err.Error(), "group_b")
	})

	t.Run("three-way circular dependency", func(t *testing.T) {
		yamlData := `
name: "Test Three-way Circular Dependency"
description: "Test three-way circular dependency detection"
command_groups:
  - name: "group_a"
    commands:
      - type: "serial"
        name: "Reference B"
        command_group: "group_b"
  - name: "group_b"
    commands:
      - type: "serial"
        name: "Reference C"
        command_group: "group_c"
  - name: "group_c"
    commands:
      - type: "serial"
        name: "Reference A"
        command_group: "group_a"
commands:
  - type: "serial"
    name: "Start"
    command_group: "group_a"
`

		ctx := context.Background()
		_, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "circular dependency")
	})

	t.Run("self-referencing group", func(t *testing.T) {
		yamlData := `
name: "Test Self-referencing Group"
description: "Test self-referencing group detection"
command_groups:
  - name: "group_a"
    commands:
      - type: "shell"
        name: "First Command"
        command_line: "echo 'first'"
      - type: "serial"
        name: "Self Reference"
        command_group: "group_a"
commands:
  - type: "serial"
    name: "Start"
    command_group: "group_a"
`

		ctx := context.Background()
		_, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "circular dependency")
		assert.Contains(t, err.Error(), "group_a")
	})

	t.Run("valid nested groups", func(t *testing.T) {
		yamlData := `
name: "Test Valid Nested Groups"
description: "Test valid nested groups (no circular dependency)"
command_groups:
  - name: "group_a"
    commands:
      - type: "shell"
        name: "First Command"
        command_line: "echo 'first'"
      - type: "serial"
        name: "Reference B"
        command_group: "group_b"
  - name: "group_b"
    commands:
      - type: "shell"
        name: "Second Command"
        command_line: "echo 'second'"
      - type: "serial"
        name: "Reference C"
        command_group: "group_c"
  - name: "group_c"
    commands:
      - type: "shell"
        name: "Third Command"
        command_line: "echo 'third'"
commands:
  - type: "serial"
    name: "Start"
    command_group: "group_a"
`

		ctx := context.Background()
		runnable, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

		require.NoError(t, err)
		assert.NotNil(t, runnable)
	})

	t.Run("circular dependency with parallel commands", func(t *testing.T) {
		yamlData := `
name: "Test Circular Dependency with Parallel"
description: "Test circular dependency detection with parallel commands"
command_groups:
  - name: "group_a"
    commands:
      - type: "parallel"
        name: "Reference B"
        command_group: "group_b"
  - name: "group_b"
    commands:
      - type: "parallel"
        name: "Reference A"
        command_group: "group_a"
commands:
  - type: "parallel"
    name: "Start"
    command_group: "group_a"
`

		ctx := context.Background()
		_, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "circular dependency")
	})
}

func TestConfigurationTimeout(t *testing.T) {
	t.Run("configuration respects context cancellation", func(t *testing.T) {
		yamlData := `
name: "Test Configuration Timeout"
description: "Test configuration timeout handling"
commands:
  - type: "shell"
    name: "Simple Command"
    command_line: "echo 'test'"
`

		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "timed out")
	})

	t.Run("configuration respects context deadline", func(t *testing.T) {
		yamlData := `
name: "Test Configuration Deadline"
description: "Test configuration deadline handling"
commands:
  - type: "shell"
    name: "Simple Command"
    command_line: "echo 'test'"
`

		// Create a context with a very short deadline
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait for the context to expire
		<-ctx.Done()

		_, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "timed out")
	})
}

func TestMaxRecursionDepth(t *testing.T) {
	t.Run("detect excessive recursion depth", func(t *testing.T) {
		// This test creates a very deep nesting to trigger max recursion depth
		ctx := context.Background()

		// Create a deep chain of command groups that reference each other in a line
		// This should trigger the max recursion depth error
		yamlData := `
name: "Test Max Recursion Depth"
description: "Test max recursion depth detection"
command_groups:`

		// Create a chain of 150 groups (exceeding MaxRecursionDepth of 100)
		for i := 0; i < 150; i++ {
			groupName := fmt.Sprintf("group_%d", i)
			nextGroupName := fmt.Sprintf("group_%d", i+1)
			if i == 149 {
				// Last group doesn't reference another
				yamlData += fmt.Sprintf(`
  - name: "%s"
    commands:
      - type: "shell"
        name: "End Command"
        command_line: "echo 'end'"`, groupName)
			} else {
				yamlData += fmt.Sprintf(`
  - name: "%s"
    commands:
      - type: "serial"
        name: "Reference Next"
        command_group: "%s"`, groupName, nextGroupName)
			}
		}

		yamlData += `
commands:
  - type: "serial"
    name: "Start"
    command_group: "group_0"
`

		_, err := config.BuildFromYAML(ctx, testRegistry, []byte(yamlData))

		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "recursion depth")
	})
}
