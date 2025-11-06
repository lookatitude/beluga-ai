# Makefile for Beluga AI Framework
# Standard build, test, and quality assurance targets

.PHONY: help build test test-race test-coverage lint fmt vet security clean all install-tools bench

# Variables
GO_VERSION := 1.24
BIN_DIR := bin
COVERAGE_DIR := coverage
COVERAGE_FILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Beluga AI Framework - Available Targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build all packages
	@echo "Building all packages..."
	@go build -v ./...

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	@go test -race -v ./...

test-coverage: ## Generate test coverage report
	@echo "Generating test coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@go tool cover -func=$(COVERAGE_FILE)
	@echo ""
	@echo "Coverage report generated: $(COVERAGE_HTML)"

test-coverage-ci: ## Generate test coverage for CI (JSON output)
	@echo "Generating test coverage for CI..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@go tool cover -func=$(COVERAGE_FILE)

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2; \
	fi
	@golangci-lint run ./...

fmt: ## Format code with gofmt
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -l -w .; \
	else \
		echo "gofumpt not installed (optional, for stricter formatting)"; \
	fi

fmt-check: ## Check if code is properly formatted
	@echo "Checking code formatting..."
	@if [ $$(gofmt -l . | wc -l) -gt 0 ]; then \
		echo "Code is not properly formatted. Run 'make fmt' to fix."; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "Code is properly formatted."

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

security: ## Run security scans (gosec and govulncheck)
	@echo "Running security scans..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "gosec not found. Installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -fmt=json -out=$(COVERAGE_DIR)/gosec-report.json ./... || true
	@gosec ./...
	@echo ""
	@echo "Running govulncheck..."
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "govulncheck not found. Installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@govulncheck ./...

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -rf $(COVERAGE_DIR)
	@go clean -cache
	@go clean -testcache
	@echo "Clean complete."

all: fmt-check vet lint test ## Run all checks (fmt, vet, lint, test)

ci: fmt-check vet lint test-coverage-ci security ## Run all CI checks

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

bench-cmp: ## Compare benchmarks (requires benchstat)
	@echo "Running benchmark comparison..."
	@if ! command -v benchstat >/dev/null 2>&1; then \
		echo "benchstat not found. Installing..."; \
		go install golang.org/x/perf/cmd/benchstat@latest; \
	fi
	@echo "Run 'go test -bench=. -benchmem -count=5 > old.txt' first"
	@echo "Then make changes and run 'go test -bench=. -benchmem -count=5 > new.txt'"
	@echo "Finally run 'benchstat old.txt new.txt'"

install-tools: ## Install all required tools
	@echo "Installing required tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install mvdan.cc/gofumpt@latest
	@go install golang.org/x/perf/cmd/benchstat@latest
	@echo "Tools installed successfully."

verify: fmt-check vet lint test ## Verify code quality (alias for 'all')

# Development helpers
mod-tidy: ## Run go mod tidy
	@echo "Running go mod tidy..."
	@go mod tidy
	@go mod verify

mod-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

mod-download: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download

# Check Go version
check-go-version: ## Check if Go version matches requirements
	@echo "Checking Go version..."
	@GO_VERSION_CURRENT=$$(go version | awk '{print $$3}' | sed 's/go//' | cut -d. -f1,2); \
	if [ "$$GO_VERSION_CURRENT" != "$(GO_VERSION)" ]; then \
		echo "Warning: Go version mismatch. Expected $(GO_VERSION), found $$GO_VERSION_CURRENT"; \
		exit 1; \
	fi
	@echo "Go version check passed: $$GO_VERSION_CURRENT"

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	@go doc -all ./...

# License check
license-check: ## Check license compatibility
	@echo "Checking license compatibility..."
	@if ! command -v go-licenses >/dev/null 2>&1; then \
		echo "go-licenses not found. Installing..."; \
		go install github.com/google/go-licenses@latest; \
	fi
	@go-licenses check ./... || true
	@go-licenses report ./... > $(COVERAGE_DIR)/licenses.txt || true
	@echo "License report generated: $(COVERAGE_DIR)/licenses.txt"

