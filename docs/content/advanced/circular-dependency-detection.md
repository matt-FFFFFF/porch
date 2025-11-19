+++
title = "Circular Dependency Detection"
weight = 1
+++

## Overview

The porch configuration system now includes robust circular dependency detection to prevent infinite loops in command group references. This feature helps catch configuration errors early and provides clear error messages to help debug issues.

## Features

### 1. Circular Dependency Detection

The system detects several types of circular dependencies:

- **Simple circular dependencies**: Group A references Group B, which references Group A
- **Multi-way circular dependencies**: Group A → Group B → Group C → Group A
- **Self-referencing groups**: A group that references itself
- **Deep nested cycles**: Circular dependencies buried deep in nested command structures

### 2. Maximum Recursion Depth Protection

To prevent stack overflow and excessive processing, the system enforces a maximum recursion depth of 100 levels when resolving command groups.

### 3. Enhanced Error Messages

When a circular dependency is detected, the system provides clear error messages showing:

- The exact circular path (e.g., "group_a → group_b → group_a")
- Which command within a group caused the issue
- The command index for easier debugging

### 4. Context Cancellation Support

Configuration building now respects context cancellation, allowing for:

- Graceful interruption with Ctrl+C during config parsing
- Timeout protection (30-second default for config building)
- Responsive signal handling during long configuration processes

## Examples

### Detecting Circular Dependencies

**Example 1: Simple Circular Dependency**

```yaml
name: infinite loop example
command_groups:
  - name: group_one
    commands:
      - type: serial
        name: reference to two
        command_group: group_two
  - name: group_two
    commands:
      - type: serial
        name: reference to one
        command_group: group_one
commands:
  - type: parallel
    name: start
    command_group: group_one
```

**Error Output:**

```
failed to build config: invalid command group 'group_one':
in command 0 of group group_one: circular dependency detected: group_one → group_two → group_one
```

**Example 2: Self-Referencing Group**

```yaml
name: self reference example
command_groups:
  - name: recursive_group
    commands:
      - type: shell
        name: first command
        command_line: echo "first"
      - type: serial
        name: self reference
        command_group: recursive_group
commands:
  - type: serial
    name: start
    command_group: recursive_group
```

**Error Output:**

```
failed to build config: invalid command group 'recursive_group':
in command 1 of group recursive_group: circular dependency detected: recursive_group → recursive_group
```

### Valid Nested Groups

This configuration is valid and will work correctly:

```yaml
name: valid nested example
command_groups:
  - name: setup
    commands:
      - type: shell
        name: prepare
        command_line: echo "preparing"
      - type: serial
        name: run tests
        command_group: test_suite
  - name: test_suite
    commands:
      - type: shell
        name: unit tests
        command_line: echo "running unit tests"
      - type: serial
        name: cleanup
        command_group: cleanup
  - name: cleanup
    commands:
      - type: shell
        name: clean up
        command_line: echo "cleaning up"
commands:
  - type: serial
    name: main workflow
    command_group: setup
```

## Signal Handling Improvements

### Configuration Building Timeout

Configuration building now has a 30-second timeout to prevent hanging on malformed configurations:

```go
// In cmd/porch/run/run.go
configCtx, configCancel := context.WithTimeout(ctx, 30*time.Second)
defer configCancel()

rb, err := config.BuildFromYAML(configCtx, factory, bytes)
```

### Context Cancellation Checks

The system now checks for context cancellation at key points:

- Before starting configuration parsing
- After command group validation
- During individual command processing

### Graceful Interruption

When you press Ctrl+C during configuration building, you'll see:

```
failed to build config: configuration building timed out: context canceled
```

## Implementation Details

### Registry-Level Detection

The `commandregistry.Registry` now includes:

- `resolveCommandGroupWithDepth()`: Tracks recursion depth and visiting state
- `validateCommandForCircularDeps()`: Validates individual commands for group references
- `formatCircularDependencyPath()`: Creates human-readable circular dependency paths

### Configuration-Level Validation

The `config.BuildFromYAML()` function:

- Validates all command groups before proceeding with command creation
- Includes context cancellation checks throughout the process
- Provides detailed error messages with command indices

### Error Types

New error types have been added:

- `ErrCircularDependency`: For circular dependency detection
- `ErrConfigurationTimeout`: For configuration timeout handling
- `ErrMaxRecursionDepth`: For recursion depth protection

## Testing

Comprehensive tests have been added covering:

- Simple circular dependencies
- Multi-way circular dependencies
- Self-referencing groups
- Maximum recursion depth
- Context cancellation
- Configuration timeouts

Run tests with:

```bash
go test ./internal/config -v -run TestCircularDependencyDetection
go test ./internal/config -v -run TestConfigurationTimeout
go test ./internal/commandregistry -v
```
