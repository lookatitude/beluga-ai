---
title: Pull Request Process
description: Guidelines for submitting pull requests to Beluga AI
---

This guide describes the process for submitting changes to Beluga AI, from opening an issue to getting your code merged.

## Before You Start

1. **Check existing issues** — Search [GitHub Issues](https://github.com/lookatitude/beluga-ai/issues) to see if someone is already working on what you have in mind.
2. **Create an issue** if one doesn't exist. This helps avoid duplicate work and gives maintainers a chance to provide early feedback.
3. **Discuss design for large changes** — For significant features, architectural changes, or new providers, open an issue or start a [Discussion](https://github.com/lookatitude/beluga-ai/discussions) to align on the approach before writing code.

## Branch Naming Convention

Use a descriptive prefix that matches the type of change:

| Prefix | Use for |
|---|---|
| `feat/` | New features |
| `fix/` | Bug fixes |
| `docs/` | Documentation changes |
| `refactor/` | Code refactoring |
| `test/` | Test additions or fixes |
| `chore/` | Maintenance tasks |

**Examples:**

```
feat/anthropic-streaming
fix/agent-nil-pointer
docs/rag-tutorial
refactor/core-error-types
test/memory-store-coverage
chore/update-golangci-lint
```

## Making Changes

1. **Create a branch** from `main`:

   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feat/my-feature
   ```

2. **Make your changes** following the [Code Style Guide](/contributing/code-style/).

3. **Write tests** for any new or changed behavior. See the [Testing Guide](/contributing/testing/).

4. **Run the full check suite** before pushing:

   ```bash
   make check
   ```

   This runs formatting, linting, and all tests in one command.

5. **Commit with Conventional Commits** format:

   ```bash
   git commit -m "feat(llm): add streaming support for Anthropic"
   ```

## Opening a Pull Request

1. **Push your branch** to your fork:

   ```bash
   git push origin feat/my-feature
   ```

2. **Open a PR** against `main` on the [Beluga AI repository](https://github.com/lookatitude/beluga-ai/pulls).

3. **Fill out the PR template** — provide a clear description of what your change does and why.

4. **Link related issues** — Use keywords like `Closes #123` or `Fixes #456` in the PR description to auto-close issues on merge.

## PR Checklist

Before requesting a review, make sure:

- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Code is formatted (`make fmt`)
- [ ] New or changed behavior has tests
- [ ] Documentation is updated if public API changed
- [ ] Commit messages follow [Conventional Commits](/contributing/code-style/#commit-message-format) format
- [ ] PR description clearly explains the change

:::note
You do **not** need to update the CHANGELOG manually. It is auto-generated from commit messages using git-cliff during the [release process](/contributing/releases/).
:::

## Review Process

1. **Automated checks** run first — CI must pass before a maintainer reviews.
2. **A maintainer will review** your PR, usually within a few business days.
3. **You may receive feedback** — this is normal and part of the collaborative process. Address comments by pushing additional commits.
4. **Once approved**, a maintainer will squash-merge your PR into `main`.

### What Reviewers Look For

- Correctness and test coverage
- Adherence to [Code Style](/contributing/code-style/) and project patterns
- Clear, focused scope — one concern per PR
- Performance implications for hot paths
- Backward compatibility

## CI Checks

The following checks must pass before merge:

| Check | Description |
|---|---|
| **Lint** | `golangci-lint` with project configuration |
| **Build** | `go build ./...` across supported platforms |
| **Unit Tests** | All unit tests pass |
| **Integration Tests** | Integration tests pass with required services |
| **Security Scans** | Dependency vulnerability and code security checks |

If a CI check fails, click the details link to see the logs and fix the issue.

## After Merge

- Your branch is **automatically deleted** after merge.
- Changes will appear in the **next release** — see [Release Process](/contributing/releases/).
- If your change is user-facing, it will be included in the auto-generated changelog.

## Tips for Good PRs

- **Keep changes small and focused.** A PR that does one thing well is easier to review and less likely to introduce bugs.
- **Write a clear description.** Explain *what* changed and *why*. Reviewers shouldn't have to read every line of code to understand the purpose.
- **One concern per PR.** Don't mix a bug fix with a refactor or a new feature with a dependency update.
- **Include before/after examples** when changing behavior — this helps reviewers verify correctness.
- **Respond to feedback promptly.** If you disagree with a suggestion, explain your reasoning — healthy discussion leads to better code.
