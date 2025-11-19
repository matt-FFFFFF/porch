+++
title = "Parallel Command"
weight = 4
+++

The `parallel` command executes a list of commands concurrently, allowing independent tasks to run simultaneously for optimal performance.

## Attributes

### Required

- **`type: "parallel"`**: Identifies this as a parallel batch command
- **`name`**: Descriptive name for the command batch

### Optional

- **`working_directory`**: Directory inherited by all child commands
- **`env`**: Environment variables inherited by all child commands
- **`runs_on_condition`**: When to run (`success`, `error`, `always`, `exit-codes`)
- **`runs_on_exit_codes`**: Specific exit codes that trigger execution
- **`commands`**: List of commands to execute (either this or `command_group`)
- **`command_group`**: Reference to a named command group (either this or `commands`)

## Basic Example

```yaml
name: "Parallel Tests"
commands:
  - type: "parallel"
    name: "Run All Tests"
    commands:
      - type: "shell"
        name: "Unit Tests"
        command_line: "go test ./..."

      - type: "shell"
        name: "Linting"
        command_line: "golangci-lint run"

      - type: "shell"
        name: "Security Scan"
        command_line: "gosec ./..."
```

## Execution Flow

In parallel execution:

1. All commands start simultaneously
2. Commands run independently of each other
3. The batch waits for all commands to complete
4. If any command fails, the batch is marked as failed
5. All commands run to completion regardless of individual failures

```yaml
- type: "parallel"
  name: "Quality Checks"
  commands:
    - type: "shell"
      name: "Test Suite 1"
      command_line: "test-suite-1.sh"
      # Starts immediately

    - type: "shell"
      name: "Test Suite 2"
      command_line: "test-suite-2.sh"
      # Starts immediately (concurrent with Suite 1)

    - type: "shell"
      name: "Test Suite 3"
      command_line: "test-suite-3.sh"
      # Starts immediately (concurrent with Suite 1 & 2)
```

## Use Cases

Parallel execution is ideal for:

- **Independent tests**: Unit tests, integration tests, E2E tests
- **Quality checks**: Linting, security scans, code coverage
- **Multi-platform builds**: Building for different OS/architectures
- **Concurrent operations**: Tasks that don't depend on each other

```yaml
name: "Quality Assurance"
commands:
  - type: "parallel"
    name: "Run All Checks"
    commands:
      - type: "shell"
        name: "Unit Tests"
        command_line: "npm run test:unit"

      - type: "shell"
        name: "Integration Tests"
        command_line: "npm run test:integration"

      - type: "shell"
        name: "ESLint"
        command_line: "npm run lint"

      - type: "shell"
        name: "TypeScript Check"
        command_line: "npm run type-check"

      - type: "shell"
        name: "Security Audit"
        command_line: "npm audit"
```

## Error Handling

All commands complete even if some fail:

```yaml
- type: "parallel"
  name: "Best Effort Tests"
  commands:
    - type: "shell"
      name: "Test 1"
      command_line: "test1.sh"
      # Continues even if other tests fail

    - type: "shell"
      name: "Test 2"
      command_line: "test2.sh"
      # Continues even if other tests fail

# Next command runs after all parallel commands complete
- type: "shell"
  name: "Generate Report"
  command_line: "test-report.sh"
  runs_on_condition: "always"
  # Runs whether parallel batch succeeded or failed
```

## Environment and Working Directory

Child commands inherit from the parallel batch:

```yaml
- type: "parallel"
  name: "Cross-Platform Builds"
  working_directory: "./dist"
  env:
    BUILD_VERSION: "1.0.0"
  commands:
    - type: "shell"
      name: "Linux Build"
      command_line: "build-linux.sh"
      env:
        GOOS: "linux"
      # Runs in ./dist with BUILD_VERSION and GOOS

    - type: "shell"
      name: "macOS Build"
      command_line: "build-macos.sh"
      env:
        GOOS: "darwin"
      # Runs in ./dist with BUILD_VERSION and GOOS
```

## Nested Batches

Combine parallel and serial execution:

```yaml
- type: "parallel"
  name: "Multi-Service Pipeline"
  commands:
    - type: "serial"
      name: "Frontend Pipeline"
      commands:
        - type: "shell"
          name: "Build Frontend"
          command_line: "npm run build"
        - type: "shell"
          name: "Test Frontend"
          command_line: "npm test"

    - type: "serial"
      name: "Backend Pipeline"
      commands:
        - type: "shell"
          name: "Build Backend"
          command_line: "go build"
        - type: "shell"
          name: "Test Backend"
          command_line: "go test ./..."
```

## Performance Considerations

Parallel execution provides performance benefits when:

- Tasks are I/O bound (network, disk)
- Tasks are independent (no shared state)
- System has multiple CPU cores

```yaml
# Fast: Parallel independent tasks
- type: "parallel"
  name: "Independent Downloads"
  commands:
    - type: "shell"
      name: "Download A"
      command_line: "wget url-a"
    - type: "shell"
      name: "Download B"
      command_line: "wget url-b"

# Better as serial: Dependent tasks
- type: "serial"
  name: "Dependent Tasks"
  commands:
    - type: "shell"
      name: "Build"
      command_line: "make build"
    - type: "shell"
      name: "Test"
      command_line: "make test" # Needs build output
```

## Best Practices

1. **Use for independent tasks**: Ensure tasks don't depend on each other
2. **Limit concurrency**: Don't run too many resource-intensive tasks simultaneously
3. **Handle all errors**: Use `runs_on_condition: "always"` for cleanup
4. **Consider system resources**: Balance parallelism with available CPU/memory
5. **Use descriptive names**: Clear names help identify which task failed

## Related

- [Serial Command](serial/) - Execute commands sequentially
- [Flow Control](../basics/flow-control/) - Conditional execution
- [ForEach Directory](foreachdirectory/) - Parallel directory iteration
