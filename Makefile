.PHONY: all build test run clean fmt lint install-config gen-routes-imports new-route verify-routes verify-routes-strict ci-local

# Build variables
BINARY_NAME=syndi
CMD_DIR=cmd
BUILD_DIR=build
CONFIG_DIR=/etc/syndi

all: build

# Generate route imports (auto-run before build)
gen-routes-imports:
	@echo "Generating route imports..."
	@go run scripts/generate-routes.go

# Scaffold a new route.
# Usage:
# make new-route NS=github FILE=stars ROUTE_PATH=stars/:owner ROUTE_NAME="GitHub Stars" EXAMPLE=github/stars/octocat
new-route:
	@if [ -z "$(NS)" ] || [ -z "$(FILE)" ] || [ -z "$(ROUTE_PATH)" ] || [ -z "$(ROUTE_NAME)" ] || [ -z "$(EXAMPLE)" ]; then \
		echo "Usage: make new-route NS=<namespace> FILE=<file> ROUTE_PATH=<path> ROUTE_NAME=<name> EXAMPLE=<example>"; \
		echo "Example: make new-route NS=github FILE=stars ROUTE_PATH=stars/:owner ROUTE_NAME=\"GitHub Stars\" EXAMPLE=github/stars/octocat"; \
		exit 1; \
	fi
	@./scripts/new-route.sh "$(NS)" "$(FILE)" "$(ROUTE_PATH)" "$(ROUTE_NAME)" "$(EXAMPLE)"

# Build the server binary
build: gen-routes-imports
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Run the server
run: gen-routes-imports
	go run ./$(CMD_DIR)

# Run the server with hot reload on file changes (requires air: go install github.com/air-verse/air@latest)
dev: gen-routes-imports
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not installed, falling back to 'go run' (no hot reload)"; \
		echo "install with: go install github.com/air-verse/air@latest"; \
		go run ./$(CMD_DIR); \
	fi

# Run tests
test:
	go test -v ./...

# Verify route metadata consistency
verify-routes: gen-routes-imports
	go run ./scripts/verify-routes

# Verify route metadata consistency (warnings are treated as errors)
verify-routes-strict: gen-routes-imports
	go run ./scripts/verify-routes --strict

# Live-verify every registered route via a running server (slow, needs network)
verify-all:
	./scripts/verify-all.sh

# Run local checks equivalent to CI workflow
ci-local:
	@echo "Running local CI checks..."
	$(MAKE) gen-routes-imports
	git diff --exit-code -- cmd/routes_gen.go scripts/verify-routes/register_gen.go
	$(MAKE) verify-routes-strict
	go test ./...
	$(MAKE) build

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

# Install configuration file to /etc/syndi/
install-config:
	@echo "Installing configuration file to $(CONFIG_DIR)/..."
	@mkdir -p $(CONFIG_DIR)
	@if [ -f "$(CONFIG_DIR)/config.yaml" ]; then \
		echo "Backing up existing config.yaml to config.yaml.bak"; \
		cp $(CONFIG_DIR)/config.yaml $(CONFIG_DIR)/config.yaml.bak; \
	fi
	@cp config.yaml $(CONFIG_DIR)/config.yaml
	@echo "Configuration file installed to $(CONFIG_DIR)/config.yaml"
	@echo "Edit this file to customize your RSSHub Go configuration"

# Generate route list (example CLI tool)
gen-routes:
	go run ./examples/cli/list-routes

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the server binary"
	@echo "  run           - Run the server directly"
	@echo "  dev           - Run with hot reload (air)"
	@echo "  new-route     - Scaffold a new route file and test skeleton"
	@echo "  verify-routes - Verify route metadata consistency"
	@echo "  verify-routes-strict - Verify routes and fail on warnings"
	@echo "  ci-local      - Run the same checks as CI locally"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests and generate coverage report"
	@echo "  fmt           - Format source code"
	@echo "  lint          - Lint source code"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  install-config - Install configuration file to /etc/syndi/"
	@echo "  help          - Show this help message"
