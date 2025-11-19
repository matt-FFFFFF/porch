+++
title = "Output Control"
weight = 3
+++

Porch provides several ways to control the output and logging behavior of your workflows. This includes environment variables for logging, command-line flags for output formatting, and color control.

## Log Levels

Porch uses structured logging with different log levels. Control the log level using the `PORCH_LOG_LEVEL` environment variable.

### Available Log Levels

| Level   | Description                    | When to Use                       |
| ------- | ------------------------------ | --------------------------------- |
| `DEBUG` | Detailed debugging information | Development and troubleshooting   |
| `INFO`  | General informational messages | Normal operation (default)        |
| `WARN`  | Warning messages               | Important but non-critical issues |
| `ERROR` | Error messages                 | Error conditions only             |

### Setting Log Level

```bash
# Set log level via environment variable
export PORCH_LOG_LEVEL=DEBUG
porch run -f workflow.yaml

# Or inline
PORCH_LOG_LEVEL=DEBUG porch run -f workflow.yaml

# Production: Only errors
PORCH_LOG_LEVEL=ERROR porch run -f workflow.yaml
```

### Example Output by Level

**DEBUG level** (most verbose):

```
DEBUG: Starting workflow execution
DEBUG: Creating command: Build Application
DEBUG: command info path=/usr/bin/go cwd=/project args=[build -o app .]
INFO: Running command: Build Application
DEBUG: Command completed with exit code 0
```

**INFO level** (default):

```
INFO: Running command: Build Application
INFO: Command completed successfully
```

**ERROR level** (least verbose):

```
ERROR: Command failed: Build Application
```

## Stdout and Stderr

By default, Porch captures stdout and stderr from all commands. You can control what appears in the results using command-line flags.

### Command-Line Flags

```bash
# Include stdout in results
porch run -f workflow.yaml --output-stdout
porch run -f workflow.yaml --stdout

# Exclude stderr from results
porch run -f workflow.yaml --no-output-stderr
porch run -f workflow.yaml --no-stderr

# Include details for successful commands
porch run -f workflow.yaml --output-success-details
porch run -f workflow.yaml --success

# Combine flags
porch run -f workflow.yaml --stdout --success --no-stderr
```

### Default Behavior

By default:

- **Stdout**: Not included in output (unless `--stdout` is specified)
- **Stderr**: Included for failed commands and commands with skip exit codes
- **Success details**: Not included for successful commands (unless `--success` is specified)

### Example: Showing Stdout

```bash
# Without --stdout
$ porch run -f workflow.yaml
✓ Build Application (1.2s)

# With --stdout
$ porch run -f workflow.yaml --stdout
✓ Build Application (1.2s)
  stdout:
    Building main.go...
    Build completed successfully
```

### Example: Hiding Stderr

```bash
# Default (stderr shown for errors)
$ porch run -f workflow.yaml
✗ Linter (0.5s) - exit code 1
  stderr: main.go:15: unused variable 'x'

# With --no-stderr
$ porch run -f workflow.yaml --no-stderr
✗ Linter (0.5s) - exit code 1
```

### Example: Showing Success Details

```bash
# Default (minimal output for success)
$ porch run -f workflow.yaml
✓ Tests (5.2s)

# With --success
$ porch run -f workflow.yaml --success
✓ Tests (5.2s)
  exit code: 0
  duration: 5.234s
  stderr:
    PASS
    coverage: 85.3% of statements
```

## Color Output

Porch automatically detects terminal capabilities and displays colored output when supported.

### Controlling Color

```bash
# Force color output
porch run -f workflow.yaml

# Disable color output
NO_COLOR=1 porch run -f workflow.yaml

# Explicitly enable color (even if not a TTY)
FORCE_COLOR=1 porch run -f workflow.yaml
```

### Color Indicators

Porch uses colors to indicate command status:

- **Green (✓)**: Successful commands
- **Red (✗)**: Failed commands
- **Yellow (⚠)**: Commands with warnings or non-zero success codes
- **Gray**: Skipped commands
- **Blue**: Running commands (in TUI)

### Disabling Color for CI/CD

Many CI/CD systems don't support ANSI color codes. Disable color:

```yaml
# GitHub Actions
- name: Run Porch
  run: NO_COLOR=1 porch run -f workflow.yaml

# GitLab CI
script:
  - export NO_COLOR=1
  - porch run -f workflow.yaml
```

## Saving Results

Save workflow results to a file for later review:

```bash
# Save results
porch run -f workflow.yaml --out results

# View saved results
porch show results

# View with all options
porch show results --stdout --success
```

### Results File Format

Results are saved as [gob](https://pkg.go.dev/encoding/gob) files containing:

- Command hierarchy
- Exit codes
- Execution duration
- Stdout and stderr output
- Environment variables
- Working directories

## Output Formats

### Tree View (Default)

```
✓ Build and Test Workflow (10.5s)
  ✓ Setup Environment (0.1s)
  ✓ Quality Checks (5.2s)
    ✓ Run Tests (4.8s)
    ✓ Run Linter (2.1s)
  ✓ Build Process (5.2s)
    ✓ Build for Linux (2.1s)
    ✓ Build for macOS (1.8s)
```

### Tree View with Details

You can add additional details to the output using the `--show-details/--details` flag.
Doing this will show information such as exit codes, the cwd, and the command type.,

```bash
porch run --details -f workflow.yaml
```

```text
✓ Skip test (type: SerialBatch) (cwd: .)
  ✓ echo pwd (type: OSCommand) (cwd: .)
  ✓ skip test (type: OSCommand) (cwd: .) (exit code: 2)
    ➜ Error: intentionally skip execution
  ~ should not run (type: OSCommand) (cwd: .)
    ➜ Error: intentionally skip execution
```

## Redirecting Command Output

Within commands, use shell redirection to control output:

```yaml
# Send to stderr
- type: "shell"
  name: "Warning"
  command_line: "echo 'Warning message' 1>&2"

# Suppress all output
- type: "shell"
  name: "Silent Command"
  command_line: "command >/dev/null 2>&1"

# Redirect to file
- type: "shell"
  name: "Save Log"
  command_line: "command > output.log 2>&1"

# Conditional output to stderr with skip
- type: "shell"
  name: "Check Condition"
  command_line: |
    if [ -z "$VAR" ]; then
      echo "VAR not set, skipping" 1>&2
      exit 99
    fi
  skip_exit_codes: [99]
```

## Practical Examples

### Verbose Development Mode

```bash
# Maximum verbosity for debugging
PORCH_LOG_LEVEL=DEBUG porch run -f workflow.yaml --stdout --success
```

### Clean CI/CD Output

```bash
# Minimal output for CI/CD
NO_COLOR=1 PORCH_LOG_LEVEL=ERROR porch run -f workflow.yaml --no-stderr --out results
```

### Detailed Results Review

```bash
# Save results with all details
porch run -f workflow.yaml --out results --stdout --success

# Review later with color
porch show results --stdout --success
```

### Production Deployment

```bash
# Only errors, save results for audit
NO_COLOR=1 PORCH_LOG_LEVEL=ERROR porch run -f workflow.yaml --out deploy-results --no-stderr
```

## Best Practices

1. **Use PORCH_LOG_LEVEL=DEBUG for development**: Get detailed information during development
2. **Use PORCH_LOG_LEVEL=ERROR for production**: Reduce noise in production logs
3. **Save results in CI/CD**: Keep audit trail with `--out results`
4. **Disable color in CI/CD**: Use `NO_COLOR=1` to avoid escape sequences
5. **Use --stdout for debugging**: See command output when troubleshooting
6. **Use --success for detailed analysis**: Review timing and exit codes
7. **Redirect important messages to stderr**: Makes them visible in default output

## Environment Variables Summary

| Variable          | Values                           | Default | Purpose                   |
| ----------------- | -------------------------------- | ------- | ------------------------- |
| `PORCH_LOG_LEVEL` | `DEBUG`, `INFO`, `WARN`, `ERROR` | `INFO`  | Control logging verbosity |
| `NO_COLOR`        | `1` or any value                 | Not set | Disable color output      |
| `FORCE_COLOR`     | `1` or any value                 | Not set | Force color output        |

## Command-Line Flags Summary

| Flag                       | Shorthand     | Description                             |
| -------------------------- | ------------- | --------------------------------------- |
| `--output-stdout`          | `--stdout`    | Include stdout in results               |
| `--no-output-stderr`       | `--no-stderr` | Exclude stderr from results             |
| `--output-success-details` | `--success`   | Include details for successful commands |
| `--out <file>`             |               | Save results to file                    |

## Related

- [TUI](../tui/) - Interactive terminal user interface
- [Flow Control](../basics/flow-control/) - Understanding skip codes and errors
- [Shell Command](../commands/shell/) - Using redirection in commands
