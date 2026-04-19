// Package version exposes the framework version baked into the beluga CLI.
//
// Precedence: ldflags -X > build info (go install @vX.Y.Z) > "(devel)".
package version

import "runtime/debug"

// Version, Commit, and Date are overridden at link time by goreleaser:
//
//	-ldflags "-X github.com/lookatitude/beluga-ai/v2/cmd/beluga/internal/version.Version=v1.2.3
//	          -X github.com/lookatitude/beluga-ai/v2/cmd/beluga/internal/version.Commit=<sha>
//	          -X github.com/lookatitude/beluga-ai/v2/cmd/beluga/internal/version.Date=<iso8601>"
//
// When Version is not set (local `go build`, ad-hoc compiles), Get falls back
// to the module version from runtime/debug.ReadBuildInfo, which is populated
// by `go install github.com/lookatitude/beluga-ai/v2/cmd/beluga@vX.Y.Z`. When
// neither source provides a value (ephemeral `go run` or a cold compile in
// the framework repo), Get returns "(devel)".
var (
	Version = ""
	Commit  = ""
	Date    = ""
)

// Get returns the resolved version string. See package doc for precedence.
func Get() string {
	if Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "(devel)"
}
