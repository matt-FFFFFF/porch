
# Developer Guide for Porch

## Extending Porch

Porch is designed to be extensible, allowing developers to add new commands and features easily. Here’s how you can extend Porch:

### Adding a New YAML Command

1. Create a directory for your command under `internal/commands`.

1. Create a definition.go file for your command, e.g., `mycommand.go` that inherits the `*commands.BaseCommand` type.

1. Create a `register.go` file to register the command with the supplied registry:

    ```go
    const commandType = "mycommand"

    // Register registers the command in the given registry.
    func Register(r commandregistry.Registry) {
      err := r.Register(commandType, &Commander{})
      if err != nil {
        panic(err)
      }
    }
    ```

1. Implement the `commands.Commander` interface to define your command's behavior.
  This should create a type that implements the `runbatch.Runnable` interface.

1. Implement the `schema.Provider` interface to provide information for command help.

1. Implement the `schema.Writer` interface to handle writing command help.

1. Write tests for your command in a `_test.go` file within the same directory.

1. Write integration tests in a `_integration_test.go` file to ensure your command works as expected when executed.

1. Update the `cmd/porch/main.go` package to register your new command.

## 🧪 Testing

Run the test suite:

```bash
# Run all tests
make test

# Run tests with coverage
make testcover

# Run linter
make lint

# Run tests with race detection
make testrace
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
