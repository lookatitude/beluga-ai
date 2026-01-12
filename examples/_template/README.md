# {Example Name}

<!--
Template Guidelines for Example READMEs:
- Write like documentation for a library you're sharing
- Be clear, concise, and assume the reader wants to learn
- Include everything needed to run the example successfully
- Anticipate common issues and address them proactively
-->

## Description

<!--
Clear explanation of what the example demonstrates.
Start with "This example shows you how to..."
Use active voice.
-->

This example shows you how to {what the example demonstrates}. You'll learn:

- {Skill/concept 1}
- {Skill/concept 2}
- {Skill/concept 3}

## Prerequisites

<!--
Specific requirements (Go version, API keys, dependencies).
Explain how to obtain each prerequisite.
-->

Before running this example, you need:

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.24+ | [Install Go](https://go.dev/doc/install) |
| Beluga AI | latest | `go get github.com/lookatitude/beluga-ai` |
| {API key} | - | Get one at [{provider}]({url}) |

### Environment Setup

Set these environment variables:

```bash
export {API_KEY}="your-api-key-here"
# Optional: Enable debug logging
export BELUGA_LOG_LEVEL=debug
```

## Usage

<!--
Step-by-step instructions to run the example.
Include expected output.
Add troubleshooting tips for common issues.
-->

### Running the Example

1. **Clone the repository** (if you haven't already):

```bash
git clone https://github.com/lookatitude/beluga-ai.git
cd beluga-ai/examples/{example-path}
```

2. **Install dependencies**:

```bash
go mod download
```

3. **Run the example**:

```bash
go run {main-file}.go
```

### Expected Output

When successful, you'll see:

```
{Expected console output}
```

### Command-Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `--{flag}` | {Description} | `{default}` |

Example with options:

```bash
go run {main-file}.go --{flag}=value
```

## Code Structure

<!--
Overview of how the code is organized.
Explain the architecture and design decisions.
-->

```
{example-name}/
├── README.md           # This file
├── {main-file}.go      # Main entry point
├── {main-file}_test.go # Tests
└── {optional-file}.go  # {Description}
```

### Key Components

| File | Purpose |
|------|---------|
| `{main-file}.go` | {Description of what this file does} |
| `{other-file}.go` | {Description} |

### Design Decisions

This example demonstrates these Beluga AI patterns:

- **{Pattern 1}**: {Brief explanation of why we use it}
- **{Pattern 2}**: {Explanation}
- **OTEL Instrumentation**: Metrics follow the `beluga.{package}.{metric}` naming convention

## Testing

<!--
Instructions for running tests.
Explain what the tests verify and how to interpret results.
-->

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...
```

### Test Structure

| Test | What it verifies |
|------|-----------------|
| `Test{Name}` | {Description of what this test checks} |
| `Test{Name}_Error` | {Error handling scenario} |
| `Benchmark{Name}` | {Performance benchmark} |

### Expected Test Output

```
=== RUN   Test{Name}
--- PASS: Test{Name} (0.00s)
PASS
coverage: {X}% of statements
```

## Troubleshooting

<!--
Common issues and solutions.
-->

### Common Issues

<details>
<summary>❌ Error: "{common error message}"</summary>

**Cause:** {Why this happens}

**Solution:**
```bash
# {Fix command or steps}
```
</details>

<details>
<summary>❌ API key not found</summary>

**Cause:** The `{API_KEY}` environment variable is not set.

**Solution:**
```bash
export {API_KEY}="your-api-key"
# Then run the example again
```
</details>

## Related Examples

<!--
Link to related examples with context about when to use each.
-->

After completing this example, you might want to explore:

- **[{Related Example 1}](../{path}/README.md)** - {Brief description of what it shows}
- **[{Related Example 2}](../{path}/README.md)** - {Description}

## Learn More

- **[{Guide}](/docs/guides/{guide}.md)** - In-depth guide on this topic
- **[{Cookbook}](/docs/cookbook/{recipe}.md)** - Quick recipes for common tasks
- **[{API Reference}](/docs/api/packages/{package}.md)** - API documentation
