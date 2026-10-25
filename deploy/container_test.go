package deploy

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// parseMajorMinor parses "1.26" / "1.26.0" into (1, 26).
func parseMajorMinor(t *testing.T, v string) (int, int) {
	t.Helper()
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		t.Fatalf("version %q has no minor component", v)
	}
	maj, err := strconv.Atoi(parts[0])
	if err != nil {
		t.Fatalf("version %q: %v", v, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("version %q: %v", v, err)
	}
	return maj, minor
}

// Regression (FI-31): the default builder toolchain must be able to build this
// module. Official golang images set GOTOOLCHAIN=local, so an older default
// fails `go mod download` against a newer go directive.
func TestDefaultGoVersion_MatchesGoMod(t *testing.T) {
	f, err := os.Open("../go.mod")
	if err != nil {
		t.Fatalf("open go.mod: %v", err)
	}
	defer func() { _ = f.Close() }()

	var goDirective string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if v, ok := strings.CutPrefix(line, "go "); ok {
			goDirective = strings.TrimSpace(v)
			break
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan go.mod: %v", err)
	}
	if goDirective == "" {
		t.Fatal("go directive not found in go.mod")
	}

	modMaj, modMin := parseMajorMinor(t, goDirective)
	defMaj, defMin := parseMajorMinor(t, DefaultGoVersion)
	if defMaj < modMaj || (defMaj == modMaj && defMin < modMin) {
		t.Errorf("DefaultGoVersion %q is older than go.mod's go directive %q", DefaultGoVersion, goDirective)
	}
	if !goVersionRe.MatchString(DefaultGoVersion) {
		t.Errorf("DefaultGoVersion %q fails validation", DefaultGoVersion)
	}
}

// Regression (FI-31): build path is configurable, defaults to "." (matching
// the `beluga init` scaffold layout), and the old ./cmd/agent layout stays
// reachable.
func TestGenerateDockerfile_BuildPath(t *testing.T) {
	tests := []struct {
		name      string
		buildPath string
		want      string
		wantErr   bool
	}{
		{name: "default is module root", buildPath: "", want: "-o /agent .\n"},
		{name: "explicit dot", buildPath: ".", want: "-o /agent .\n"},
		{name: "legacy cmd/agent layout", buildPath: "./cmd/agent", want: "-o /agent ./cmd/agent\n"},
		{name: "relative without dot-slash is normalised", buildPath: "cmd/agent", want: "-o /agent ./cmd/agent\n"},
		{name: "trailing slash cleaned", buildPath: "./cmd/agent/", want: "-o /agent ./cmd/agent\n"},
		{name: "traversal rejected", buildPath: "../other", wantErr: true},
		{name: "absolute rejected", buildPath: "/src/cmd", wantErr: true},
		{name: "newline injection rejected", buildPath: "./cmd\nRUN evil", wantErr: true},
		{name: "shell meta rejected", buildPath: "./cmd;rm -rf /", wantErr: true},
		{name: "flag-like rejected", buildPath: "-ldflags=x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateDockerfile(DockerfileConfig{AgentConfig: "config/agent.yaml", Port: 8080, BuildPath: tt.buildPath})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got:\n%s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("missing %q in:\n%s", tt.want, got)
			}
		})
	}
}

// Regression (FI-31): `USER nonroot:nonroot` only resolves on distroless
// images; other bases get a numeric non-root UID that works without a passwd
// entry, and the user is configurable.
func TestGenerateDockerfile_User(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		user    string
		want    string
		notWant string
		wantErr bool
	}{
		{name: "distroless default", base: "", want: "USER nonroot:nonroot\n"},
		{name: "distroless explicit", base: "gcr.io/distroless/base-debian12:nonroot", want: "USER nonroot:nonroot\n"},
		{name: "alpine gets numeric uid", base: "alpine:3.20", want: "USER 65532:65532\n", notWant: "nonroot"},
		{name: "scratch gets numeric uid", base: "scratch", want: "USER 65532:65532\n", notWant: "nonroot"},
		{name: "custom user", base: "alpine:3.20", user: "app:app", want: "USER app:app\n"},
		{name: "custom numeric user", base: "ubuntu:24.04", user: "1000", want: "USER 1000\n"},
		{name: "user injection rejected", user: "app\nRUN evil", wantErr: true},
		{name: "user shell meta rejected", user: "app;id", wantErr: true},
		{name: "user empty group rejected", user: "app:", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateDockerfile(DockerfileConfig{BaseImage: tt.base, User: tt.user, AgentConfig: "config/agent.yaml", Port: 8080})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got:\n%s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("missing %q in:\n%s", tt.want, got)
			}
			if tt.notWant != "" && strings.Contains(got, tt.notWant) {
				t.Errorf("unexpected %q in:\n%s", tt.notWant, got)
			}
		})
	}
}
