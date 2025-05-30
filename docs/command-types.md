# Command Type System

This document describes how to implement custom command types in the avmtool system using YAML-based configuration.

## Overview

The system uses `github.com/goccy/go-yaml` to decode command definitions. Each command type:

1. Must have a `type` field that identifies the command type
2. Can define its own YAML schema for additional fields
3. Must implement the `commands.Commander` interface
4. Must register itself with the `commandregistry` package

## Architecture

- `internal/commandregistry/` - Central registry for command types
- `internal/commands/` - Base interfaces and common definitions
- `internal/commands/{commandtype}/` - Individual command implementations
- `internal/config/` - High-level configuration processing

## Creating a New Command Type

### 1. Create the Command Package

Create a new directory under `internal/commands/{yourcommandtype}/`

### 2. Define the YAML Schema

Create a `Definition` struct that embeds `commands.BaseDefinition`:

```go
// Definition represents the YAML configuration for your command.
type Definition struct {
    commands.BaseDefinition `yaml:",inline"`
    // Add your custom fields here
    CustomField string   `yaml:"customField"`
    Options     []string `yaml:"options,omitempty"`
}
```

### 3. Implement the Commander Interface

```go
const CommandType = "yourcommandtype"

var _ commands.Commander = (*Commander)(nil)

// init registers the command type.
func init() {
    commandregistry.Register(CommandType, &Commander{})
}

type Commander struct{}

func (c *Commander) Create(ctx context.Context, payload []byte) (runbatch.Runnable, error) {
    def := new(Definition)
    if err := yaml.Unmarshal(payload, def); err != nil {
        return nil, fmt.Errorf("failed to unmarshal command definition: %w", err)
    }

    // Create and return your runnable implementation
    return YourRunnableImplementation(def), nil
}
```

### 4. Register the Command

Import your command package in `internal/config/commandChain.go`:

```go
import (
    // ... other imports
    _ "github.com/matt-FFFFFF/avmtool/internal/commands/yourcommandtype"
)
```

## Common Base Fields

All commands inherit these fields from `commands.BaseDefinition`:

- `type` (string) - The command type identifier
- `name` (string) - Human-readable name for the command
- `cwd` (string, optional) - Working directory for the command

## Examples

### Simple Command (shell command)

```yaml
- type: "shell"
  name: "List Files"
  exec: "ls"
  args:
    - "-la"
  cwd: "/tmp"
```

### Container Command (serial execution)

```yaml
- type: "serial"
  name: "Sequential Tasks"
  commands:
    - type: "shell"
      name: "First Task"
      exec: "echo"
      args: ["Starting"]

    - type: "shell"
      name: "Second Task"
      exec: "echo"
      args: ["Done"]
```

### Container Command (parallel execution)

```yaml
- type: "parallel"
  name: "Parallel Tasks"
  commands:
    - type: "shell"
      name: "Task A"
      exec: "echo"
      args: ["Task A"]

    - type: "shell"
      name: "Task B"
      exec: "echo"
      args: ["Task B"]
```

## Available Command Types

1. **shell** - Execute shell commands
   - `exec` (string) - Command to execute
   - `args` ([]string) - Command arguments

2. **copycwdtotemp** - Copy current directory to temporary location
   - No additional fields

3. **serial** - Execute commands sequentially
   - `commands` ([]interface{}) - List of commands to execute

4. **parallel** - Execute commands in parallel
   - `commands` ([]interface{}) - List of commands to execute

## Testing

Create integration tests in `internal/config/integration_test.go` to verify your command type works correctly with the YAML configuration system.
