# Makefile for Beluga AI Framework
# Standard build, test, and quality assurance targets

.PHONY: help build test test-unit test-integration test-race test-coverage test-coverage-threshold lint lint-fix fmt vet security security-full clean all install-tools install-system-tools bench ci-local

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
	@go build -v $$(go list ./... | grep -v -E '(specs|examples)')

test: ## Run all tests
	@echo "Running tests..."
	@go test -v $$(go list ./... | grep -v -E '(specs|examples)')

test-unit: ## Run unit tests only (pkg packages, excluding integration tests)
	@echo "Running unit tests..."
	@go test -v -race $$(go list ./pkg/... | grep -v -E '(specs|examples)')

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -race -timeout=15m ./tests/integration/...

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	@go test -race -v $$(go list ./... | grep -v -E '(specs|examples)')

test-coverage: ## Generate test coverage report
	@echo "Generating test coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_FILE) -covermode=atomic $$(go list ./... | grep -v -E '(specs|examples)')
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@go tool cover -func=$(COVERAGE_FILE)
	@echo ""
	@echo "Coverage report generated: $(COVERAGE_HTML)"

test-coverage-ci: ## Generate test coverage for CI (JSON output)
	@echo "Generating test coverage for CI..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_FILE) -covermode=atomic $$(go list ./... | grep -v -E '(specs|examples)')
	@go tool cover -func=$(COVERAGE_FILE)

test-coverage-threshold: ## Check if coverage meets 80% threshold
	@echo "Checking coverage threshold (80%)..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_FILE) -covermode=atomic $$(go list ./... | grep -v -E '(specs|examples)') > /dev/null 2>&1
	@pct=$$(go tool cover -func=$(COVERAGE_FILE) | tail -n1 | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$pct" ]; then \
		echo "❌ Failed to calculate coverage"; \
		exit 1; \
	fi; \
	threshold=80; \
	if awk "BEGIN {exit !($$pct < $$threshold)}"; then \
		echo "❌ Coverage $$pct% is below minimum $$threshold%"; \
		go tool cover -func=$(COVERAGE_FILE) | tail -n1; \
		exit 1; \
	else \
		echo "✅ Coverage $$pct% meets minimum $$threshold% requirement"; \
		go tool cover -func=$(COVERAGE_FILE) | tail -n1; \
	fi

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.60.1; \
	fi
	@PATH="$$PATH:$$(go env GOPATH)/bin" golangci-lint run $$(go list ./... | grep -v -E '(specs|examples)') || true

lint-fix: ## Run golangci-lint with auto-fix
	@echo "Running golangci-lint with auto-fix..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.60.1; \
	fi
	@if [ -n "$(PKG)" ]; then \
		echo "Fixing lint errors in $(PKG)..."; \
		PATH="$$PATH:$$(go env GOPATH)/bin" golangci-lint run --fix $(PKG) || true; \
	else \
		echo "Fixing lint errors in all packages..."; \
		PATH="$$PATH:$$(go env GOPATH)/bin" golangci-lint run --fix $$(go list ./... | grep -v -E '(specs|examples)') || true; \
	fi

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
	@go vet $$(go list ./... | grep -v -E '(specs|examples)')

security: ## Run security scans (gosec, govulncheck, and gitleaks)
	@echo "Running security scans..."
	@mkdir -p $(COVERAGE_DIR)
	@echo "Running gosec..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "gosec not found. Installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -fmt=json -out=$(COVERAGE_DIR)/gosec-report.json $$(go list ./... | grep -v -E '(specs|examples)') || true
	@gosec $$(go list ./... | grep -v -E '(specs|examples)') || true
	@echo ""
	@echo "Running govulncheck..."
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "govulncheck not found. Installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@govulncheck $$(go list ./... | grep -v -E '(specs|examples)') 2>&1 | tee $(COVERAGE_DIR)/govulncheck-report.txt || true
	@echo ""
	@echo "Running gitleaks..."
	@if ! command -v gitleaks >/dev/null 2>&1; then \
		echo "gitleaks not found. Installing..."; \
		if [ -f ./scripts/install-gitleaks.sh ]; then \
			./scripts/install-gitleaks.sh || (echo "Failed to install gitleaks. Please install manually from https://github.com/gitleaks/gitleaks"; exit 1); \
		else \
			echo "⚠️  gitleaks install script not found. Please install manually:"; \
			echo "   Linux: wget -q https://github.com/gitleaks/gitleaks/releases/download/v8.18.0/gitleaks_8.18.0_linux_x64.tar.gz && tar -xzf gitleaks_8.18.0_linux_x64.tar.gz && chmod +x gitleaks && sudo mv gitleaks /usr/local/bin/"; \
			exit 1; \
		fi; \
	fi
	@gitleaks detect --no-banner --redact --config=.gitleaks.toml --report-path=$(COVERAGE_DIR)/gitleaks-report.json || true
	@if [ -f $(COVERAGE_DIR)/gitleaks-report.json ] && [ -s $(COVERAGE_DIR)/gitleaks-report.json ] && [ "$$(cat $(COVERAGE_DIR)/gitleaks-report.json)" != "[]" ]; then \
		echo "❌ Secrets detected by gitleaks"; \
		gitleaks detect --no-banner --redact --config=.gitleaks.toml; \
		exit 1; \
	fi
	@echo "✅ No secrets detected"

security-full: security ## Run all security scans including Trivy (requires Docker)
	@echo ""
	@echo "Running Trivy (optional, requires Docker or Trivy binary)..."
	@if command -v trivy >/dev/null 2>&1; then \
		echo "Running Trivy file system scan..."; \
		trivy fs --severity CRITICAL,HIGH --skip-dirs specs,examples,docs,website . || true; \
	elif command -v docker >/dev/null 2>&1; then \
		echo "Running Trivy via Docker..."; \
		docker run --rm -v $$(pwd):/app -w /app aquasec/trivy:latest fs --severity CRITICAL,HIGH --skip-dirs specs,examples,docs,website . || true; \
	else \
		echo "⚠️  Trivy not available (install from https://aquasecurity.github.io/trivy/ or use Docker)"; \
	fi

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -rf $(COVERAGE_DIR)
	@go clean -cache
	@go clean -testcache
	@echo "Clean complete."

all: fmt-check vet lint test ## Run all checks (fmt, vet, lint, test)

ci: fmt-check vet lint test-coverage-ci security ## Run all CI checks

ci-local: ## Run all CI checks locally (matches CI workflow)
	@echo "🚀 Running comprehensive CI checks locally..."
	@echo ""
	@echo "📋 Step 1: Format check..."
	@$(MAKE) fmt-check
	@echo ""
	@echo "🔍 Step 2: Lint & Format..."
	@$(MAKE) lint
	@echo ""
	@echo "🔍 Step 3: Go vet..."
	@$(MAKE) vet
	@echo ""
	@echo "🔒 Step 4: Security scans..."
	@$(MAKE) security
	@echo ""
	@echo "🧪 Step 5: Unit tests..."
	@$(MAKE) test-unit
	@echo ""
	@echo "🔗 Step 6: Integration tests..."
	@$(MAKE) test-integration
	@echo ""
	@echo "📈 Step 7: Coverage check..."
	@$(MAKE) test-coverage-threshold
	@echo ""
	@echo "🔨 Step 8: Build verification..."
	@$(MAKE) build
	@echo ""
	@echo "✅ All CI checks passed!"

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

install-tools: ## Install all required Go tools
	@echo "Installing required Go tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.1
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install mvdan.cc/gofumpt@latest
	@go install golang.org/x/perf/cmd/benchstat@latest
	@echo "Go tools installed successfully."
	@echo ""
	@echo "Note: gitleaks is a system tool. Run 'make install-system-tools' to install it."

install-system-tools: ## Install system tools (gitleaks, jq)
	@echo "Installing system tools..."
	@if ! command -v gitleaks >/dev/null 2>&1; then \
		echo "Installing gitleaks..."; \
		if [ -f ./scripts/install-gitleaks.sh ]; then \
			./scripts/install-gitleaks.sh; \
		else \
			echo "⚠️  gitleaks install script not found. Please install manually from https://github.com/gitleaks/gitleaks"; \
		fi; \
	else \
		echo "✅ gitleaks already installed"; \
	fi
	@if ! command -v jq >/dev/null 2>&1; then \
		echo "Installing jq..."; \
		if command -v apt-get >/dev/null 2>&1; then \
			sudo apt-get update && sudo apt-get install -y jq; \
		elif command -v brew >/dev/null 2>&1; then \
			brew install jq; \
		elif command -v yum >/dev/null 2>&1; then \
			sudo yum install -y jq; \
		else \
			echo "⚠️  Please install jq manually for your system: https://stedolan.github.io/jq/download/"; \
		fi; \
	else \
		echo "✅ jq already installed"; \
	fi
	@echo "System tools installation complete."

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
	fi; \
	echo "Go version check passed: $$GO_VERSION_CURRENT"

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	@go doc -all ./...

docs-generate: ## Generate API documentation using gomarkdoc
	@echo "Generating API documentation..."
	@./scripts/generate-docs.sh

docs-verify: ## Verify API documentation is up to date
	@echo "Verifying API documentation..."
	@if ! ./scripts/generate-docs.sh > /tmp/docs-generated.md 2>&1; then \
		echo "Documentation generation failed"; \
		exit 1; \
	fi
	@if git diff --quiet website/docs/api/packages/ 2>/dev/null; then \
		echo "Documentation is up to date"; \
	else \
		echo "Documentation is out of date. Run 'make docs-generate' to update."; \
		git diff website/docs/api/packages/ || true; \
		exit 1; \
	fi

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

