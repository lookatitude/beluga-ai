// Command beluga provides CLI tools for managing Beluga AI projects.
//
// It is the primary developer-facing entry point in Layer 7 (Application)
// of the Beluga architecture, offering subcommands for project scaffolding,
// version introspection, provider discovery, local development, testing,
// and deployment.
//
// # Subcommands
//
//   - version    Print framework version, Go runtime, and provider counts.
//   - providers  List providers compiled into this binary (supports --output json).
//   - init       Scaffold a new Beluga agent project.
//   - dev        Start the development server (playground, hot reload).
//   - test       Run agent tests via `go test`.
//   - deploy     Generate deployment artifacts (Dockerfile, compose, k8s).
//
// # Installation
//
//	go install github.com/lookatitude/beluga-ai/v2/cmd/beluga@latest
//
// Release binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64,
// and windows/amd64 are attached to each GitHub release with a sha256
// checksums.txt. See .goreleaser.yml for the build matrix.
//
// # Bundled providers
//
// The binary ships with a curated, CGo-free provider set (see
// cmd/beluga/providers/providers.go). Run `beluga providers` to list them.
// To build a binary with a different set, create your own main package
// and blank-import the provider packages you need.
//
// # Version resolution
//
// `beluga version` resolves the framework version string via:
//
//	ldflags -X .../internal/version.Version > runtime/debug.ReadBuildInfo > "(devel)"
//
// goreleaser injects the Version, Commit, and Date at link time on tag push.
// `go install ...@vX.Y.Z` populates them via the build info fallback.
package main
