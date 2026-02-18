.PHONY: all build test run clean fmt lint

# Build variables
BINARY_NAME=rsshub-go
CMD_DIR=cmd/server
BUILD_DIR=build

all: build

# Build the server binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Run the server
run:
	go run ./$(CMD_DIR)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	go fmt ./...
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
	fi

# Lint (requires golangci-lint)
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Generate route list (example CLI tool)
gen-routes:
	go run ./examples/cli/list-routes

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build the server binary"
	@echo "  run         - Run the server directly"
	@echo "  test        - Run all tests"
	@echo "  test-coverage - Run tests and generate coverage report"
	@echo "  fmt         - Format source code"
	@echo "  lint        - Lint source code"
	@echo "  clean       - Remove build artifacts"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  help        - Show this help message"
