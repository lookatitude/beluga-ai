# Contributing to Beluga-AI

We appreciate your interest in contributing to Beluga-AI! To ensure consistency and maintain a clear project history, we follow the Conventional Commits specification for all commit messages.

## Conventional Commits

All commit messages should adhere to the [Conventional Commits specification](https://www.conventionalcommits.org/en/v1.0.0/). This format allows for automated changelog generation and makes it easier to track features, fixes, and breaking changes.

A commit message should be structured as follows:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

The following types are commonly used:

*   **feat**: A new feature for the user (corresponds to a MINOR version bump when `release-please` runs).
*   **fix**: A bug fix for the user (corresponds to a PATCH version bump).
*   **docs**: Changes to documentation only.
*   **style**: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc).
*   **refactor**: A code change that neither fixes a bug nor adds a feature.
*   **perf**: A code change that improves performance.
*   **test**: Adding missing tests or correcting existing tests.
*   **build**: Changes that affect the build system or external dependencies (example scopes: gulp, broccoli, npm).
*   **ci**: Changes to our CI configuration files and scripts (example scopes: Travis, Circle, BrowserStack, SauceLabs).
*   **chore**: Other changes that don"t modify src or test files (e.g., updating dependencies).
*   **revert**: Reverts a previous commit.

### Scope

The scope provides additional contextual information and is contained within parentheses, e.g., `feat(parser): add ability to parse arrays`.

### Breaking Changes

Breaking changes MUST be indicated at the very beginning of the body or footer section of a commit. A breaking change MUST consist of the uppercase text `BREAKING CHANGE:`, followed by a summary of the breaking change. This will trigger a MAJOR version bump.

Example:

```
feat: allow provided config object to extend other configs

BREAKING CHANGE: `extends` key in config file is now used for extending other config files
```

### Examples

*   Commit message with no body:
    `docs: correct spelling of CHANGELOG`

*   Commit message with scope:
    `feat(lang): add polish language`

*   Commit message with a body:
    ```
    fix: correct minor typos in code

    see the issue for details on typos fixed

    Reviewed-by: Z
    Refs #133
    ```

## Release Process

This project uses [release-please](https://github.com/googleapis/release-please) to automate releases. When commits adhering to the Conventional Commits specification are merged into the `main` branch, `release-please` will automatically create a Pull Request proposing the next release version and updating the `CHANGELOG.md`.

Once this Release PR is merged, `release-please` will then tag the release and create a GitHub Release.

For pre-releases (like alpha, beta), ensure your commit messages are clear about the pre-release nature if applicable, though `release-please-config.json` is set up to handle alpha versions automatically.

## Local Development Setup

### Prerequisites

*   **Go 1.24+**: The framework requires Go 1.24 or later
*   **Make**: Required for build automation (available on most systems)
*   **Git**: Version control

### Initial Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/lookatitude/beluga-ai.git
   cd beluga-ai
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   go mod verify
   ```

3. **Install development tools:**
   ```bash
   make install-tools
   ```
   This installs:
   - `golangci-lint` - Comprehensive linter
   - `gosec` - Security scanner
   - `govulncheck` - Vulnerability checker
   - `gofumpt` - Stricter formatter
   - `benchstat` - Benchmark comparison tool

4. **Set up pre-commit hooks (recommended):**
   ```bash
   # Install pre-commit (requires Python)
   pip install pre-commit
   
   # Install git hooks
   pre-commit install
   
   # Or manually install hooks
   pre-commit install --hook-type pre-commit --hook-type pre-push
   ```

### Development Workflow

#### Running Tests

```bash
# Run all tests
make test

# Run tests with race detection
make test-race

# Generate coverage report
make test-coverage
# Opens coverage/coverage.html in browser

# Run integration tests
go test -v ./tests/integration/...

# Run benchmarks
make bench
```

#### Code Quality Checks

```bash
# Run all checks (format, vet, lint, test)
make all

# Or run individually:
make fmt          # Format code
make fmt-check     # Check formatting without fixing
make vet          # Run go vet
make lint         # Run golangci-lint
```

#### Security Scanning

```bash
# Run all security scans
make security

# This runs:
# - gosec (static security analysis)
# - govulncheck (dependency vulnerabilities)
```

#### Building

```bash
# Build all packages
make build

# Clean build artifacts
make clean
```

### Pre-commit Hooks

Pre-commit hooks automatically run checks before commits. They verify:
- Code formatting (gofmt, goimports)
- Linting (golangci-lint)
- Security (gosec)
- Tests (quick unit tests)
- File formatting (YAML, JSON, TOML)

To manually run hooks:
```bash
pre-commit run --all-files
```

### Making Changes

1. **Create a feature branch:**
   ```bash
   git checkout -b feat/your-feature-name
   ```

2. **Make your changes** following the [Development Guidelines](#development-guidelines)

3. **Run quality checks:**
   ```bash
   make all
   ```

4. **Run security scans:**
   ```bash
   make security
   ```

5. **Commit your changes** (hooks will run automatically):
   ```bash
   git add .
   git commit -m "feat(scope): your feature description"
   ```

6. **Push and create a Pull Request:**
   ```bash
   git push origin feat/your-feature-name
   ```

## Development Guidelines

### Code Quality Standards

*   **Go Best Practices**: Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
*   **Error Handling**: Use explicit error handling with proper context
*   **Concurrency**: Use channels and sync primitives safely
*   **Testing**: Write comprehensive unit and integration tests
*   **Documentation**: Document all exported functions and types
*   **Linting**: All code must pass `golangci-lint` checks
*   **Formatting**: Code must be formatted with `gofmt` or `gofumpt`

### Advanced Features Usage

#### Dependency Injection
When adding new components, use the DI container for better testability and flexibility:

```go
// Register your component in the DI container
container.Register(func(deps Dependency) (MyComponent, error) {
    return NewMyComponent(deps), nil
})
```

#### Context Propagation
Always propagate context through function calls for proper cancellation and tracing:

```go
func (c *MyComponent) Process(ctx context.Context, input interface{}) error {
    // Use context for timeouts, cancellation, and tracing
    span := monitoring.SpanFromContext(ctx)
    if span != nil {
        span.Log("Processing input", map[string]interface{}{"input_size": len(input)})
    }

    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Process input
    }

    return nil
}
```

#### Structured Logging
Use the structured logger for consistent, searchable logs:

```go
logger := monitoring.NewStructuredLogger("my-component")
logger.Info(ctx, "Operation completed", map[string]interface{}{
    "operation": "data_processing",
    "records_processed": 1000,
    "duration_ms": 150,
})
```

#### Metrics and Monitoring
Instrument your code with appropriate metrics:

```go
metrics := monitoring.NewMetricsCollector()
timer := metrics.StartTimer(ctx, "operation_duration", map[string]string{
    "component": "my-component",
    "operation": "process_data",
})
defer timer.Stop(ctx, "Data processing duration")

metrics.Counter(ctx, "operations_total", "Total operations", 1, map[string]string{
    "component": "my-component",
    "status": "success",
})
```

### Adding New Providers

When implementing new providers (LLMs, VectorStores, etc.), follow this pattern:

1. **Define Interface**: Ensure your provider implements the appropriate interface
2. **Factory Registration**: Register your provider in the DI container
3. **Configuration**: Add configuration structs with proper validation tags
4. **Error Handling**: Use proper error wrapping and context
5. **Testing**: Write comprehensive tests with mocks
6. **Documentation**: Document usage and configuration options

Example provider implementation:

```go
type MyProvider struct {
    config MyProviderConfig
    logger *monitoring.StructuredLogger
}

func NewMyProvider(config MyProviderConfig, logger *monitoring.StructuredLogger) *MyProvider {
    return &MyProvider{
        config: config,
        logger: logger,
    }
}

func (p *MyProvider) Process(ctx context.Context, input interface{}) (interface{}, error) {
    span := monitoring.SpanFromContext(ctx)
    if span != nil {
        span.SetTag("provider", "my-provider")
    }

    // Implementation with proper error handling and logging
    return result, nil
}
```

## Pull Requests

### Before Submitting

Before submitting a pull request, ensure:

1. **Your branch is up-to-date:**
   ```bash
   git checkout main
   git pull origin main
   git checkout your-branch
   git rebase main  # or merge main
   ```

2. **All checks pass:**
   ```bash
   make all          # Run all quality checks
   make security     # Run security scans
   make test-race    # Run tests with race detection
   ```

3. **Code quality:**
   - All code passes `golangci-lint` checks
   - Code is properly formatted (`make fmt-check`)
   - All tests pass (`make test`)
   - No security issues (`make security`)

4. **Documentation:**
   - Update README.md for user-facing changes
   - Update package documentation for new APIs
   - Add examples if adding new features

5. **Commit messages:**
   - Follow Conventional Commits format
   - Include scope when applicable
   - Mark breaking changes with `BREAKING CHANGE:`

### PR Checklist

When creating a pull request, ensure:

*   ✅ All tests pass (`make test`)
*   ✅ Tests pass with race detection (`make test-race`)
*   ✅ Code passes linting (`make lint`)
*   ✅ Code is properly formatted (`make fmt-check`)
*   ✅ No security issues (`make security`)
*   ✅ Commit messages follow Conventional Commits format
*   ✅ Documentation is updated
*   ✅ Appropriate metrics and logging added to new components
*   ✅ Integration tests added for complex features

### CI/CD Pipeline

When you create a PR, the following checks run automatically:

1. **Build**: Builds on multiple Go versions (1.22, 1.23, 1.24)
2. **Format Check**: Verifies code formatting
3. **Lint**: Runs golangci-lint
4. **Vet**: Runs go vet
5. **Tests**: Runs tests on multiple Go versions
6. **Race Detection**: Runs tests with race detector
7. **Integration Tests**: Runs integration test suite
8. **Coverage**: Generates and reports code coverage
9. **Security**: Runs security scans (gosec, govulncheck)

All checks must pass before merging.

## Release Process (Maintainers)

This project uses [release-please](https://github.com/googleapis/release-please) for automated releases.

### Automatic Releases

1. Commits following Conventional Commits are automatically processed
2. Release-please creates a PR with version bump and CHANGELOG updates
3. When the PR is merged, the release is automatically created

### Manual Release Process

If needed, maintainers can trigger releases manually:

1. **Tag the release:**
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. **Release workflow:**
   - The `.github/workflows/release.yml` workflow runs automatically on tag push
   - It uses goreleaser to create release artifacts
   - GitHub release is created with artifacts and changelog

### Version Management

*   **Major** (1.0.0): Breaking changes
*   **Minor** (0.1.0): New features (backward compatible)
*   **Patch** (0.0.1): Bug fixes (backward compatible)

## Troubleshooting

### Common Issues

**Pre-commit hooks failing:**
```bash
# Run hooks manually to see detailed errors
pre-commit run --all-files

# Skip hooks for a commit (not recommended)
git commit --no-verify
```

**Linting errors:**
```bash
# Auto-fix some issues
golangci-lint run --fix ./...

# Check specific package
make lint
```

**Test failures:**
```bash
# Run tests with verbose output
go test -v ./pkg/your-package

# Run specific test
go test -v -run TestYourTest ./pkg/your-package
```

**Go version mismatch:**
```bash
# Check required version
make check-go-version

# Update Go version
# See: https://go.dev/doc/install
```

Thank you for contributing!

