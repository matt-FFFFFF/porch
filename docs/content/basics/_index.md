+++
title = "Getting Started"
weight = 1
+++

This section covers the fundamental concepts you need to understand before building workflows with Porch.

## Why Use Porch?

Porch was designed to solve several common challenges in command orchestration:

### Portability

Run the same workflows across different environments:

- **Local Development**: Test and debug workflows on your local machine
- **CI/CD Pipelines**: Execute workflows in continuous integration environments
- **Production Servers**: Deploy and run workflows consistently across servers

Unlike shell scripts that depend on specific shell implementations or system utilities, Porch provides a consistent execution environment.

### Simplicity

Define complex workflows using simple, readable YAML:

- No need to learn complex scripting languages
- Clear, declarative syntax
- Easy to version control and review
- Self-documenting workflow definitions

### Reliability

Built-in error handling and flow control:

- Graceful shutdown on interruption (SIGINT/SIGTERM)
- Conditional execution based on exit codes
- Comprehensive error reporting
- Skip controls for non-critical failures

### Visibility

Real-time monitoring and detailed results:

- Interactive TUI for live progress tracking
- Hierarchical tree visualization of command execution
- Detailed execution metrics and timing
- Structured output in JSON format

## Core Concepts

### Workflows

A workflow is a collection of commands defined in a YAML file. Each workflow has:

- **Name**: A descriptive name for the workflow
- **Description**: Optional explanation of what the workflow does
- **Commands**: List of commands to execute
- **Command Groups**: Optional reusable command sets

Example:

```yaml
name: "My Workflow"
description: "A simple example workflow"
commands:
  - type: "shell"
    name: "Hello World"
    command_line: "echo 'Hello, World!'"
```

The top level commands in a workflow are executed serially using the current directory as the working directory.
They follow the same rules as a [`serial`](commands/serial) command (because that's what they are under the hood).

### Commands

Commands are the building blocks of workflows. Porch supports several command types:

- **shell**: Execute shell commands
- **pwsh**: Execute PowerShell scripts
- **serial**: Run commands sequentially
- **parallel**: Run commands concurrently
- **foreachdirectory**: Execute commands in multiple directories
- **copycwdtotemp**: Copy working directory to a temporary location

Each command type has specific attributes and behaviors. See the [Commands](../commands/) section for details.

### Execution Flow

Commands execute in the order defined, with support for:

- **Sequential Execution**: Commands run one after another (serial)
- **Parallel Execution**: Commands run simultaneously (parallel)
- **Nested Batches**: Combine serial and parallel execution for complex flows
- **Conditional Execution**: Run commands based on previous results

## Next Steps

Learn about the key features that make Porch powerful:

- [Path Inheritance](path-inheritance/) - How working directories are resolved
- [Flow Control](flow-control/) - Skipping commands and handling errors
