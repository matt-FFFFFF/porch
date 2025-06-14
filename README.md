# porch - A Portable Process Orchestration Framework

**porch** is a sophisticated Go-based process orchestration framework designed for running and managing complex command workflows. It provides a flexible, YAML-driven approach to define, compose, and execute command chains with advanced flow control, parallel processing, and comprehensive error handling.

It was designed to solve the problem of orchestrating complex command workflows in a portable and easy-to-use manner.
Its portability means that you can confidently run the same workflows locally, in CI/CD pipelines, or on any server without worrying about compatibility issues.

## ✨ Features

### 🚀 **Command Orchestration**

- **Serial Execution**: Run commands sequentially with dependency management
- **Parallel Execution**: Execute independent commands concurrently for optimal performance
- **Nested Batches**: Compose complex workflows with serial and parallel command combinations
- **Shell Commands**: Execute any shell command with full environment control
- **Directory Iteration**: Execute commands across multiple directories with flexible traversal options

### 📋 **Flexible Configuration**

- **YAML-Based Workflows**: Define command chains using simple, readable YAML syntax
- **Working Directory Management**: Control execution context with per-command working directories
- **Environment Variables**: Set and inherit environment variables at any level
- **Conditional Execution**: Run commands based on success, failure, or specific exit codes
- **Command Groups**: Define reusable command sets that can be referenced by container commands

### 🛡️ **Robust Error Handling**

- **Graceful Signal Handling**: Proper SIGINT/SIGTERM handling with graceful shutdown
- **Context-Aware Execution**: Full context propagation for cancellation and timeouts
- **Comprehensive Results**: Detailed execution results with exit codes, stdout, and stderr
- **Error Aggregation**: Collect and report errors across complex command hierarchies
- **Skip Controls**: Configure commands to skip remaining tasks based on exit codes

### 🎨 **Beautiful Output**

- **Tree Visualization**: Clear hierarchical display of command execution
- **Colorized Output**: Terminal-aware colored output for better readability
- **Structured Results**: JSON and pretty-printed result formatting
- **Progress Tracking**: Real-time execution status and progress indication

### 🔧 **Extensible Architecture**

- **Plugin-Based Commands**: Modular command registry for easy extension
- **Custom Commanders**: Implement your own command types via the Commander interface
- **Provider System**: Extensible item providers for dynamic command generation
- **Temporary Directory Support**: Isolated execution environments for testing and builds

## 📦 Installation

### Operating System Support

Porch currently supports the following operating systems:

- Linux
- macOS

Porch does compile and run on Windows, but the tests do not pass.
Therefore it is not officially supported.

### Install from Go modules

```bash
go install github.com/matt-FFFFFF/porch@latest
```

### Build from source

```bash
git clone https://github.com/matt-FFFFFF/porch.git
cd porch
make build
```

### Development setup

See [DEVELOPER.md](DEVELOPER.md) for detailed instructions on setting up a development environment, running tests, and contributing to the project.

## 🚀 Quick Start

### Basic Workflow

Create a YAML file defining your command workflow:

```yaml
name: "Build and Test Workflow"
description: "Complete CI/CD pipeline example"
commands:
  - type: "shell"
    name: "Setup Environment"
    command_line: "echo 'Setting up build environment...'"

  - type: "parallel"
    name: "Quality Checks"
    commands:
      - type: "shell"
        name: "Run Tests"
        command_line: "go test ./..."

      - type: "shell"
        name: "Run Linter"
        command_line: "golangci-lint run"

      - type: "shell"
        name: "Security Scan"
        command_line: "gosec ./..."

  - type: "serial"
    name: "Build and Package"
    commands:
      - type: "shell"
        name: "Build Application"
        command_line: "go build -o app ."

      - type: "copycwdtotemp"
        name: "Copy to Temp"

      - type: "shell"
        name: "Create Archive"
        command_line: "tar -czf app.tar.gz app"
```

### Execute the workflow

```bash
porch run -f workflow.yaml
```

### View saved results

```bash
porch show results
```

## 📚 CLI Commands

porch provides a comprehensive CLI with the following commands:

### `porch run --file <workflow.yaml>`

Execute a workflow defined in a YAML file.

**Usage:**

```bash
# Execute workflow
porch run --file workflow.yaml

# Execute and save results
porch run --file workflow.yaml --out results

# Execute from a remote repository
porch run --file git:://github.com/yourusername/yourrepo/workflow.yaml --out results

# Execute multiple workflows (in series)
porch run --file workflow1.yaml --file workflow2.yaml --out results
```

**Options:**

- `--output-success-details`, `--success`: Include successful results in the output
- `--no-output-stderr`, `--no-stderr`: Exclude stderr output in the results
- `--output-stdout`, `--stdout`: Include stdout output in the results

**Description:**

Reads a YAML workflow definition, executes commands according to their type (serial, parallel, shell, etc.), handles signal interruption gracefully, and optionally saves execution results for later analysis.

### `porch show <results>`

Display the results of a previous workflow execution.

**Usage:**

```bash
porch show results
```

**Options:**

- `--output-success-details`, `--success`: Include successful results in the output
- `--no-output-stderr`, `--no-stderr`: Exclude stderr output in the results
- `--output-stdout`, `--stdout`: Include stdout output in the results

**Description:**

Displays saved execution results with pretty-printed tree visualization, colorized output with error highlighting, detailed execution metrics, and supports JSON export capability.

### `porch config [command]`

Get information about configuration format and available commands.

**Usage:**

```bash
porch config                   # List all available commands
porch config shell             # Show shell command schema and example
porch config serial --markdown # Show serial command docs in Markdown format
```

**Options:**

- `--markdown`, `--md`: Output the configuration example in Markdown format

**Available Commands:**

- `porch config schema`: Output the complete JSON schema for configuration files

**Description:**

Provides comprehensive documentation for each command type, including YAML schema definitions, examples, and detailed descriptions of all available configuration options.

## � Configuration Schema

### Root Configuration

Every porch workflow starts with a root configuration that defines the overall structure:

```yaml
name: "Workflow Name"                    # Required: Descriptive name for the workflow
description: "Workflow description"      # Optional: Description of what this workflow does
commands: []                             # Required: List of commands to execute
command_groups: []                       # Optional: Named groups of commands for reuse
```

### Command Groups

Command groups allow you to define reusable sets of commands that can be referenced by container commands:

```yaml
command_groups:
  - name: "test-group"
    description: "Common testing commands"
    commands:
      - type: "shell"
        name: "Unit Tests"
        command_line: "go test ./..."
      - type: "shell"
        name: "Integration Tests"
        command_line: "go test -tags=integration ./..."
```

## 🛠️ Available Commands

porch supports five built-in command types for different execution patterns:

### 1. Shell Commands (`shell`)

Execute any shell command with full environment control and configurable exit code handling.

**Required Attributes:**

- `type: "shell"`
- `name`: Descriptive name for the command
- `command_line`: The shell command to execute

**Optional Attributes:**

- `working_directory`: Directory to execute the command in
- `env`: Environment variables as key-value pairs
- `runs_on_condition`: When to run (`success`, `error`, `always`, `exit-codes`)
- `runs_on_exit_codes`: Specific exit codes that trigger execution (used with `runs_on_condition: exit-codes`)
- `success_exit_codes`: Exit codes that indicate success (defaults to `[0]`)
- `skip_exit_codes`: Exit codes that skip remaining commands in the current batch

**Using Redirection:**

Porch captures stdout and stderr automatically, and will by default display stderr in the results if the step fails or returns a skippable exit code. Therefore you can use redirection to capture output. E.g. to output to stderr, use `command_line: "your_command 1>&2"`.

```yaml
command_line: |
  if [ -z  "$FOO" ]; then
  echo "FOO is not set. Skipping" 1>&2
  exit 99
fi
skip_exit_codes: [99]
```

**Example:**

```yaml
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

### 2. PowerShell commands (`pwsh`)

Execute any PowerShell script with full environment control and configurable exit code handling.

**Required Attributes:**

- `type: "pwsh"`
- `name`: Descriptive name for the command
- `script`: Path to the PowerShell script file to execute. Mutually exclusive with `script_file`.
- `script_file`: Path to the PowerShell script file to execute. Mutually exclusive with `script`.

**Optional Attributes:**

- `working_directory`: Directory to execute the command in
- `env`: Environment variables as key-value pairs
- `runs_on_condition`: When to run (`success`, `error`, `always`, `exit-codes`)
- `runs_on_exit_codes`: Specific exit codes that trigger execution (used with `runs_on_condition: exit-codes`)
- `success_exit_codes`: Exit codes that indicate success (defaults to `[0]`)
- `skip_exit_codes`: Exit codes that skip remaining commands in the current batch

**Example:**

```yaml
- type: "pwsh"
  name: "Run PowerShell Script"
  script: |
    Write-Host "Starting PowerShell script..."
    # Your PowerShell commands here
    Write-Host "PowerShell script completed."
  working_directory: "/path/to/project"
  env:
    VAR1: "value1"
    VAR2: "value2"
  success_exit_codes: [0]
  skip_exit_codes: [2]
  runs_on_condition: "success"
```

### 3. Serial Commands (`serial`)

Execute commands sequentially where order matters. Each command waits for the previous one to complete before starting.

**Required Attributes:**

- `type: "serial"`
- `name`: Descriptive name for the command batch

**Optional Attributes:**

- `working_directory`: Directory to execute commands in
- `env`: Environment variables inherited by all child commands
- `runs_on_condition`: When to run (`success`, `error`, `always`, `exit-codes`)
- `runs_on_exit_codes`: Specific exit codes that trigger execution
- `commands`: List of commands to execute sequentially (either this or `command_group`)
- `command_group`: Reference to a named command group (either this or `commands`)

**Example:**

```yaml
- type: "serial"
  name: "Setup and Build"
  working_directory: "/project"
  env:
    BUILD_MODE: "production"
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

### 4. Parallel Commands (`parallel`)

Execute independent commands concurrently for optimal performance. All commands start simultaneously.

**Required Attributes:**

- `type: "parallel"`
- `name`: Descriptive name for the command batch

**Optional Attributes:**

- `working_directory`: Directory to execute commands in
- `env`: Environment variables inherited by all child commands
- `runs_on_condition`: When to run (`success`, `error`, `always`, `exit-codes`)
- `runs_on_exit_codes`: Specific exit codes that trigger execution
- `commands`: List of commands to execute in parallel (either this or `command_group`)
- `command_group`: Reference to a named command group (either this or `commands`)

**Example:**

```yaml
- type: "parallel"
  name: "Quality Assurance"
  commands:
    - type: "shell"
      name: "Unit Tests"
      command_line: "go test -race ./..."
    - type: "shell"
      name: "Linting"
      command_line: "golangci-lint run --timeout=5m"
    - type: "shell"
      name: "Security Scan"
      command_line: "gosec -quiet ./..."
    - type: "shell"
      name: "Vulnerability Check"
      command_line: "govulncheck ./..."
```

### 5. ForEach Directory Commands (`foreachdirectory`)

Execute commands in each directory found by traversing the filesystem. Useful for monorepos or multi-module projects.
For each command, an environment variable called `ITEM` is set to the path of the current directory being processed.

**Required Attributes:**

- `type: "foreachdirectory"`
- `name`: Descriptive name for the command
- `mode`: Execution mode (`parallel` or `serial`)
- `depth`: Directory traversal depth (0 for unlimited, 1 for immediate children only)
- `include_hidden`: Whether to include hidden directories (`true` or `false`)
- `working_directory_strategy`: How to set working directory (`none`, `item_relative`, `item_absolute`)

**Optional Attributes:**

- `working_directory`: Base directory to start traversal from
- `env`: Environment variables inherited by all child commands
- `runs_on_condition`: When to run (`success`, `error`, `always`, `exit-codes`)
- `runs_on_exit_codes`: Specific exit codes that trigger execution
- `commands`: List of commands to execute in each directory (either this or `command_group`)
- `command_group`: Reference to a named command group (either this or `commands`)

**Working Directory Strategies:**

- `none`: Don't change working directory for child commands
- `item_relative`: Set working directory relative to the current directory
- `item_absolute`: **experimental** Set working directory to the absolute path of each found directory

**Example:**

```yaml
- type: "foreachdirectory"
  name: "Test All Modules"
  working_directory: "./modules"
  mode: "parallel"
  depth: 1
  include_hidden: false
  working_directory_strategy: "item_relative"
  commands:
    - type: "shell"
      name: "Run Module Tests"
      command_line: "go test ./..."
    - type: "shell"
      name: "Check Module"
      command_line: "go mod verify"
```

### 6. Copy Current Working Directory to Temp (`copycwdtotemp`)

A specialized command for working in temporary directories. Copies the current working directory to a temporary location for isolated execution.

**Required Attributes:**

- `type: "copycwdtotemp"`
- `name`: Descriptive name for the command

**Optional Attributes:**

- `working_directory`: Directory to copy (defaults to current working directory)
- `env`: Environment variables
- `runs_on_condition`: When to run (`success`, `error`, `always`, `exit-codes`)
- `runs_on_exit_codes`: Specific exit codes that trigger execution

**Example:**

```yaml
- type: "copycwdtotemp"
  name: "Create Isolated Environment"
  working_directory: "."
```

This command is particularly useful for:

- Testing builds without affecting the source directory
- Creating clean environments for packaging
- Isolating potentially destructive operations

## ⚙️ Common Configuration Options

### Conditional Execution

All commands support conditional execution based on the results of previous commands:

```yaml
- type: "shell"
  name: "Cleanup on Error"
  command_line: "rm -rf temp_files"
  runs_on_condition: "error"           # Only run if previous commands failed

- type: "shell"
  name: "Deploy on Success"
  command_line: "./deploy.sh"
  runs_on_condition: "success"         # Only run if previous commands succeeded

- type: "shell"
  name: "Always Cleanup"
  command_line: "cleanup.sh"
  runs_on_condition: "always"          # Always run regardless of previous results

- type: "shell"
  name: "Run on Specific Exit Codes"
  command_line: "handle_warning.sh"
  runs_on_condition: "exit-codes"
  runs_on_exit_codes: [1, 2]           # Only run if previous command exited with code 1 or 2
```

### Environment Variables

Environment variables can be set at any level and are inherited by child commands:

```yaml
env:
  GLOBAL_VAR: "global_value"
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
          OPTIMIZATION: "O3"              # This command sees GLOBAL_VAR, BUILD_TYPE, and OPTIMIZATION
```

### Working Directory Management

Control execution context with flexible working directory options:

```yaml
- type: "shell"
  name: "Build in Subdirectory"
  command_line: "make build"
  working_directory: "./subproject"      # Relative to current directory

- type: "serial"
  name: "Build Multiple Projects"
  working_directory: "/absolute/path"    # Absolute path
  commands:
    - type: "shell"
      name: "Build Project A"
      command_line: "make -C project-a"
    - type: "shell"
      name: "Build Project B"
      command_line: "make -C project-b"
```

## 🏗️ Advanced Examples

### Complete CI/CD Pipeline

```yaml
name: "Complete CI/CD Pipeline"
description: "Full build, test, and deployment workflow"
commands:
  # Environment setup
  - type: "serial"
    name: "Environment Setup"
    commands:
      - type: "shell"
        name: "Check Go Version"
        command_line: "go version"
      - type: "shell"
        name: "Install Dependencies"
        command_line: "go mod download"

  # Quality assurance (parallel)
  - type: "parallel"
    name: "Quality Assurance"
    commands:
      - type: "shell"
        name: "Unit Tests"
        command_line: "go test -race -coverprofile=coverage.out ./..."
      - type: "shell"
        name: "Linting"
        command_line: "golangci-lint run --timeout=5m"
      - type: "shell"
        name: "Security Scan"
        command_line: "gosec -quiet ./..."
      - type: "shell"
        name: "Vulnerability Check"
        command_line: "govulncheck ./..."

  # Build process
  - type: "serial"
    name: "Build Process"
    runs_on_condition: "success"
    commands:
      - type: "shell"
        name: "Build for Linux"
        command_line: "go build -o dist/app-linux ."
        env:
          GOOS: "linux"
          GOARCH: "amd64"
          CGO_ENABLED: "0"
      - type: "shell"
        name: "Build for macOS"
        command_line: "go build -o dist/app-macos ."
        env:
          GOOS: "darwin"
          GOARCH: "amd64"
          CGO_ENABLED: "0"
      - type: "shell"
        name: "Build for Windows"
        command_line: "go build -o dist/app-windows.exe ."
        env:
          GOOS: "windows"
          GOARCH: "amd64"
          CGO_ENABLED: "0"

  # Deployment (only on success)
  - type: "shell"
    name: "Deploy to Staging"
    command_line: "./scripts/deploy.sh staging"
    runs_on_condition: "success"
```

### Monorepo Testing with ForEach Directory

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
            echo "No tests found in $(pwd)"
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

### Isolated Testing Environment

```yaml
name: "Isolated Testing"
description: "Run tests in isolated temporary directory"
commands:
  - type: "serial"
    name: "Isolated Test Workflow"
    commands:
      - type: "copycwdtotemp"
        name: "Copy to Temp Directory"
      - type: "shell"
        name: "Modify Files Safely"
        command_line: "echo 'test content' > test.txt"
      - type: "shell"
        name: "Run Tests"
        command_line: "go test ./..."
      - type: "shell"
        name: "Generate Report"
        command_line: "go test -json ./... > test-results.json"
```

### Command Groups Example

```yaml
name: "Command Groups Demo"
description: "Example showing reusable command groups"

command_groups:
  - name: "quality-checks"
    description: "Standard quality assurance commands"
    commands:
      - type: "shell"
        name: "Run Tests"
        command_line: "go test ./..."
      - type: "shell"
        name: "Run Linter"
        command_line: "golangci-lint run"
      - type: "shell"
        name: "Security Scan"
        command_line: "gosec ./..."

  - name: "build-commands"
    description: "Standard build commands"
    commands:
      - type: "shell"
        name: "Build"
        command_line: "go build -o app ."
      - type: "shell"
        name: "Package"
        command_line: "tar -czf app.tar.gz app"

commands:
  - type: "parallel"
    name: "Quality Checks"
    command_group: "quality-checks"

  - type: "serial"
    name: "Build Process"
    runs_on_condition: "success"
    command_group: "build-commands"
```

### Conditional Execution Patterns

```yaml
name: "Conditional Execution Demo"
description: "Examples of conditional command execution"
commands:
  - type: "shell"
    name: "Primary Task"
    command_line: "make build"

  - type: "shell"
    name: "Success Handler"
    command_line: "echo 'Build succeeded, deploying...'"
    runs_on_condition: "success"

  - type: "shell"
    name: "Error Handler"
    command_line: "echo 'Build failed, cleaning up...'"
    runs_on_condition: "error"

  - type: "shell"
    name: "Warning Handler"
    command_line: "echo 'Build completed with warnings'"
    runs_on_condition: "exit-codes"
    runs_on_exit_codes: [1, 2]

  - type: "shell"
    name: "Always Cleanup"
    command_line: "rm -rf temp/"
    runs_on_condition: "always"
```

## 🎨 Output and Results

### Pretty-Printed Output

porch automatically generates beautiful tree-structured output showing the execution hierarchy:

```text
✓ Build and Test Workflow
  ✓ Setup Environment (0ms)
  ✓ Quality Checks (2.1s)
    ✓ Run Tests (1.8s)
    ✓ Run Linter (1.2s)
    ✗ Security Scan (0.5s) - exit code 1
      stderr: gosec: 1 issue found
  ✓ Build Process (1.5s)
    ✓ Build for Linux (0.8s)
    ✓ Build for macOS (0.7s)
```

## 🚦 Signal Handling

porch provides sophisticated signal handling for graceful shutdown:

**First Signal (SIGINT/SIGTERM)**: Initiates graceful shutdown

- Running commands receive the signal for cleanup
- No new commands are started
- Context remains active for cleanup operations

**Second Signal**: Forces immediate termination

- Context is cancelled
- All running processes are killed
- Execution stops immediately

This allows for proper cleanup and prevents data corruption during interruption.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/) and the amazing Go ecosystem
- CLI powered by [urfave/cli](https://github.com/urfave/cli)
- YAML parsing with [goccy/go-yaml](https://github.com/goccy/go-yaml)
- Beautiful JSON output with [TylerBrock/colorjson](https://github.com/TylerBrock/colorjson)

---

**porch** - Making process orchestration simple, powerful, and beautiful. 🏠
