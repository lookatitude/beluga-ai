GO ?= go
GOTEST ?= $(GO) test
GOLINT ?= golangci-lint
GOFMT ?= gofmt
GOIMPORTS ?= goimports
PKGSITE ?= pkgsite
RM ?= rm -f

# Tool versions pinned to match CI (.github/workflows/_ci-checks.yml and
# _security-checks.yml). `make tools` installs exactly these.
GOLANGCI_LINT_VERSION ?= v2.15.0
GOSEC_VERSION ?= v2.30.0
GOVULNCHECK_VERSION ?= v1.9.0
GO_LICENSES_VERSION ?= v2.0.1
FUZZ_TIME ?=

.DEFAULT_GOAL := help

.PHONY: help build test test-verbose integration-test coverage lint lint-fix fmt tidy fuzz bench docs docs-providers clean check tools security licenses test-scripts ci

help: ## Show available targets
	@echo "Beluga AI development targets:"
	@awk 'BEGIN {FS = ":.*## "; printf "\nUsage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_.-]+:.*## / { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build all Go packages
	$(GO) build ./...

test: ## Run unit tests with race detector
	$(GOTEST) -race ./...

test-verbose: ## Run unit tests with verbose output
	$(GOTEST) -race -v ./...

integration-test: ## Run integration tests
	$(GOTEST) -race -tags integration ./...

coverage: ## Generate coverage report and HTML output
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -func=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"
	@xdg-open coverage.html >/dev/null 2>&1 || true

lint: ## Run static analysis and lint checks
	$(GO) vet ./...
	$(GOLINT) run

lint-fix: ## Run linter with automatic fixes
	$(GOLINT) run --fix

fmt: ## Format code with gofmt and goimports
	$(GOFMT) -s -w .
	$(GOIMPORTS) -w .

tidy: ## Tidy Go modules and ensure no module drift
	$(GO) mod tidy
	@git diff --exit-code go.mod go.sum

fuzz: ## Run every fuzz target, one package at a time (FUZZ_TIME, default 30s)
	GO=$(GO) scripts/fuzz.sh $(FUZZ_TIME)

bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

docs: ## Start local pkgsite on :8080
	$(GO) install golang.org/x/pkgsite/cmd/pkgsite@v0.6.0
	$(PKGSITE) -http=:8080 .

clean: ## Remove generated artifacts
	$(RM) coverage.out coverage.html

docs-providers: ## Check provider catalog is up-to-date (run in CI)
	@tmp=$$(mktemp); trap 'rm -f "$$tmp"' EXIT; \
	$(GO) run ./internal/tools/provider-catalog > "$$tmp" && \
	diff -u docs/reference/providers.md "$$tmp" || { echo "ERROR: docs/reference/providers.md is stale. Run: go run ./internal/tools/provider-catalog > docs/reference/providers.md"; exit 1; }

check: lint test tidy ## Run pre-commit checks (lint + test + tidy)

tools: ## Install the CI-pinned golangci-lint, gosec, govulncheck and go-licenses
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	$(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	$(GO) install github.com/google/go-licenses/v2@$(GO_LICENSES_VERSION)

security: ## Run the blocking Go security scanners (gosec, govulncheck)
	gosec -quiet ./...
	govulncheck ./...

licenses: ## Run the license compliance gate used by CI
	scripts/check-licenses.sh

test-scripts: ## Run the tests for scripts/
	scripts/fuzz_test.sh
	scripts/release-version_test.sh

ci: lint build tidy docs-providers test-scripts test integration-test security licenses ## Mirror the blocking CI checks locally (Trivy, Gitleaks and CodeQL run only in CI)
