.PHONY: build
build:
	@echo "Building the project..."
	@go build -o porch ./cmd/porch

.PHONY: install
install:
	@echo "Installing..."
	@go install ./cmd/porch

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
