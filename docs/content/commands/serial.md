+++
title = "Serial Command"
weight = 3
+++

The `serial` command executes a list of commands sequentially, where each command waits for the previous one to complete before starting.

## Attributes

### Required

- **`type: "serial"`**: Identifies this as a serial batch command
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
name: "Build Pipeline"
commands:
  - type: "serial"
    name: "Build Steps"
    commands:
      - type: "shell"
        name: "Install Dependencies"
        command_line: "npm install"

      - type: "shell"
        name: "Build"
        command_line: "npm run build"

      - type: "shell"
        name: "Test"
        command_line: "npm test"
```

## Execution Flow

In serial execution:

1. Commands execute in the order defined
2. Each command waits for the previous to complete
3. If a command fails, subsequent commands are skipped (unless they have `runs_on_condition: "error"` or `runs_on_condition: "always"`)
4. Skip codes propagate through the serial batch

```yaml
- type: "serial"
  name: "Sequential Tasks"
  commands:
    - type: "shell"
      name: "Task 1"
      command_line: "task1.sh"
      # Runs first

    - type: "shell"
      name: "Task 2"
      command_line: "task2.sh"
      # Runs after Task 1 completes

    - type: "shell"
      name: "Task 3"
      command_line: "task3.sh"
      # Runs after Task 2 completes
```

## Environment and Working Directory Inheritance

Child commands inherit environment variables and working directory:

```yaml
- type: "serial"
  name: "Build Process"
  working_directory: "./app"
  env:
    BUILD_MODE: "production"
  commands:
    - type: "shell"
      name: "Build"
      command_line: "make build"
      # Runs in ./app with BUILD_MODE=production

    - type: "shell"
      name: "Test"
      working_directory: "tests"
      command_line: "make test"
      # Runs in ./app/tests with BUILD_MODE=production
```

## Error Handling

Use conditional execution for error handling:

```yaml
- type: "serial"
  name: "Build with Error Handling"
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
      name: "Always Notify"
      command_line: "notify.sh"
      runs_on_condition: "always"
      # Always runs
```

## Nested Batches

Serial batches can contain other batches:

```yaml
- type: "serial"
  name: "Complete Pipeline"
  commands:
    - type: "shell"
      name: "Setup"
      command_line: "setup.sh"

    - type: "parallel"
      name: "Quality Checks"
      commands:
        - type: "shell"
          name: "Tests"
          command_line: "make test"
        - type: "shell"
          name: "Lint"
          command_line: "make lint"

    - type: "shell"
      name: "Deploy"
      command_line: "deploy.sh"
      runs_on_condition: "success"
```

## Using Command Groups

Reference reusable command groups:

```yaml
name: "Workflow with Command Groups"

command_groups:
  - name: "build-steps"
    commands:
      - type: "shell"
        name: "Install"
        command_line: "npm install"
      - type: "shell"
        name: "Build"
        command_line: "npm run build"

commands:
  - type: "serial"
    name: "Execute Build"
    command_group: "build-steps"
```

## Best Practices

1. **Group related tasks**: Use serial for tasks that must run in order
2. **Handle errors**: Add error handlers with `runs_on_condition: "error"`
3. **Use always for cleanup**: Ensure cleanup runs with `runs_on_condition: "always"`
4. **Set common env at batch level**: Inherit environment variables to child commands
5. **Use descriptive names**: Name each step clearly for better visibility

## Related

- [Parallel Command](parallel/) - Execute commands concurrently
- [Flow Control](../basics/flow-control/) - Conditional execution and error handling
- [Path Inheritance](../basics/path-inheritance/) - Working directory resolution
