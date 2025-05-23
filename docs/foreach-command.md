# ForEach Command Type

The `foreach` command type allows you to execute a set of commands for each item returned by an item provider function.

## YAML Structure

```yaml
name: "ForEach Example"
description: "Example of foreach command"
commands:
  - type: "foreach"
    name: "Process Go Files"
    itemProvider: "list-go-files"
    mode: "serial"  # or "parallel"
    itemVariable: "CURRENT_FILE"  # optional: environment variable for the current item
    commands:
      - type: "command"
        name: "Process File"
        command: "oscommand"
        args:
          - "go"
          - "vet"
          - "${CURRENT_FILE}"
```

## Item Providers

Item providers are registered functions that generate lists of items to iterate over. They are registered in `internal/registry/itemProviders.go`:

### Built-in Item Providers

- `example`: Returns a fixed list of items: "item1", "item2", "item3"
- `list-go-files`: Lists all `.go` files in the current directory
- `list-yaml-files`: Lists all `.yaml` files in the current directory
- `list-directories`: Lists all directories in the current directory
- `comma-separated`: Splits a comma-separated string into individual items

### Creating Custom Item Providers

You can create custom item providers by following these steps:

1. Create a function that conforms to the `ItemsProviderFunc` signature:

```go
func(ctx context.Context, workingDirectory string) ([]string, error)
```

2. Register your provider in the `DefaultItemProviderRegistry`:

```go
import "github.com/matt-FFFFFF/avmtool/internal/registry"

func init() {
    registry.DefaultItemProviderRegistry.Register("my-provider", func(ctx context.Context, workingDir string) ([]string, error) {
        // Your implementation here
        return []string{"item1", "item2", "item3"}, nil
    })
}
```

## Execution Modes

The `foreach` command supports two execution modes:

- `serial`: Execute commands for each item one after another (default)
- `parallel`: Execute commands for all items in parallel

## Environment Variables

If `itemVariable` is specified, each command execution will have an environment variable set to the current item. This allows you to reference the current item in your commands.

## Example Use Cases

- Lint all Go files in a directory
- Run tests in each subdirectory
- Process a list of configuration files
- Apply operations to a list of resources (e.g., Kubernetes resources)
