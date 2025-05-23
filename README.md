# AVMTool - A Powerful Command Orchestration Framework

AVMTool is a sophisticated command orchestration framework for running and managing complex command workflows. It provides a flexible way to define, compose, and execute command chains with sophisticated flow control, parallel processing, and comprehensive error handling.

![AVMTool Diagram](https://via.placeholder.com/800x400?text=AVMTool+Command+Orchestration)

## Features

- **Sophisticated Command Orchestration**
  - Serial execution for dependent commands
  - Parallel execution for independent commands
  - ForEach execution to process items in collection (parallel or serial)
  - Function commands for executing Go code inline

- **Flexible Configuration**
  - Define command chains via YAML
  - Define complex workflows with nested batches
  - Dynamic item providers for ForEach operations
  - Variable substitution in commands

- **Robust Error Handling**
  - Comprehensive error reporting
  - Graceful signal handling (SIGINT, SIGTERM)
  - Context-aware command execution
  - Detailed execution results

- **Beautiful Output**
  - Pretty-printed execution results
  - Tree visualization of command execution
  - Colorized output (with terminal detection)
  - Detailed or summarized results

- **Extensible Architecture**
  - Plugin-based command registry
  - Custom item providers
  - Working directory management avmtool - a portable process orchestrater

## Installation

```bash
go install github.com/matt-FFFFFF/avmtool@latest
```

Or build from source:

```bash
git clone https://github.com/matt-FFFFFF/avmtool.git
cd avmtool
go build -o avmtool
```

## Quick Start

### Basic Command Chain

Create a YAML file defining your command chain:

```yaml
name: "Basic Workflow"
description: "A simple workflow with sequential and parallel tasks"
commands:
  - type: "serial"
    name: "Setup and Build"
    commands:
      - type: "command"
        name: "Check Dependencies"
        command: "commandinpath"
        args:
          - "go"
          - "git"

      - type: "parallel"
        name: "Build Process"
        commands:
          - type: "command"
            name: "Lint Code"
            command: "oscommand"
            args:
              - "go"
              - "vet"
              - "./..."

          - type: "command"
            name: "Run Tests"
            command: "oscommand"
            args:
              - "go"
              - "test"
              - "./..."
```

Run it with:

```bash
avmtool run workflow.yaml
```

### ForEach Iteration

Process multiple items in parallel or series:

```yaml
name: "Process Files"
description: "Process all Go files in parallel"
commands:
  - type: "foreach"
    name: "Go Files"
    itemProvider: "list-go-files"
    mode: "parallel"
    itemVariable: "FILE"
    commands:
      - type: "command"
        name: "Format File"
        command: "oscommand"
        args:
          - "gofmt"
          - "-w"
          - "${FILE}"
```

## Command Types

AVMTool supports several command types:

### 1. Basic Command

Executes a single command:

```yaml
- type: "command"
  name: "Build Project"
  command: "oscommand"
  args:
    - "go"
    - "build"
```

### 2. Serial Batch

Executes commands in sequence:

```yaml
- type: "serial"
  name: "Setup and Build"
  commands:
    - type: "command"
      name: "Clean"
      # ...
    - type: "command"
      name: "Build"
      # ...
```

### 3. Parallel Batch

Executes commands in parallel:

```yaml
- type: "parallel"
  name: "Test and Lint"
  commands:
    - type: "command"
      name: "Run Tests"
      # ...
    - type: "command"
      name: "Run Linter"
      # ...
```

### 4. ForEach Command

Processes a collection of items:

```yaml
- type: "foreach"
  name: "Process Directories"
  itemProvider: "list-directories"
  mode: "serial"  # or "parallel"
  itemVariable: "DIR"
  commands:
    # Commands to run for each directory
```

## Item Providers

AVMTool includes several built-in item providers for ForEach commands:

- `list-go-files`: Lists all Go files in the current directory
- `list-yaml-files`: Lists all YAML files
- `list-directories`: Lists all directories
- `comma-separated`: Splits a comma-separated string into items

Custom providers can be registered programmatically.

## Extending AVMTool

### Creating Custom Commands

Implement the `Commander` interface:

```go
type Commander interface {
  Create(name, exec, cwd string, args ...string) (runbatch.Runnable, error)
}
```

### Creating Custom Item Providers

Create and register item provider functions:

```go
func MyProvider() runbatch.ItemsProviderFunc {
  return func(ctx context.Context, workingDir string) ([]string, error) {
    // Return a slice of items
    return []string{"item1", "item2", "item3"}, nil
  }
}

// Register provider
registry.DefaultItemProviderRegistry.Register("my-provider", MyProvider())
```

## Advanced Examples

### Working Directory Management

```yaml
- type: "serial"
  name: "Work in Temp Directory"
  commands:
    - type: "command"
      name: "Copy to Temp"
      command: "copycwdtotemp"
    - type: "command"
      name: "Run in Temp"
      # ...
```

### Environment Variable Substitution

```yaml
- type: "foreach"
  name: "Process Environments"
  itemProvider: "comma-separated"  # Returns: "dev,staging,prod"
  itemVariable: "ENV"
  commands:
    - type: "command"
      name: "Deploy to ${ENV}"
      command: "oscommand"
      args:
        - "./deploy.sh"
        - "${ENV}"
```

## Core Concepts

### Runnable Interface

The foundation of AVMTool is the `Runnable` interface, which defines anything that can be executed:

```go
type Runnable interface {
  Run(context.Context) Results
  GetLabel() string
  SetCwd(string)
}
```

### Results

All command executions produce structured results with rich metadata:

```go
type Result struct {
  ExitCode int
  Error    error
  StdOut   []byte
  StdErr   []byte
  Label    string
  Children Results
}
```

### Signal Handling

AVMTool handles OS signals gracefully:

- First signal: Attempt graceful shutdown of running commands
- Second signal: Force termination

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
