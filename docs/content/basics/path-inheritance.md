+++
title = "Path Inheritance"
weight = 2
+++

One of Porch's most powerful features is its intelligent handling of working directories through **path inheritance**. This allows you to build complex workflows where commands naturally inherit and modify their working directory context.

## How It Works

Porch uses a **recursive dynamic programming approach** to resolve working directories. Each command's working directory is determined by combining its own `working_directory` setting with its parent's resolved directory.

### Resolution Rules

Porch resolves working directories using these rules:

1. The top level workflow directory is ".", the current directory where Porch is run.
2. If the command's `working_directory` is empty, it will use the parent's resolved directory.
3. If the command's `working_directory` is an absolute path, it will use that path directly.
4. If the command's `working_directory` is a relative path and has a parent, it will be joined to its parent's resolved directory.

This creates a hierarchical resolution system where child commands automatically inherit their parent's working directory unless they specify otherwise.

## Default Behavior: Relative Paths

By default, Porch uses **relative paths** for working directories. This means:

- Child commands inherit the parent's working directory
- Relative paths are resolved against the parent's directory
- The working directory "flows down" through the command hierarchy

### Example: Relative Paths

```yaml
name: "Build Workflow"
commands:
  - type: "serial"
    name: "Build Steps"
    working_directory: "./project"
    commands:
      - type: "shell"
        name: "Install Dependencies"
        command_line: "npm install"
        # Runs in ./project (inherited)

      - type: "shell"
        name: "Build Subpackage"
        working_directory: "packages/core"
        command_line: "npm run build"
        # Runs in ./project/packages/core (joined with parent)

      - type: "shell"
        name: "Run Tests"
        working_directory: "../.."
        command_line: "npm test"
        # Runs in . (relative to ./project)
```

In this example:

- The first command runs in `./project`
- The second command runs in `./project/packages/core`
- The third command runs in `.` (going up two levels from `./project`)

## Breaking Inheritance: Absolute Paths

If you specify a command with an **absolute path**, it will not inherit from the parent. The absolute path is used directly.

### Example: Absolute Paths

```yaml
name: "Multi-Project Build"
commands:
  - type: "serial"
    name: "Build All Projects"
    working_directory: "./frontend"
    commands:
      - type: "shell"
        name: "Build Frontend"
        command_line: "npm run build"
        # Runs in ./frontend (inherited)

      - type: "shell"
        name: "Build Backend"
        working_directory: "/home/user/projects/backend"
        command_line: "go build"
        # Runs in /home/user/projects/backend (absolute path)

      - type: "shell"
        name: "Package Frontend"
        command_line: "tar -czf dist.tar.gz dist/"
        # Runs in ./frontend (back to inherited path)
```

In this example:

- First command: Uses `./frontend` (inherited)
- Second command: Uses `/home/user/projects/backend` (absolute, breaks inheritance)
- Third command: Uses `./frontend` (back to inherited path)

## Practical Examples

### Monorepo Structure

```yaml
name: "Monorepo Tests"
commands:
  - type: "foreachdirectory"
    name: "Test Each Package"
    working_directory: "./packages"
    mode: "parallel"
    depth: 1
    include_hidden: false
    working_directory_strategy: "item_relative"
    commands:
      - type: "shell"
        name: "Run Package Tests"
        command_line: "npm test"
        # Each iteration runs in ./packages/<package-name>
```

### Nested Serial Execution

```yaml
name: "Deployment Pipeline"
working_directory: "./dist"
commands:
  - type: "serial"
    name: "Prepare"
    commands:
      - type: "shell"
        name: "Clean"
        command_line: "rm -rf *"
        # Runs in ./dist

      - type: "shell"
        name: "Copy Assets"
        working_directory: "../src/assets"
        command_line: "cp -r . ../../dist/"
        # Runs in ./src/assets
```

### Temporary Directory Workflow

```yaml
name: "Isolated Build"
commands:
  - type: "copycwdtotemp"
    name: "Copy to Temp"
    working_directory: "./src"
    # Creates temp dir and copies ./src there
    # Sets absolute path to temp dir

  - type: "shell"
    name: "Build in Isolation"
    command_line: "make build"
    # Runs in the temp directory (absolute path from copycwdtotemp)

  - type: "shell"
    name: "Copy Back Results"
    working_directory: "/original/path/dist"
    command_line: "cp ../build/* ."
    # Runs in absolute path, independent of temp dir
```

## Key Takeaways

1. **Relative paths are inherited**: Child commands inherit parent's working directory
2. **Absolute paths break inheritance**: Use absolute paths when you need to "escape" the hierarchy
3. **Empty paths inherit**: If you don't specify a working directory, you get the parent's
4. **Path joining is automatic**: Relative paths are joined with parent's path using `filepath.Join()`
5. **Resolution is recursive**: The algorithm walks up the command tree to resolve the final path

This design makes it easy to:

- Build hierarchical workflows with natural directory context
- Override paths when needed with absolute paths
- Keep workflows portable using relative paths
- Avoid repetitive path specifications
