# Contributing to Beluga AI

Thank you for your interest in contributing to Beluga AI! Every contribution, whether it's a bug report, feature request, documentation improvement, or code change, helps make this framework better for everyone.

For comprehensive developer documentation, visit our [Contributing Guide](https://lookatitude.github.io/beluga-ai/contributing/) on the docs website.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to conduct@lookatitude.com.

## Quick Start

### Prerequisites

- **Go 1.23+** (uses `iter.Seq2` for streaming)
- **Git**
- **Make** (optional but recommended)
- **golangci-lint** (for linting)

### Setup

```bash
# Fork and clone the repository
git clone https://github.com/<your-username>/beluga-ai.git
cd beluga-ai

# Build
make build    # or: go build ./...

# Run tests
make test     # or: go test -race ./...

# Run linter
make lint     # or: golangci-lint run
```

### Common Make Targets

| Target             | Description                          |
|--------------------|--------------------------------------|
| `make build`       | Build all packages                   |
| `make test`        | Run unit tests with race detection   |
| `make lint`        | Run go vet and golangci-lint         |
| `make integration-test` | Run integration tests           |
| `make coverage`    | Generate HTML coverage report        |
| `make fuzz`        | Run fuzz tests (30s each)            |
| `make bench`       | Run benchmarks                       |
| `make check`       | Full pre-commit check (lint + test)  |
| `make fmt`         | Format code with gofmt + goimports   |
| `make tidy`        | Run go mod tidy and verify clean     |

## How to Contribute

### Reporting Bugs

Open a [bug report](https://github.com/lookatitude/beluga-ai/issues/new?template=bug_report.yml) with a clear description, steps to reproduce, and expected vs actual behavior.

### Suggesting Features

Open a [feature request](https://github.com/lookatitude/beluga-ai/issues/new?template=feature_request.yml) describing the problem, proposed solution, and use case.

### Submitting Code

1. **Create a branch** from `main` using the naming convention:
   - `feat/description` -- new features
   - `fix/description` -- bug fixes
   - `docs/description` -- documentation
   - `refactor/description` -- code refactoring
   - `test/description` -- test additions/fixes

2. **Make your changes** following our [Code Style Guide](https://lookatitude.github.io/beluga-ai/contributing/code-style/).

3. **Write tests** for any new or changed functionality.

4. **Run the full check** before committing:
   ```bash
   make check
   ```

5. **Commit using Conventional Commits** format:
   ```
   feat(llm): add streaming support for Anthropic provider
   fix(agent): prevent nil pointer in handoff execution
   docs(rag): add hybrid search tutorial
   test(tool): add FuncTool edge case tests
   ```

6. **Open a Pull Request** against `main` and fill out the PR template.

## Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

**Types:** `feat`, `fix`, `docs`, `chore`, `test`, `refactor`, `perf`, `ci`

**Scopes** (optional): `llm`, `agent`, `tool`, `rag`, `voice`, `memory`, `core`, `schema`, `guard`, `protocol`, `cache`, `auth`, `eval`, `config`, `o11y`

**Breaking changes:** Append `!` after the type/scope (e.g. `feat(llm)!: remove deprecated API`) or include a `BREAKING CHANGE` line in the commit body. This signals a major version bump and prevents automatic release — the team must publish a major version manually.

## Releases

### Automatic releases (patch and minor)

When the [Main](.github/workflows/main.yml) pipeline succeeds (CI and security checks pass), a job inspects **all commits since the last tag** using conventional commit prefixes:

| Commits since last tag contain | Bump | Example |
|--------------------------------|------|---------|
| Any `feat:` (no breaking) | **Minor** | `v1.2.3` → `v1.3.0` |
| Only `fix:`, `docs:`, `chore:`, etc. | **Patch** | `v1.2.3` → `v1.2.4` |
| Any breaking change (`feat!:`, `BREAKING CHANGE`) | **Skipped** | No tag created |

When a tag is created it is pushed to the repository, which triggers the [Release](.github/workflows/release.yml) workflow (GoReleaser, changelog, docs rebuild).

### Major releases (manual only)

Major versions are **never created automatically**. When breaking changes land on `main`, the auto-release is skipped. To publish a major release:

1. Go to **Actions → Release → Run workflow**.
2. Enter the desired tag (e.g. `v2.0.0`) and the ref to release from (default `main`).

### Tag push

You can also create and push a version tag directly (e.g. `git tag v1.2.3 && git push origin v1.2.3`). The Release workflow runs on any push to tags matching `v*.*.*`.

See the [releases guide](https://lookatitude.github.io/beluga-ai/contributing/releases/) on the docs site for full details.

## Security Vulnerabilities

If you discover a security vulnerability, **do not** open a public issue. Please report it responsibly by emailing **security@lookatitude.com**.

## License

By contributing to Beluga AI, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
