+++
title = "TUI Implementation Details"
weight = 2
+++

## Overview

The Porch orchestration tool now includes a real-time Terminal User Interface (TUI) that provides live progress monitoring during command execution. This enhancement offers better visibility into complex workflows compared to traditional text-based output.

## Features Implemented

### ✅ Core TUI Infrastructure

- **Event-driven progress system**: Real-time progress events flow from executing commands to the TUI
- **Channel-based reporters**: Thread-safe progress reporting with proper resource cleanup
- **Hierarchical progress contexts**: Commands can report progress with full command path context

### ✅ Interactive TUI Display

- **Real-time command tree**: Shows the hierarchical structure of executing commands
- **Status indicators**: Visual icons for command states (⏳ pending, ⚡ running, ✅ success, ❌ failed)
- **Live timing**: Shows elapsed time for running and completed commands
- **Keyboard controls**: 'q' to quit, 'r' to refresh

### ✅ Progressive Command Support

- **Progressive shell commands**: Shell commands now support real-time progress reporting
- **Progressive batch commands**: Serial and parallel batches propagate progress events
- **Backward compatibility**: Existing commands work unchanged; TUI is opt-in

### ✅ CLI Integration

- **TUI flag**: `--tui`, `-t`, or `--interactive` enables the TUI mode
- **Graceful fallback**: If TUI can't initialize, falls back to standard mode
- **Clean shutdown**: Proper cleanup of TUI and progress resources

## Usage

### Basic TUI Usage

```bash
# Run with TUI
./porch run -f config.yaml --tui

# Alternative flags
./porch run -f config.yaml -t
./porch run -f config.yaml --interactive

# Standard mode (unchanged)
./porch run -f config.yaml
```

### Example YAML Configuration

The TUI works with any existing Porch YAML configuration. Here's an example that showcases the TUI features:

```yaml
name: Build Pipeline
description: Demonstration of TUI capabilities
commands:
  - type: serial
    name: Main Pipeline
    commands:
      - type: shell
        name: Setup
        command_line: echo "Setting up..." && sleep 1

      - type: parallel
        name: Quality Checks
        commands:
          - type: shell
            name: Linting
            command_line: echo "Running linter..." && sleep 2

          - type: shell
            name: Unit Tests
            command_line: echo "Running tests..." && sleep 3

      - type: shell
        name: Build
        command_line: echo "Building..." && sleep 2
```

## Architecture

### Progress Event System

```text
Commands → Reporter → Events → TUI Display
```

1. **Commands**: Execute while emitting progress events
2. **Reporter**: Thread-safe event routing
3. **Events**: Structured event data (started, progress, output, completed, failed)
4. **TUI Display**: Real-time tree view with bubbletea framework

### Key Components

#### Progress Events (`internal/progress/`)

- `Event`: Core event structure with command path, type, and data
- `Reporter`: Interface for sending events
- `ChannelReporter`: Channel-based implementation for TUI
- `NullReporter`: No-op implementation for standard mode
- `TransparentReporter`: Pass-through for existing commands

#### TUI System (`internal/tui/`)

- `Model`: Main TUI state and bubbletea model
- `CommandNode`: Tree structure for command hierarchy
- `Runner`: Orchestrates TUI and command execution
- `TUIReporter`: Bridges progress events to TUI updates

#### Progressive Commands (`internal/runbatch/`)

- `ProgressiveRunnable`: Interface for progress-aware commands
- `ProgressiveOSCommand`: Shell commands with progress reporting
- `ProgressiveSerialBatch`: Serial execution with progress
- `ProgressiveParallelBatch`: Parallel execution with progress

## Testing

The implementation includes comprehensive tests:

```bash
# Run all tests
go test ./internal/...

# Run with race detector
go test ./internal/... -race

# Test specific packages
go test ./internal/tui/...
go test ./internal/progress/...
go test ./internal/runbatch/...
```

## Examples

### Complex Workflow

```bash
./porch run -f examples/tui-demo.yaml --tui
```

## Implementation Status

### ✅ Completed

- Core progress event infrastructure
- TUI display with command tree and status indicators
- Progressive shell command implementation
- Progressive serial and parallel batch support
- CLI integration with TUI flag
- Thread-safe progress reporting
- Comprehensive test coverage

## Technical Details

### Thread Safety

- All progress reporters use proper synchronization
- Command tree updates are mutex-protected
- Race detector passes on all tests

### Resource Management

- Proper cleanup of goroutines and channels
- Graceful shutdown on context cancellation
- TUI resources cleaned up on exit

### Backward Compatibility

- Existing configurations work unchanged
- Non-TUI mode behavior is identical
- Progressive features are additive

## Dependencies Added

- `github.com/charmbracelet/bubbletea`: TUI framework
- `github.com/charmbracelet/lipgloss`: Styling for TUI

The TUI implementation provides a solid foundation for real-time command monitoring while maintaining full backward compatibility with existing Porch workflows.
