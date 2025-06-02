# porch - A Portable Process Orchestration Framework

**porch** is a sophisticated Go-based process orchestration framework designed for running and managing complex command workflows. It provides a flexible, YAML-driven approach to define, compose, and execute command chains with advanced flow control, parallel processing, and comprehensive error handling.

## ✨ Features

### 🚀 **Command Orchestration**

- **Serial Execution**: Run commands sequentially with dependency management
- **Parallel Execution**: Execute independent commands concurrently for optimal performance
- **Nested Batches**: Compose complex workflows with serial and parallel command combinations
- **Shell Commands**: Execute any shell command with full environment control

### 📋 **Flexible Configuration**

- **YAML-Based Workflows**: Define command chains using simple, readable YAML syntax
- **Working Directory Management**: Control execution context with per-command working directories
- **Environment Variables**: Set and inherit environment variables at any level
- **Conditional Execution**: Run commands based on success, failure, or specific exit codes

### 🛡️ **Robust Error Handling**

- **Graceful Signal Handling**: Proper SIGINT/SIGTERM handling with graceful shutdown
- **Context-Aware Execution**: Full context propagation for cancellation and timeouts
- **Comprehensive Results**: Detailed execution results with exit codes, stdout, and stderr
- **Error Aggregation**: Collect and report errors across complex command hierarchies

### 🎨 **Beautiful Output**

- **Tree Visualization**: Clear hierarchical display of command execution
- **Colorized Output**: Terminal-aware colored output for better readability
- **Structured Results**: JSON and pretty-printed result formatting
- **Progress Tracking**: Real-time execution status and progress indication

### 🔧 **Extensible Architecture**

- **Plugin-Based Commands**: Modular command registry for easy extension
- **Custom Commanders**: Implement your own command types via the Commander interface
- **Provider System**: Extensible item providers for dynamic command generation

## 📦 Installation

### Install from Go modules

```bash
go install github.com/matt-FFFFFF/porch@latest
```

### Build from source

```bash
git clone https://github.com/matt-FFFFFF/porch.git
cd porch
go build -o porch cmd/main.go
```

### Development setup

```bash
git clone https://github.com/matt-FFFFFF/porch.git
cd porch
make test      # Run tests
make testcover # Run tests with coverage
make lint      # Run linter
```

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
porch run workflow.yaml
```

### View saved results

```bash
porch show results.gob
```

## 📚 Available Commands

porch provides a CLI with the following commands:

### `porch run <workflow.yaml>`

Execute a workflow defined in a YAML file.

**Options:**

- Reads YAML workflow definition
- Executes commands according to their type (serial, parallel, shell, etc.)
- Handles signal interruption gracefully
- Saves execution results for later analysis

### `porch show <results.gob>`

Display the results of a previous workflow execution.

**Features:**

- Pretty-printed tree visualization
- Colorized output with error highlighting
- Detailed execution metrics
- JSON export capability

## 🛠️ Command Types

porch supports several built-in command types for different execution patterns:

### 1. Shell Commands

Execute any shell command with full environment control:

```yaml
- type: "shell"
  name: "Build Project"
  command_line: "go build -o app ."
  working_directory: "/path/to/project"
  env:
    CGO_ENABLED: "0"
    GOOS: "linux"
```

### 2. Serial Batches

Execute commands sequentially where order matters:

```yaml
- type: "serial"
  name: "Setup and Build"
  commands:
    - type: "shell"
      name: "Install Dependencies"
      command_line: "go mod download"
    - type: "shell"
      name: "Build"
      command_line: "go build ."
```

### 3. Parallel Batches

Execute independent commands concurrently for optimal performance:

```yaml
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
```

### 4. Copy Current Working Directory to Temp

A specialized command for working in temporary directories:

```yaml
- type: "copycwdtotemp"
  name: "Work in Isolation"
  cwd: "."
```

This command copies the current working directory to a temporary location, allowing subsequent commands to work in isolation without affecting the original directory.

## ⚙️ Configuration Options

### Common Fields

All commands support these common configuration options:

```yaml
- type: "shell"
  name: "Command Name"                    # Descriptive name for the command
  working_directory: "/custom/path"       # Override working directory
  runs_on_condition: "success"           # When to run: success, error, always, exit-codes
  runs_on_exit_codes: [0, 1]            # Specific exit codes (when runs_on_condition is exit-codes)
  env:                                   # Environment variables
    KEY: "value"
    PATH: "/custom/bin:$PATH"
```

### Conditional Execution

Control when commands run based on previous results:

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
```

## 🔧 Extending porch

### Creating Custom Commands

Implement the `Commander` interface to create custom command types:

```go
package mycommand

import (
    "context"
    "github.com/matt-FFFFFF/porch/internal/commands"
    "github.com/matt-FFFFFF/porch/internal/runbatch"
)

type MyCommander struct{}

func (c *MyCommander) Create(ctx context.Context, payload []byte) (runbatch.Runnable, error) {
    // Parse payload (YAML command definition)
    // Create and return a Runnable implementation
    return myRunnable, nil
}

// Register the command in init()
func init() {
    commands.Register("mycommand", &MyCommander{})
}
```

### Implementing Runnable

Create custom execution logic by implementing the `Runnable` interface:

```go
type MyRunnable struct {
    // Command configuration
}

func (r *MyRunnable) Run(ctx context.Context) runbatch.Results {
    // Implement execution logic
    // Handle context cancellation
    // Return structured results
}

func (r *MyRunnable) SetCwd(cwd string) {
    // Set working directory
}

func (r *MyRunnable) InheritEnv(env map[string]string) {
    // Inherit environment variables
}
```

## 🏗️ Advanced Examples

### Complex CI/CD Pipeline

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

### Working with Temporary Directories

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

## 🎨 Output and Results

### Result Structure

Every command execution produces detailed results:

```go
type Result struct {
    ExitCode  int           // Command exit code
    Error     error         // Any execution error
    StdOut    []byte        // Standard output
    StdErr    []byte        // Standard error
    Label     string        // Command label/name
    Children  Results       // Nested command results
}
```

### Pretty-Printed Output

porch automatically generates beautiful tree-structured output:

```text
✓ Build and Test Workflow
  ✓ Setup Environment (0ms)
  ✓ Quality Checks (2.1s)
    ✓ Run Tests (1.8s)
    ✓ Run Linter (1.2s)
    ✗ Security Scan (0.5s) - exit code 1
  ✓ Build Process (1.5s)
    ✓ Build for Linux (0.8s)
    ✓ Build for macOS (0.7s)
```

### JSON Export

Results can be exported as JSON for programmatic analysis:

```bash
porch show results.gob --format=json
```

## 🚦 Signal Handling

porch provides sophisticated signal handling for graceful shutdown:

- **First Signal (SIGINT/SIGTERM)**: Initiates graceful shutdown
  - Running commands receive the signal for cleanup
  - No new commands are started
  - Context remains active for cleanup operations

- **Second Signal**: Forces immediate termination
  - Context is cancelled
  - All running processes are killed
  - Execution stops immediately

This allows for proper cleanup and prevents data corruption during interruption.

## 🧪 Testing

Run the test suite:

```bash
# Run all tests
make test

# Run tests with coverage
make testcover

# Run linter
make lint
```

The project includes comprehensive tests covering:

- Unit tests for all components
- Integration tests for command execution
- Error handling scenarios
- Signal handling behavior
- Result formatting and output

## 🤝 Contributing

Contributions are welcome! Here's how to get started:

1. **Fork the repository**

   ```bash
   git clone https://github.com/yourusername/porch.git
   ```

2. **Create a feature branch**

   ```bash
   git checkout -b feature/amazing-feature
   ```

3. **Make your changes**

   - Add tests for new functionality
   - Update documentation as needed
   - Follow Go best practices and conventions

4. **Run tests and linting**

   ```bash
   make test
   make lint
   ```

5. **Commit and push**

   ```bash
   git commit -m 'Add amazing feature'
   git push origin feature/amazing-feature
   ```

6. **Open a Pull Request**

### Development Guidelines

- Write comprehensive tests for new features
- Follow the existing code style and patterns
- Update documentation for user-facing changes
- Use meaningful commit messages
- Keep pull requests focused and atomic

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/) and the amazing Go ecosystem
- CLI powered by [urfave/cli](https://github.com/urfave/cli)
- YAML parsing with [goccy/go-yaml](https://github.com/goccy/go-yaml)
- Beautiful JSON output with [TylerBrock/colorjson](https://github.com/TylerBrock/colorjson)

---

**porch** - Making process orchestration simple, powerful, and beautiful. 🏠
