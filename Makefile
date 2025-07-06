.PHONY: build
build:
	@echo "Building the project..."
	@go build -o porch ./cmd/porch

.PHONY: testcover
testcover:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test
test:
	@echo "Running tests..."
	@go test ./...

.PHONY: testrace
testrace:
	@echo "Running tests with race detection..."
	@go test -race ./...

.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run

.PHONY: precommit
precommit:
	@echo "Running pre-commit checks..."
	@go fmt ./...
	@go vet ./...
	@golangci-lint run --fix

# Language Server targets
.PHONY: build-lsp
build-lsp:
	@echo "Building Porch HCL Language Server..."
	@cd tools && make build-lsp

.PHONY: build-extension
build-extension:
	@echo "Building VSCode Extension..."
	@cd tools && make build-extension

.PHONY: tools-dev-setup
tools-dev-setup:
	@echo "Setting up language support tools development environment..."
	@cd tools && make dev-setup

.PHONY: tools-clean
tools-clean:
	@echo "Cleaning language support tools..."
	@cd tools && make clean

# Combined build target
.PHONY: build-all
build-all: build build-lsp
	@echo "All components built successfully"

# Release preparation
.PHONY: release-check
release-check: test testrace lint build-all
	@echo "Release checks completed successfully"
