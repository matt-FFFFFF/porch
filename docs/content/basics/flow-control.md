+++
title = "Flow Control"
weight = 3
+++

# Flow Control

Porch provides sophisticated flow control mechanisms to handle complex execution scenarios, including conditional execution, error handling, and intentional skipping.

## Conditional Execution

Every command in Porch can specify when it should run using the `runs_on_condition` attribute. This allows you to create workflows that adapt to the results of previous commands.

### Run Conditions

There are four run conditions available:

| Condition | Description |
|-----------|-------------|
| `success` | Run only if all previous commands succeeded (default) |
| `error` | Run only if a previous command failed |
| `always` | Run regardless of previous command results |
| `exit-codes` | Run only if previous command exited with specific codes |

### Example: Basic Conditions

```yaml
name: "Conditional Workflow"
commands:
  - type: "shell"
    name: "Primary Task"
    command_line: "make build"
    # runs_on_condition defaults to "success"

  - type: "shell"
    name: "Success Handler"
    command_line: "echo 'Build succeeded, deploying...'"
    runs_on_condition: "success"

  - type: "shell"
    name: "Error Handler"
    command_line: "echo 'Build failed, cleaning up...'"
    runs_on_condition: "error"

  - type: "shell"
    name: "Always Cleanup"
    command_line: "rm -rf temp/"
    runs_on_condition: "always"
```

### Exit Code Conditions

For fine-grained control, use `exit-codes` condition with `runs_on_exit_codes`:

```yaml
name: "Exit Code Handling"
commands:
  - type: "shell"
    name: "Check Database"
    command_line: "db-check.sh"

  - type: "shell"
    name: "Handle Warning"
    command_line: "echo 'Database has warnings'"
    runs_on_condition: "exit-codes"
    runs_on_exit_codes: [1, 2]  # Run on warning codes

  - type: "shell"
    name: "Handle Critical Error"
    command_line: "echo 'Critical database error'"
    runs_on_condition: "exit-codes"
    runs_on_exit_codes: [3, 4, 5]  # Run on critical codes
```

## Skip Controls

Porch allows commands to intentionally skip remaining tasks in the current batch using **skip exit codes**. This is useful for optional steps or early termination scenarios.

### Skip Exit Codes

Define exit codes that should skip remaining commands using `skip_exit_codes`:

```yaml
name: "Optional Steps"
commands:
  - type: "serial"
    name: "Build Pipeline"
    commands:
      - type: "shell"
        name: "Check Prerequisites"
        command_line: |
          if [ -z "$DEPLOY_KEY" ]; then
            echo "DEPLOY_KEY not set. Skipping deployment." 1>&2
            exit 99
          fi
        skip_exit_codes: [99]

      - type: "shell"
        name: "Deploy"
        command_line: "deploy.sh"
        # This will be skipped if previous command exits with 99
```

When a command exits with a skip code:
1. The command is marked as intentionally skipped
2. Remaining commands in the same batch are skipped
3. The workflow continues with commands at the next level
4. The overall workflow does not fail

### Practical Skip Example

```yaml
name: "Feature Flag Workflow"
commands:
  - type: "shell"
    name: "Check Feature Flag"
    command_line: |
      if [ "$ENABLE_FEATURE_X" != "true" ]; then
        echo "Feature X is disabled. Skipping tests." 1>&2
        exit 100
      fi
    skip_exit_codes: [100]

  - type: "shell"
    name: "Run Feature X Tests"
    command_line: "npm run test:feature-x"
    # Skipped if feature flag is not set

  - type: "shell"
    name: "Deploy Feature X"
    command_line: "deploy-feature-x.sh"
    # Also skipped if feature flag is not set
```

## Success Exit Codes

By default, only exit code `0` indicates success. You can customize this using `success_exit_codes`:

```yaml
name: "Custom Success Codes"
commands:
  - type: "shell"
    name: "Run Tool"
    command_line: "my-tool.sh"
    success_exit_codes: [0, 1, 2]
    # Exits 0, 1, or 2 are considered successful
    skip_exit_codes: [3]
    # Exit 3 skips remaining commands
    # Any other exit code is a failure
```

### Exit Code Priorities

When a command exits, Porch evaluates exit codes in this order:

1. **Skip codes** (`skip_exit_codes`): If matched, skip remaining commands
2. **Success codes** (`success_exit_codes`): If matched, continue normally
3. **Default**: Any other exit code is treated as a failure

## Error Handling

Porch provides comprehensive error handling throughout the workflow:

### Serial Batch Errors

In serial execution, an error stops subsequent commands unless they have `runs_on_condition: "error"` or `runs_on_condition: "always"`:

```yaml
name: "Error Handling in Serial"
commands:
  - type: "serial"
    name: "Build and Test"
    commands:
      - type: "shell"
        name: "Build"
        command_line: "make build"

      - type: "shell"
        name: "Test"
        command_line: "make test"
        # Skipped if Build fails

      - type: "shell"
        name: "Cleanup on Error"
        command_line: "make clean"
        runs_on_condition: "error"
        # Only runs if Build or Test fails

      - type: "shell"
        name: "Always Log"
        command_line: "save-log.sh"
        runs_on_condition: "always"
        # Always runs
```

### Parallel Batch Errors

In parallel execution, all commands run simultaneously. Errors are collected and reported after all commands complete:

```yaml
name: "Parallel Error Handling"
commands:
  - type: "parallel"
    name: "Run All Tests"
    commands:
      - type: "shell"
        name: "Unit Tests"
        command_line: "npm run test:unit"

      - type: "shell"
        name: "Integration Tests"
        command_line: "npm run test:integration"

      - type: "shell"
        name: "E2E Tests"
        command_line: "npm run test:e2e"
    # All three run concurrently
    # If any fail, the batch reports an error

  - type: "shell"
    name: "Generate Report"
    command_line: "test-report.sh"
    runs_on_condition: "always"
    # Runs regardless of test results
```

## Tolerating Errors

You can make commands "tolerant" of failures by using success exit codes or run conditions:

### Method 1: Success Exit Codes

```yaml
- type: "shell"
  name: "Optional Linter"
  command_line: "optional-lint.sh"
  success_exit_codes: [0, 1]
  # Both 0 and 1 are considered successful
```

### Method 2: Always Continue

```yaml
- type: "serial"
  name: "Best Effort Tasks"
  commands:
    - type: "shell"
      name: "Task 1"
      command_line: "task1.sh"

    - type: "shell"
      name: "Task 2 (Best Effort)"
      command_line: "task2.sh"
      success_exit_codes: [0, 1, 2, 3, 4, 5]
      # Any exit code up to 5 continues workflow

    - type: "shell"
      name: "Task 3"
      command_line: "task3.sh"
      # Will run even if Task 2 "fails"
```

## Complete Example

Here's a comprehensive example combining all flow control features:

```yaml
name: "CI/CD Pipeline with Flow Control"
commands:
  - type: "serial"
    name: "CI Pipeline"
    commands:
      # Check if deployment is enabled
      - type: "shell"
        name: "Check Deploy Flag"
        command_line: |
          if [ "$DEPLOY" != "true" ]; then
            echo "Deployment disabled. Skipping deploy steps." 1>&2
            exit 99
          fi
        skip_exit_codes: [99]

      # Build the application
      - type: "shell"
        name: "Build"
        command_line: "make build"

      # Run tests (parallel)
      - type: "parallel"
        name: "Tests"
        commands:
          - type: "shell"
            name: "Unit Tests"
            command_line: "make test-unit"

          - type: "shell"
            name: "Integration Tests"
            command_line: "make test-integration"

      # Deploy (only if tests passed)
      - type: "shell"
        name: "Deploy to Staging"
        command_line: "deploy.sh staging"
        runs_on_condition: "success"

      # Rollback on deploy failure
      - type: "shell"
        name: "Rollback"
        command_line: "rollback.sh"
        runs_on_condition: "error"

      # Always send notification
      - type: "shell"
        name: "Send Notification"
        command_line: "notify.sh"
        runs_on_condition: "always"
```

## Key Takeaways

1. **Conditional execution** allows workflows to adapt to previous command results
2. **Skip codes** provide graceful early termination without failing the workflow
3. **Success codes** let you define what "success" means for each command
4. **Run conditions** (`success`, `error`, `always`, `exit-codes`) control when commands execute
5. **Serial batches** stop on error unless explicitly handled
6. **Parallel batches** run all commands and collect errors
7. Combine these features to create robust, flexible workflows
