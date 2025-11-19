+++
title = "ForEach Directory Command"
weight = 5
+++

The `foreachdirectory` command executes commands in each directory found by traversing the filesystem. This is particularly useful for monorepos or multi-module projects.

Each item found is made available to child commands via the `ITEM` environment variable, which contains the relative path of the current directory being processed.

## Attributes

### Required

- **`type: "foreachdirectory"`**: Identifies this as a foreach directory command
- **`name`**: Descriptive name for the command
- **`mode`**: Execution mode (`parallel` or `serial`)
- **`depth`**: Directory traversal depth (0 for unlimited, 1 for immediate children only)
- **`include_hidden`**: Whether to include hidden directories (`true` or `false`)
- **`working_directory_strategy`**: How to set working directory (`none`, `item_relative`)

### Optional

- **`working_directory`**: Base directory to start traversal from
- **`env`**: Environment variables inherited by all child commands
- **`runs_on_condition`**: When to run (`success`, `error`, `always`, `exit-codes`)
- **`runs_on_exit_codes`**: Specific exit codes that trigger execution
- **`commands`**: List of commands to execute in each directory (either this or `command_group`)
- **`command_group`**: Reference to a named command group (either this or `commands`)

## Basic Example

```yaml
name: "Test All Modules"
commands:
  - type: "foreachdirectory"
    name: "Run Module Tests"
    working_directory: "./packages"
    mode: "parallel"
    depth: 1
    include_hidden: false
    working_directory_strategy: "item_relative"
    commands:
      - type: "shell"
        name: "Test Package"
        command_line: "npm test"
```

## Working Directory Strategy

### `item_relative`

Sets the working directory to each found directory relative to the current directory:

```yaml
- type: "foreachdirectory"
  name: "Build Each Module"
  working_directory: "./modules"
  working_directory_strategy: "item_relative"
  mode: "parallel"
  depth: 1
  include_hidden: false
  commands:
    - type: "shell"
      name: "Build"
      command_line: "go build"
      # Runs in ./modules/module1, ./modules/module2, etc.
```

### `none`

Does not change the working directory; commands run in the parent's working directory:

```yaml
- type: "foreachdirectory"
  name: "Process Directories"
  working_directory: "./data"
  working_directory_strategy: "none"
  mode: "serial"
  depth: 1
  include_hidden: false
  commands:
    - type: "shell"
      name: "Process Directory"
      command_line: "process-dir.sh $ITEM"
      # $ITEM contains the directory path
      # Command runs in ./data (not in each subdirectory)
```

## Environment Variable: ITEM

For each directory iteration, an environment variable `ITEM` is set to the path of the current directory:

```yaml
- type: "foreachdirectory"
  name: "List Directories"
  working_directory: "./projects"
  mode: "serial"
  depth: 1
  include_hidden: false
  working_directory_strategy: "none"
  commands:
    - type: "shell"
      name: "Show Directory"
      command_line: "echo 'Processing: $ITEM'"
      # $ITEM will be "project1", "project2", etc.
```

## Execution Modes

### Parallel Mode

Process all directories concurrently:

```yaml
- type: "foreachdirectory"
  name: "Parallel Module Tests"
  working_directory: "./packages"
  mode: "parallel"
  depth: 1
  include_hidden: false
  working_directory_strategy: "item_relative"
  commands:
    - type: "shell"
      name: "Run Tests"
      command_line: "npm test"
    # All packages tested simultaneously
```

### Serial Mode

Process directories one at a time:

```yaml
- type: "foreachdirectory"
  name: "Serial Module Builds"
  working_directory: "./packages"
  mode: "serial"
  depth: 1
  include_hidden: false
  working_directory_strategy: "item_relative"
  commands:
    - type: "shell"
      name: "Build Package"
      command_line: "npm run build"
    # Packages built one after another
```

## Depth Control

### Depth 1 - Immediate Children Only

```yaml
- type: "foreachdirectory"
  name: "Top Level Only"
  working_directory: "./src"
  depth: 1
  include_hidden: false
  mode: "parallel"
  working_directory_strategy: "item_relative"
  commands:
    - type: "shell"
      name: "Process"
      command_line: "process.sh"
  # Only processes ./src/dir1, ./src/dir2
  # Does NOT process ./src/dir1/subdir
```

### Depth 0 - Unlimited Recursion

```yaml
- type: "foreachdirectory"
  name: "All Directories"
  working_directory: "./src"
  depth: 0
  include_hidden: false
  mode: "parallel"
  working_directory_strategy: "item_relative"
  commands:
    - type: "shell"
      name: "Find Go Modules"
      command_line: "test -f go.mod && go test ./..."
  # Processes all directories at any depth
```

## Complete Example

```yaml
name: "Monorepo Testing"
description: "Test all modules in a monorepo structure"
commands:
  - type: "foreachdirectory"
    name: "Test All Modules"
    working_directory: "./modules"
    mode: "parallel"
    depth: 1
    include_hidden: false
    working_directory_strategy: "item_relative"
    commands:
      - type: "shell"
        name: "Check for Tests"
        command_line: |
          if [ ! -d ./tests ]; then
            echo "No tests found in $(pwd)" 1>&2
            exit 99
          fi
        skip_exit_codes: [99]

      - type: "shell"
        name: "Install Dependencies"
        command_line: "npm install"

      - type: "shell"
        name: "Run Tests"
        command_line: "npm test"

      - type: "shell"
        name: "Build Module"
        command_line: "npm run build"
```

## Skip Pattern

Use skip codes to skip directories without certain criteria:

```yaml
- type: "foreachdirectory"
  name: "Build Go Modules"
  working_directory: "./services"
  mode: "parallel"
  depth: 1
  include_hidden: false
  working_directory_strategy: "item_relative"
  commands:
    - type: "shell"
      name: "Check for go.mod"
      command_line: |
        if [ ! -f go.mod ]; then
          echo "No go.mod in $ITEM, skipping" 1>&2
          exit 100
        fi
      skip_exit_codes: [100]

    - type: "shell"
      name: "Build"
      command_line: "go build ./..."
```

## Hidden Directories

Control whether hidden directories (starting with `.`) are included:

```yaml
# Include hidden directories
- type: "foreachdirectory"
  name: "Process All"
  working_directory: "."
  mode: "parallel"
  depth: 1
  include_hidden: true # Includes .git, .github, etc.
  working_directory_strategy: "item_relative"
  commands:
    - type: "shell"
      name: "Process"
      command_line: "process.sh"

# Exclude hidden directories (recommended)
- type: "foreachdirectory"
  name: "Process Visible Only"
  working_directory: "."
  mode: "parallel"
  depth: 1
  include_hidden: false # Skips .git, .github, etc.
  working_directory_strategy: "item_relative"
  commands:
    - type: "shell"
      name: "Process"
      command_line: "process.sh"
```

## Best Practices

1. **Use depth: 1 when possible**: Prevents unexpected deep recursion
2. **Set include_hidden: false**: Avoid processing system directories
3. **Use skip codes**: Skip directories that don't meet criteria
4. **Choose appropriate mode**: Parallel for speed, serial for order
5. **Use item_relative strategy**: Most common and intuitive behavior
6. **Access $ITEM variable**: Use the ITEM environment variable in commands

## Common Use Cases

### Monorepo Package Testing

```yaml
- type: "foreachdirectory"
  name: "Test All Packages"
  working_directory: "./packages"
  mode: "parallel"
  depth: 1
  include_hidden: false
  working_directory_strategy: "item_relative"
  commands:
    - type: "shell"
      name: "Run Package Tests"
      command_line: "npm test"
```

### Multi-Service Deployment

```yaml
- type: "foreachdirectory"
  name: "Deploy All Services"
  working_directory: "./services"
  mode: "serial"
  depth: 1
  include_hidden: false
  working_directory_strategy: "item_relative"
  commands:
    - type: "shell"
      name: "Build Docker Image"
      command_line: "docker build -t $ITEM ."
    - type: "shell"
      name: "Deploy Service"
      command_line: "kubectl apply -f deployment.yaml"
```

## Related

- [Path Inheritance](../basics/path-inheritance/) - Working directory resolution
- [Parallel Command](parallel/) - Concurrent execution
- [Serial Command](serial/) - Sequential execution
- [Flow Control](../basics/flow-control/) - Skip codes and conditional execution
