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

## Pull Requests

*   Ensure your branch is up-to-date with the `main` branch before submitting a pull request.
*   Ensure all tests pass (`go test ./...`).
*   Ensure your commit messages follow the Conventional Commits format.

Thank you for contributing!

