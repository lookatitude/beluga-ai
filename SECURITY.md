# Security Policy

## Supported versions

Security fixes are released for the latest minor version of the `v2` module (`github.com/lookatitude/beluga-ai/v2`). Older minor versions and the `v1` module do not receive fixes; upgrade to the latest `v2` release.

| Version | Supported |
|---|---|
| latest `v2.x` minor | Yes |
| older `v2` minors | No |
| `v1` | No |

## Reporting a vulnerability

Email **security@lookatitude.com**. Do not open public issues, discussions or pull requests for security vulnerabilities.

Please include:

- the affected package(s) and version or commit,
- a description of the issue and its impact,
- steps or a minimal program to reproduce it.

We will acknowledge your report and agree a disclosure date with you once a fix is ready.

## Scan status

CI runs govulncheck, gosec, Trivy, Gitleaks, CodeQL and a license check. The latest generated summary is in [docs/security-status.md](docs/security-status.md); full results are in the repository's GitHub Security tab.
