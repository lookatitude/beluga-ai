GO ?= go
GOTEST ?= $(GO) test
GOLINT ?= golangci-lint
GOFMT ?= gofmt
GOIMPORTS ?= goimports
PKGSITE ?= pkgsite
RM ?= rm -f

.DEFAULT_GOAL := help

.PHONY: help build test test-verbose integration-test coverage lint lint-fix fmt tidy fuzz bench docs docs-website clean check

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

fuzz: ## Run all fuzz tests (30s each)
	@set -e; \
	targets=$$($(GO) test ./... -list '^Fuzz' 2>/dev/null | awk '/^Fuzz/ {print $$1}'); \
	if [ -z "$$targets" ]; then \
		echo "No fuzz tests found."; \
		exit 0; \
	fi; \
	for t in $$targets; do \
		echo "Running fuzz target $$t for 30s..."; \
		$(GO) test ./... -run=^$$ -fuzz=$$t -fuzztime=30s; \
	done

bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

docs: ## Start local pkgsite on :8080
	$(GO) install golang.org/x/pkgsite/cmd/pkgsite@latest
	$(PKGSITE) -http=:8080 .

docs-website: ## Start docs website locally
	cd docs/website && yarn dev

clean: ## Remove generated artifacts
	$(RM) coverage.out coverage.html

check: lint test tidy ## Run pre-commit checks (lint + test + tidy)
