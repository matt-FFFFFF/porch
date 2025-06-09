### Agent behaviour (that's you!)

- The agent should ask before making any changes to the codebase.
  If the prompt does not contain a specific request for code changes,
  the agent should respond with guidance and code snippets on how to implement the requested feature or fix the issue.
  It should then ask if the user would like to proceed with the changes to the codebase.

### Code Style

- Use `gofmt` to format your Go code.
- Run `golangci-lint run` to check for linting issues, use with the `--fix` flag to automatically fix issues where possible.
  - Inspect the `.golangci.yml` file in the root of this repo for specific linting rules.
- Each package should have a `doc.go` file that contains package-level documentation using the `// Package <name> ...` comment format.
- Each go file must have the following header, followed by a blank line and then the package keyword:
    // Copyright (c) matt-FFFFFF 2025. All rights reserved.
    // SPDX-License-Identifier: MIT
- Complex nested if statements should be avoided. Use switch statements or early returns instead.
- All exported functions and types should have comments that explain their purpose.
- Keep the happy path left aligned, and use indentation for error handling or complex logic.

### Error Handling

- Use static error types for common errors, such as `ErrCommandFailed`, `ErrCommandNotFound`, etc.
- Wrap errors using `errors.Join` (preferred) or `fmt.Errorf("additional context: %w", err)`.

### Logging

- The logger should be configured to use the `ctxlog` package (github.com/matt-FFFFFF/porch/internal/ctxlog), which is based on `slog`.
- Each function should accept a `context.Context` parameter for logging and cancellation.
- Use structured logging with key-value pairs for better context.

### Context

- All functions that perform operations should accept a `context.Context` parameter in the first position to allow for cancellation and timeouts, as well as logging.

### Testing

- Test assertions and requirements should use the `"github.com/stretchr/testify/assert"`,
and `"github.com/stretchr/testify/require"` packages.
- Use table-driven tests for better organization and readability.
- Unit tests should reside in a file named as per the file containing the code being tested, with the suffix `_test.go`.
- Integration tests should reside in a file named as per the file or package containing the code being tested, with the suffix `_integration_test.go`.
- This package uses concurrency so tests should be run with the `-race` flag to ensure they are thread-safe. This should be done after verifying unit tests pass in isolation.

### Building and Running

- The binary is built from the `./cmd` package.
- Use `make build` to build the project.
