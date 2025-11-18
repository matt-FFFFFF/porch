+++
title = "Copy to Temp Command"
weight = 6
+++

# Copy to Temp Command

The `copycwdtotemp` command copies the current working directory to a temporary location for isolated execution. This is useful for testing, building, or any operations that should not affect the source directory.

## Attributes

### Required

- **`type: "copycwdtotemp"`**: Identifies this as a copy to temp command
- **`name`**: Descriptive name for the command

### Optional

- **`working_directory`**: Directory to copy (defaults to current working directory)
- **`env`**: Environment variables
- **`runs_on_condition`**: When to run (`success`, `error`, `always`, `exit-codes`)
- **`runs_on_exit_codes`**: Specific exit codes that trigger execution

## Basic Example

```yaml
name: "Isolated Build"
commands:
  - type: "copycwdtotemp"
    name: "Create Isolated Environment"
    working_directory: "."
```

## How It Works

When `copycwdtotemp` executes:

1. Creates a new temporary directory
2. Copies all contents from the working directory to the temp directory
3. Sets the working directory to the absolute path of the temp directory
4. Subsequent commands inherit this temp directory path

The temp directory is automatically cleaned up when the workflow completes.

## Complete Example

```yaml
name: "Isolated Testing"
description: "Run tests in isolated temporary directory"
commands:
  - type: "serial"
    name: "Isolated Test Workflow"
    commands:
      - type: "copycwdtotemp"
        name: "Copy to Temp Directory"
        working_directory: "./src"
        # Copies ./src to /tmp/porch-12345/src

      - type: "shell"
        name: "Modify Files Safely"
        command_line: "echo 'test content' > test.txt"
        # Runs in temp directory, doesn't affect source

      - type: "shell"
        name: "Run Tests"
        command_line: "go test ./..."
        # Tests run in isolated environment

      - type: "shell"
        name: "Generate Report"
        command_line: "go test -json ./... > test-results.json"
        # Results saved in temp directory
```

## Use Cases

### 1. Isolated Testing

Test without affecting source files:

```yaml
commands:
  - type: "copycwdtotemp"
    name: "Create Test Environment"

  - type: "shell"
    name: "Run Destructive Tests"
    command_line: "npm test -- --coverage"
    # Coverage files and test artifacts don't pollute source
```

### 2. Clean Builds

Build in a clean environment:

```yaml
commands:
  - type: "copycwdtotemp"
    name: "Create Build Environment"
    working_directory: "."

  - type: "shell"
    name: "Clean Build"
    command_line: "make clean && make build"
    # Build artifacts stay in temp directory
```

### 3. Temporary Modifications

Make temporary changes safely:

```yaml
commands:
  - type: "copycwdtotemp"
    name: "Create Temp Copy"

  - type: "shell"
    name: "Modify Configuration"
    command_line: "sed -i 's/debug=true/debug=false/' config.yaml"
    # Original config.yaml is unchanged

  - type: "shell"
    name: "Test Production Config"
    command_line: "npm test"
```

### 4. Package Generation

Create distribution packages:

```yaml
commands:
  - type: "copycwdtotemp"
    name: "Prepare Package Directory"
    working_directory: "./dist"

  - type: "shell"
    name: "Remove Dev Files"
    command_line: "rm -rf *.map *.test.js"

  - type: "shell"
    name: "Create Archive"
    command_line: "tar -czf ../package.tar.gz ."
```

## Working with Multiple Steps

The temp directory path persists for subsequent commands:

```yaml
- type: "serial"
  name: "Multi-Step Temp Workflow"
  commands:
    - type: "copycwdtotemp"
      name: "Step 1: Copy to Temp"
      # Sets cwd to /tmp/porch-xyz/

    - type: "shell"
      name: "Step 2: Build"
      command_line: "make build"
      # Runs in /tmp/porch-xyz/

    - type: "shell"
      name: "Step 3: Test"
      command_line: "make test"
      # Runs in /tmp/porch-xyz/

    - type: "shell"
      name: "Step 4: Package"
      command_line: "tar -czf output.tar.gz ."
      # Runs in /tmp/porch-xyz/
```

## Path Inheritance

`copycwdtotemp` sets an **absolute path** to the temp directory, breaking path inheritance:

```yaml
- type: "serial"
  name: "Temp and Regular Mix"
  working_directory: "./project"
  commands:
    - type: "shell"
      name: "Before Temp"
      command_line: "pwd"
      # Prints: ./project

    - type: "copycwdtotemp"
      name: "Copy to Temp"
      # Sets absolute path: /tmp/porch-abc123/

    - type: "shell"
      name: "In Temp"
      command_line: "pwd"
      # Prints: /tmp/porch-abc123/

    - type: "shell"
      name: "Back to Project"
      working_directory: "/original/project/path"
      command_line: "pwd"
      # Prints: /original/project/path
```

## Combining with ForEach Directory

```yaml
- type: "foreachdirectory"
  name: "Test Each Module in Isolation"
  working_directory: "./modules"
  mode: "serial"
  depth: 1
  include_hidden: false
  working_directory_strategy: "item_relative"
  commands:
    - type: "copycwdtotemp"
      name: "Copy Module to Temp"
      # Each module gets its own temp directory

    - type: "shell"
      name: "Run Module Tests"
      command_line: "npm test"
      # Tests run in isolated temp copy
```

## Best Practices

1. **Use for isolation**: When you need to modify files without affecting source
2. **Clean builds**: Ensure builds start from a clean state
3. **Testing**: Run destructive tests without risk
4. **Temporary modifications**: Make config changes for testing
5. **Remember it's temporary**: Files in temp directory are lost after workflow

## Limitations

1. **Temporary files are deleted**: Results are lost unless copied out
2. **Disk space**: Copying large directories uses disk space
3. **Performance**: Copying takes time for large directories
4. **Absolute path**: Sets absolute path, breaking relative path inheritance

## Saving Results

To save results from temp directory, copy them back:

```yaml
- type: "serial"
  name: "Build and Save"
  commands:
    - type: "copycwdtotemp"
      name: "Copy to Temp"
      working_directory: "."

    - type: "shell"
      name: "Build"
      command_line: "make build"

    - type: "shell"
      name: "Copy Results Back"
      working_directory: "/original/path/dist"
      command_line: "cp /tmp/porch-*/build/* ."
      # Copy build artifacts back to original location
```

## Related

- [Path Inheritance](../basics/path-inheritance/) - How working directories work
- [Serial Command](serial/) - Sequential execution
- [Shell Command](shell/) - Execute shell commands
