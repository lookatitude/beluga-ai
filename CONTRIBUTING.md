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

## Development Guidelines

### Code Quality Standards

*   **Go Best Practices**: Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
*   **Error Handling**: Use explicit error handling with proper context
*   **Concurrency**: Use channels and sync primitives safely
*   **Testing**: Write comprehensive unit and integration tests
*   **Documentation**: Document all exported functions and types

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

*   Ensure your branch is up-to-date with the `main` branch before submitting a pull request.
*   Ensure all tests pass (`go test ./...`).
*   Run `go vet ./...` and fix any issues.
*   Ensure your commit messages follow the Conventional Commits format.
*   Update documentation for any new features or configuration options.
*   Add appropriate metrics and logging to new components.

Thank you for contributing!

