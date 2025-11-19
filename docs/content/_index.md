+++
title = "Porch Documentation"
description = "A Portable Process Orchestration Framework"
+++

![porch TUI](/porch/images/tui-example.png)

**porch** is a sophisticated Go-based process orchestration framework designed for running and managing complex command workflows. It provides a flexible, YAML-driven approach to define, compose, and execute command chains with advanced flow control, parallel processing, and comprehensive error handling.

It was designed to solve the problem of orchestrating complex command workflows in a portable and easy-to-use manner. Its portability means that you can confidently run the same workflows locally, in CI/CD pipelines, or on any server without worrying about compatibility issues.

## Features at a Glance

### 🚀 Command Orchestration

- **Serial Execution**: Run commands sequentially with dependency management
- **Parallel Execution**: Execute independent commands concurrently for optimal performance
- **Nested Batches**: Compose complex workflows with serial and parallel command combinations
- **Shell Commands**: Execute any shell command with full environment control
- **Directory Iteration**: Execute commands across multiple directories with flexible traversal options

### 📋 Flexible Configuration

- **YAML-Based Workflows**: Define command chains using simple, readable YAML syntax
- **Working Directory Management**: Control execution context with per-command working directories
- **Environment Variables**: Set and inherit environment variables at any level
- **Conditional Execution**: Run commands based on success, failure, or specific exit codes
- **Command Groups**: Define reusable command sets that can be referenced by container commands

### 🛡️ Robust Error Handling

- **Graceful Signal Handling**: Proper SIGINT/SIGTERM handling with graceful shutdown
- **Context-Aware Execution**: Full context propagation for cancellation and timeouts
- **Comprehensive Results**: Detailed execution results with exit codes, stdout, and stderr
- **Error Aggregation**: Collect and report errors across complex command hierarchies
- **Skip Controls**: Configure commands to skip remaining tasks based on exit codes

### 🎨 Beautiful Output

- **TUI**: Real-time terminal user interface for live command progress monitoring
- **Tree Visualization**: Clear hierarchical display of command execution
- **Colorized Output**: Terminal-aware colored output for better readability
- **Structured Results**: JSON and pretty-printed result formatting
- **Progress Tracking**: Real-time execution status and progress indication

## Quick Start

### Installation

#### Operating System Support

Porch currently supports the following operating systems:

- Linux
- macOS
- Windows

#### Install from Go modules

```bash
go install github.com/matt-FFFFFF/porch@latest
```

#### Build from source

```bash
git clone https://github.com/matt-FFFFFF/porch.git
cd porch
make build
```

### Basic Workflow

Create a YAML file defining your command workflow:

```yaml
name: "Build and Test Workflow"
description: "Complete CI/CD pipeline example"
commands:
  - type: "shell"
    name: "Setup Environment"
    command_line: "echo 'Setting up build environment...'"

  - type: "parallel"
    name: "Quality Checks"
    commands:
      - type: "shell"
        name: "Run Tests"
        command_line: "go test ./..."

      - type: "shell"
        name: "Run Linter"
        command_line: "golangci-lint run"

  - type: "serial"
    name: "Build and Package"
    commands:
      - type: "shell"
        name: "Build Application"
        command_line: "go build -o app ."
```

### Execute the workflow

```bash
porch run -f workflow.yaml
```

## What's Next?

- **[Getting Started](basics/)** - Learn the fundamentals of Porch
- **[Commands](commands/)** - Detailed documentation for each command type
- **[Output Control](output/)** - Configure logging and output options
- **[TUI](tui/)** - Interactive terminal user interface

## License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/matt-FFFFFF/porch/blob/main/LICENSE) file for details.
