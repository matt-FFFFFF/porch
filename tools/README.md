# Porch HCL Language Support Tools

This directory contains tools for providing language support for Porch HCL configuration files.

## Components

### Language Server (`language-server/`)

A Language Server Protocol (LSP) implementation that provides:
- Syntax validation for `.porch.hcl` files
- Auto-completion for workflow blocks, command types, and HCL functions
- Hover documentation for functions and attributes
- Real-time diagnostics

**Features:**
- Support for all Porch block types: `workflow`, `variable`, `locals`, `dynamic`
- Command type validation: `shell`, `pwsh`, `serial`, `parallel`, `foreachdirectory`, `copycwdtotemp`
- Full integration with `hclfuncs` library providing 30+ HCL functions including:
  - String functions: `length()`, `replace()`, `startswith()`, `endswith()`
  - Date/time: `timestamp()`, `timecmp()`
  - Encoding: `base64encode()`, `urlencode()`, `yaml2json()`
  - Collections: `alltrue()`, `anytrue()`, `sum()`
  - Utilities: `uuid()`, `env()`, `cidrcontains()`

### VSCode Extension (`vscode-extension/`)

A Visual Studio Code extension that provides:
- Syntax highlighting for `.porch.hcl` files
- Code snippets for common patterns
- Integration with the language server
- File creation commands
- Language configuration (auto-closing brackets, comments, etc.)

## Building

### Language Server

```bash
cd language-server
go build -o porch-lsp ./cmd/porch-lsp
```

### VSCode Extension

```bash
cd vscode-extension
npm install
npm run compile
```

## Installation

### Language Server

#### From GitHub Releases (Recommended)
Pre-built binaries for the language server are included in the main [Porch releases](https://github.com/matt-FFFFFF/porch/releases). Download the appropriate `porch-lsp_*` archive for your platform.

#### Build from Source
1. Build the language server binary
2. Place it in your PATH or specify the path in VSCode settings

```bash
make build-lsp
```

### VSCode Extension

#### Development
1. Compile the extension
2. Press F5 in VSCode to open a new Extension Development Host window
3. Open a `.porch.hcl` file to activate the extension

#### Production
```bash
cd vscode-extension
npm install -g vsce
vsce package
vsce publish
```

## Usage

### Creating a Porch HCL File

Use the Command Palette (`Cmd+Shift+P`) and run "Porch HCL: Create New File" to create a new file with a template.

### Language Features

- **Auto-completion**: Type block names, attributes, or function names and press `Ctrl+Space`
- **Snippets**: Type prefixes like `workflow`, `shell`, `parallel`, etc.
- **Hover Documentation**: Hover over functions to see their documentation
- **Syntax Validation**: Real-time error highlighting for invalid syntax

### Configuration

In VSCode settings:

```json
{
  "porch-hcl.server.path": "porch-lsp",
  "porch-hcl.server.args": [],
  "porch-hcl.trace.server": "off"
}
```

## Example Porch HCL File

```hcl
# Example workflow configuration
variable "environment" {
  description = "Target environment"
  type        = string
  default     = "development"
}

locals {
  build_env = {
    GO_ENV     = var.environment
    BUILD_TIME = timestamp()
    VERSION    = "1.0.0"
  }
}

workflow "build_and_test" {
  name        = "Build and Test Pipeline"
  description = "Complete build and test workflow"

  command {
    type = "shell"
    name = "Environment Setup"
    env  = local.build_env
    command_line = "echo 'Building for ${var.environment}'"
  }

  command {
    type = "parallel"
    name = "Parallel Tests"

    command {
      type = "shell"
      name = "Unit Tests"
      command_line = "go test ./..."
    }

    command {
      type = "shell"
      name = "Integration Tests"
      command_line = "go test -tags=integration ./..."
    }
  }
}
```

## Supported HCL Functions

The language server provides completion and documentation for all functions from the `hclfuncs` library:

- **String Functions**: `length`, `replace`, `startswith`, `endswith`, `strcontains`
- **Collection Functions**: `alltrue`, `anytrue`, `index`, `matchkeys`, `transpose`, `sum`
- **Encoding Functions**: `textencodebase64`, `textdecodebase64`, `urlencode`, `urldecode`, `yaml2json`
- **Date/Time Functions**: `timestamp`, `timecmp`, `legacyisotime`, `legacystrftime`
- **Utility Functions**: `uuid`, `type`, `env`, `cidrcontains`
- **Validation Functions**: `semvercheck`, `compliment`
- **Security Functions**: `sensitive`, `nonsensitive`, `issensitive`

## Development

### Language Server

The language server is built with Go and uses:
- `github.com/hashicorp/hcl/v2` for HCL parsing
- `github.com/lonegunmanb/hclfuncs` for HCL functions
- LSP protocol for editor communication

### VSCode Extension

The extension is built with TypeScript and uses:
- `vscode-languageclient` for LSP communication
- TextMate grammars for syntax highlighting
- JSON snippets for code completion

## Contributing

1. Make changes to the language server or extension
2. Test with sample `.porch.hcl` files
3. Build and verify functionality
4. Submit a pull request

## License

MIT - See the main Porch repository license.
