+++
title = "Terminal User Interface (TUI)"
weight = 4
+++

# Terminal User Interface (TUI)

Porch includes a real-time Terminal User Interface (TUI) that provides live progress monitoring during command execution. The TUI offers better visibility into complex workflows compared to traditional text-based output.

![TUI Example](/images/tui-example.png)

## Enabling the TUI

Use one of these flags to enable the TUI:

```bash
# Using --tui flag
porch run -f workflow.yaml --tui

# Using -t shorthand
porch run -f workflow.yaml -t

# Using --interactive flag
porch run -f workflow.yaml --interactive
```

## Features

### Real-Time Command Tree

The TUI displays a hierarchical tree structure showing:
- **Command names**: Clear identification of each command
- **Status indicators**: Visual icons showing command state
- **Execution timing**: Live elapsed time for running and completed commands
- **Progress updates**: Real-time progress as commands execute

### Status Indicators

| Icon | Status | Description |
|------|--------|-------------|
| ⏳ | Pending | Command waiting to execute |
| ⚡ | Running | Command currently executing |
| ✅ | Success | Command completed successfully |
| ❌ | Failed | Command failed with error |
| ⊘ | Skipped | Command intentionally skipped |

### Keyboard Controls

| Key | Action |
|-----|--------|
| `q` | Quit the TUI and stop workflow |
| `r` | Refresh the display |
| `↑`/`↓` | Scroll through command tree (if needed) |

## Example TUI Output

```
Build Pipeline                                                    ⚡ 5.2s

  ✅ Setup Environment                                           0.1s
  ⚡ Quality Checks                                              3.8s
    ⚡ Unit Tests                                                2.1s
    ✅ Linting                                                   1.2s
    ⚡ Security Scan                                             0.5s
  ⏳ Build and Package
    ⏳ Build Application
    ⏳ Create Archive

[q] Quit  [r] Refresh
```

## When to Use the TUI

### Best Use Cases

**Long-running workflows**: Monitor progress of builds, tests, or deployments
```bash
porch run -f ci-pipeline.yaml --tui
```

**Complex parallel execution**: See which parallel tasks are still running
```bash
porch run -f parallel-tests.yaml --tui
```

**Interactive debugging**: Watch workflow execution in real-time
```bash
porch run -f workflow.yaml --tui
```

**Development**: Get immediate feedback during workflow development
```bash
porch run -f new-workflow.yaml --tui
```

### When NOT to Use the TUI

**CI/CD pipelines**: Use standard output for log collection
```bash
# CI/CD - Don't use --tui
NO_COLOR=1 porch run -f workflow.yaml --out results
```

**Scripting**: When output needs to be parsed
```bash
# Scripting - Don't use --tui  
porch run -f workflow.yaml --out results.json
```

**Background jobs**: TUI requires interactive terminal
```bash
# Background - Don't use --tui
nohup porch run -f workflow.yaml --out results &
```

## Architecture

The TUI is built on the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework and uses a sophisticated event-driven architecture:

### Progress Event System

```
Commands → Reporter → Events → TUI Display
```

1. **Commands**: Execute while emitting progress events
2. **Reporter**: Thread-safe event routing via channels
3. **Events**: Structured event data (started, progress, output, completed, failed)
4. **TUI Display**: Real-time tree view updates

### Event Types

The TUI responds to several event types:

- **Started**: Command begins execution
- **Progress**: Command reports progress update
- **Output**: Command produces stdout/stderr
- **Completed**: Command finishes successfully
- **Failed**: Command fails with error
- **Skipped**: Command intentionally skipped

## Progressive Commands

Commands that support real-time progress reporting:

### Progressive Shell Commands

Shell commands report when they start and complete:

```yaml
- type: "shell"
  name: "Long Running Build"
  command_line: "npm run build"
  # TUI shows: ⏳ → ⚡ Running (with timer) → ✅ or ❌
```

### Progressive Batch Commands

Serial and parallel batches propagate progress from child commands:

```yaml
- type: "parallel"
  name: "Run Tests"
  commands:
    - type: "shell"
      name: "Unit Tests"
      command_line: "npm test"
    - type: "shell"
      name: "Integration Tests"
      command_line: "npm run test:integration"
  # TUI shows both tests running concurrently with individual timers
```

## Graceful Fallback

If the TUI cannot initialize (e.g., not a TTY, terminal not supported):
- Porch automatically falls back to standard output mode
- No error is raised
- Workflow continues normally with text output

```bash
# TUI fallback example
porch run -f workflow.yaml --tui 2>&1 | tee log.txt
# Falls back to standard output because stdout is redirected
```

## Resource Management

The TUI properly manages resources:

- **Goroutines**: Cleaned up on exit
- **Channels**: Properly closed after use
- **Terminal**: Restored to normal mode on exit
- **Context**: Respects cancellation signals

### Clean Shutdown

Press `q` or `Ctrl+C` to quit:
- TUI closes gracefully
- Running commands receive cancellation signal
- Terminal is restored to normal state
- Partial results are displayed

## Combining with Output Flags

TUI can be combined with result saving:

```bash
# Use TUI for monitoring, save results for later
porch run -f workflow.yaml --tui --out results

# Review results later without TUI
porch show results --stdout --success
```

## Thread Safety

The TUI implementation is thread-safe:
- Progress reporters use proper synchronization
- Command tree updates are mutex-protected
- All tests pass with `-race` flag

## Example Workflows

### Development Workflow

```bash
# Watch tests run in real-time
porch run -f test-workflow.yaml --tui
```

### CI/CD Pipeline Monitoring

```bash
# Monitor deployment locally
porch run -f deploy.yaml --tui

# In CI/CD (no TUI)
porch run -f deploy.yaml --out deployment-results
```

### Debugging Failed Workflow

```bash
# Run with TUI to see where it fails
porch run -f broken-workflow.yaml --tui

# See which command fails in real-time
# Press 'q' to quit if needed
```

## Best Practices

1. **Use for interactive sessions**: Perfect for local development and debugging
2. **Skip in CI/CD**: Use standard output mode for automated environments  
3. **Save results**: Combine `--tui` with `--out` to save results
4. **Monitor long workflows**: TUI is ideal for workflows that take several minutes
5. **Watch parallel execution**: See which parallel tasks complete first

## Limitations

1. **Terminal required**: Must run in an interactive terminal
2. **No piping**: Output redirection disables TUI (fallback to standard mode)
3. **Screen space**: Very large command trees may require scrolling
4. **Single workflow**: TUI shows one workflow at a time

## Technical Details

### Dependencies

The TUI uses these libraries:
- **Bubble Tea**: TUI framework
- **Lipgloss**: Terminal styling
- **Charmbracelet**: Terminal utilities

### Implementation

Key components:
- `internal/tui/model.go`: Main TUI state and Bubble Tea model
- `internal/tui/runner.go`: Orchestrates TUI and command execution
- `internal/progress/`: Progress event system
- `internal/runbatch/progressive.go`: Progressive command implementations

### Testing

Comprehensive test coverage:
```bash
# Run TUI tests
go test ./internal/tui/...
go test ./internal/progress/...

# Run with race detector
go test -race ./internal/...
```

## Backward Compatibility

The TUI is fully backward compatible:
- Existing workflows work unchanged
- TUI is opt-in via flags
- Non-TUI mode behavior is identical
- Progressive features are additive

## Related

- [Output Control](../output/) - Configure logging and output
- [Flow Control](../basics/flow-control/) - Understanding command execution
- [Commands](../commands/) - All command types support the TUI
