default:
	@echo "No default target specified. Please specify a target."

testcover:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test:
	@echo "Running tests..."
	@go test -race ./...

lint:
	@echo "Running linter..."
	@golangci-lint run --fix
