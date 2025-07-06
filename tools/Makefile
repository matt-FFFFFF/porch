# Makefile for Porch Language Support Tools

.PHONY: all clean build-lsp build-extension install-extension package-extension test cross-compile-lsp

# Go build flags for size optimization
GO_BUILD_FLAGS := -ldflags="-s -w" -trimpath

# Default target
all: cross-compile-lsp build-extension

# Build the language server for current platform
build-lsp:
	@echo "Building Porch HCL Language Server..."
	cd language-server && go build $(GO_BUILD_FLAGS) -o porch-lsp ./cmd/porch-lsp
	@echo "Language server built successfully: language-server/porch-lsp"

# Cross-compile language server for all platforms
cross-compile-lsp:
	@echo "Cross-compiling Porch HCL Language Server for all platforms..."
	@mkdir -p vscode-extension/bin

	# macOS ARM64
	cd language-server && GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o ../vscode-extension/bin/porch-lsp-darwin-arm64 ./cmd/porch-lsp

	# macOS AMD64
	cd language-server && GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o ../vscode-extension/bin/porch-lsp-darwin-amd64 ./cmd/porch-lsp

	# Linux ARM64
	cd language-server && GOOS=linux GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o ../vscode-extension/bin/porch-lsp-linux-arm64 ./cmd/porch-lsp

	# Linux AMD64
	cd language-server && GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o ../vscode-extension/bin/porch-lsp-linux-amd64 ./cmd/porch-lsp

	# Windows ARM64
	cd language-server && GOOS=windows GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o ../vscode-extension/bin/porch-lsp-windows-arm64.exe ./cmd/porch-lsp

	# Windows AMD64
	cd language-server && GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o ../vscode-extension/bin/porch-lsp-windows-amd64.exe ./cmd/porch-lsp

	@echo "Language server binaries compiled for all platforms:"
	@ls -la vscode-extension/bin/

# Build the VSCode extension (depends on cross-compiled binaries)
build-extension: cross-compile-lsp
	@echo "Building VSCode Extension..."
	cd vscode-extension && rm -f *.vsix && npm run package && npx vsce package
	@echo "Extension built successfully"

# Install dependencies for VSCode extension
install-extension-deps:
	@echo "Installing VSCode Extension dependencies..."
	cd vscode-extension && npm install

# Run tests for the language server
test-lsp:
	@echo "Running language server tests..."
	cd language-server && go test ./...

# Run golangci-lint on the language server
lint-lsp:
	@echo "Running golangci-lint on language server..."
	cd language-server && golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f language-server/porch-lsp
	rm -rf vscode-extension/out
	rm -rf vscode-extension/node_modules
	rm -f vscode-extension/*.vsix
	@echo "Clean complete"
