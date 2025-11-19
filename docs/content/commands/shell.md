+++
title = "Shell Command"
weight = 1
+++

The `shell` command executes shell commands with full environment control and configurable exit code handling.

## Attributes

### Required

- **`type: "shell"`**: Identifies this as a shell command
- **`name`**: Descriptive name for the command
- **`command_line`**: The shell command to execute

### Optional

- **`working_directory`**: Directory to execute the command in (inherits from parent if not specified)
- **`env`**: Environment variables as key-value pairs
- **`runs_on_condition`**: When to run (`success`, `error`, `always`, `exit-codes`)
- **`runs_on_exit_codes`**: Specific exit codes that trigger execution
- **`success_exit_codes`**: Exit codes indicating success (defaults to `[0]`)
- **`skip_exit_codes`**: Exit codes that skip remaining commands

## Basic Example

```yaml
name: "Simple Build"
commands:
  - type: "shell"
    name: "Build Application"
    command_line: "go build -o app ."
```

## Complete Example

```yaml
name: "Advanced Shell Command"
commands:
  - type: "shell"
    name: "Build Go Application"
    command_line: "go build -ldflags='-s -w' -o dist/app ."
    working_directory: "/path/to/project"
    env:
      CGO_ENABLED: "0"
      GOOS: "linux"
      GOARCH: "amd64"
    success_exit_codes: [0]
    skip_exit_codes: [2]
    runs_on_condition: "success"
```

## Environment Variables

Environment variables are inherited from:

1. The system environment
2. Parent command's environment
3. The command's own `env` settings (takes precedence)

```yaml
env:
  GLOBAL_VAR: "global"
commands:
  - type: "serial"
    name: "Build Process"
    env:
      BUILD_TYPE: "release"
    commands:
      - type: "shell"
        name: "Compile"
        command_line: "make build"
        env:
          OPTIMIZATION: "O3"
        # This command sees: GLOBAL_VAR, BUILD_TYPE, and OPTIMIZATION
```

## Exit Code Handling

### Success Codes

Define which exit codes indicate success:

```yaml
- type: "shell"
  name: "Tolerant Lint"
  command_line: "golangci-lint run"
  success_exit_codes: [0, 1]
  # Exit codes 0 and 1 are both considered successful
```

### Skip Codes

Exit codes that skip remaining commands in the batch:

```yaml
- type: "shell"
  name: "Check for Skip Condition"
  command_line: |
    if [ -z "$DEPLOY_KEY" ]; then
      echo "DEPLOY_KEY not set. Skipping deployment." 1>&2
      exit 99
    fi
  skip_exit_codes: [99]
```

## Using Redirection

Porch captures stdout and stderr automatically. Use redirection to control output:

```yaml
# Output to stderr
- type: "shell"
  name: "Warning Message"
  command_line: "echo 'Warning: Low disk space' 1>&2"

# Conditional output to stderr with skip
- type: "shell"
  name: "Check Prerequisites"
  command_line: |
    if [ -z "$FOO" ]; then
      echo "FOO is not set. Skipping" 1>&2
      exit 99
    fi
  skip_exit_codes: [99]
```

By default, stderr is displayed in results if the step fails or returns a skippable exit code.

## Multi-line Commands

Use YAML multi-line strings for complex commands:

```yaml
- type: "shell"
  name: "Complex Build Script"
  command_line: |
    set -e
    echo "Starting build..."
    npm install
    npm run lint
    npm run build
    npm run test
    echo "Build completed successfully"
```

## Conditional Execution

Run commands based on previous results:

```yaml
commands:
  - type: "shell"
    name: "Build"
    command_line: "make build"

  - type: "shell"
    name: "Deploy on Success"
    command_line: "./deploy.sh"
    runs_on_condition: "success"

  - type: "shell"
    name: "Cleanup on Error"
    command_line: "make clean"
    runs_on_condition: "error"

  - type: "shell"
    name: "Always Send Notification"
    command_line: "notify.sh"
    runs_on_condition: "always"
```

## Common Patterns

### Build with Environment Variables

```yaml
- type: "shell"
  name: "Cross-Platform Build"
  command_line: "go build -o dist/app-$GOOS-$GOARCH"
  env:
    CGO_ENABLED: "0"
    GOOS: "linux"
    GOARCH: "amd64"
```

### Script Execution

```yaml
- type: "shell"
  name: "Run Deployment Script"
  command_line: "./scripts/deploy.sh production"
  working_directory: "/app"
```

### Pipeline with Skip

```yaml
- type: "shell"
  name: "Check Condition"
  command_line: |
    if [ "$SKIP_TESTS" == "true" ]; then
      echo "Tests skipped by user" 1>&2
      exit 100
    fi
  skip_exit_codes: [100]

- type: "shell"
  name: "Run Tests"
  command_line: "npm test"
  # Skipped if previous command exits with 100
```

## Best Practices

1. **Use explicit exit codes**: Define `success_exit_codes` for commands with non-standard success codes
2. **Set working directory**: Use `working_directory` instead of `cd` in commands
3. **Use environment variables**: Prefer `env` over hardcoded values
4. **Redirect important messages**: Send warnings and skip messages to stderr
5. **Use multi-line for complex scripts**: Makes commands more readable
6. **Handle errors gracefully**: Use `skip_exit_codes` for optional steps

## Related

- [Flow Control](../basics/flow-control/) - Learn about skip codes and conditional execution
- [Path Inheritance](../basics/path-inheritance/) - Understand how working directories are resolved
- [Serial Command](serial/) - Execute shell commands sequentially
- [Parallel Command](parallel/) - Execute shell commands concurrently
