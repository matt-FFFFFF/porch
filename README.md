# AVMTool - A Powerful Command Orchestration Framework

AVMTool is a sophisticated command orchestration framework for running and managing complex command workflows. It provides a flexible way to define, compose, and execute command chains with sophisticated flow control, parallel processing, and comprehensive error handling.

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
  - Working directory management

## Installation

```bash
go install github.com/matt-FFFFFF/pporch@latest
```

Or build from source:

```bash
git clone https://github.com/matt-FFFFFF/pporch.git
cd avmtool
go build -o avmtool
```

## Quick Start

### Basic Command Chain

Create a YAML file defining your command chain:

```yaml
name: "Complex Workflow Example"
description: "Example showing nested serial and parallel commands"
commands:
  - type: "shell"
    name: "Setup"
    command_line: "echo banana"

  - type: "parallel"
    name: "Parallel Tasks"
    commands:
      - type: "shell"
        name: "Task 1"
        command_line: "echo Task 1 running"

      - type: "shell"
        name: "Task 3"
        command_line: '>&2 echo "an error message" && exit 2 '

      - type: "shell"
        name: "Task 2"
        command_line: "echo Task 2 running"

  - type: "serial"
    name: "Copy to tmp demo"
    commands:
      - type: "copycwdtotemp"
        cwd: "."
      - type: "shell"
        name: "List files"
        command_line: "ls -l"
      - type: "shell"
        name: "Print working directory"
        command_line: "pwd"

```

Run it with:

```bash
avmtool run workflow.yaml
```

## Command Types

AVMTool supports several command types:

### 1. Basic Command

Executes a single command:

```yaml
- type: "shell"
  name: "Build Project"
  command_line: "make build"
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

## Extending AVMTool

### Creating Custom Commands

Implement the `Commander` interface:

```go
// Commander is an interface for converting commands into runnables.
type Commander interface {
 // Create creates a runnable command from the provided payload.
 // The payload is the YAML command in bytes.
 Create(ctx context.Context, payload []byte) (runbatch.Runnable, error)
}
```

## Advanced Examples

### Working Directory Management

```yaml
- type: "serial"
  name: "Work in Temp Directory"
  commands:
    - type: "copycwdtotemp"
    - type: "shell"
      name: "Run in Temp"
      command_line: "pwd"
```

### Environment Variable Substitution

```yaml
- type: "foreach"
  name: "Process Environments"
  itemProvider: "comma-separated"  # Returns: "dev,staging,prod"
  itemVariable: "ENV"
  commands:
    - type: "shell"
      name: "Deploy to ${ENV}"
      command_line: "oscommand"
```

## Core Concepts

### Runnable Interface

The foundation of AVMTool is the `Runnable` interface, which defines anything that can be executed:

```go
// Runnable is an interface for something that can be run as part of a batch (either a Command or a nested Batch).
type Runnable interface {
 // Run executes the command or batch and returns the results.
 // It should handle context cancellation and passing signals to any spawned process.
 Run(context.Context) Results
 // SetCwd sets the working directory for the command or batch.
 // It should be called before Run() to ensure the command or batch runs in the correct directory.
 SetCwd(string)
 // InheritEnv sets the environment variables for the command or batch.
 // It should not overwrite the existing environment variables, but rather add to them.
 InheritEnv(map[string]string)
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
