//go:build integration

// End-to-end integration test for `beluga init` + `beluga new agent|tool|planner`.
// This test does the full round-trip: scaffold a project, replace-directive
// it to point at the in-tree framework, run `go mod tidy`, `go build ./...`,
// `go test ./...`, then run `beluga new <kind>` for each kind and re-run the
// build/test gate. It caught three defects the unit tests missed:
//
//  1. Package-collision between generated main.go (`package main`) and new
//     stubs that declared `package <short-module-name>` in the same dir.
//  2. Malformed test imports pointing at a short package name instead of a
//     module path.
//  3. `v0.0.0-unknown` pseudo-version rejected by Go as invalid for /v2
//     module path.
//
// The test is gated behind `//go:build integration` so it runs in CI but not
// every local `go test ./...` invocation (it shells out to `go build` /
// `go mod tidy` which take tens of seconds).

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// findFrameworkRoot walks up from the test binary's directory looking for a
// go.mod whose module directive is "github.com/lookatitude/beluga-ai/v2".
// The test runs from cmd/beluga/ so the root is two directories up.
func findFrameworkRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		goMod := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(goMod); err == nil {
			if strings.Contains(string(data), "module github.com/lookatitude/beluga-ai/v2") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("framework root not found walking up from %s", filepath.Dir(file))
		}
		dir = parent
	}
}

// runCmd runs a command in the given working directory and fails the test
// with captured stdout/stderr when the command exits non-zero. The
// GOTOOLCHAIN=local envvar mirrors the framework CI so this test behaves
// identically locally and in CI.
func runCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %s %v\ndir: %s\nstdout:\n%s\nstderr:\n%s\nerr: %v",
			name, args, dir, stdout.String(), stderr.String(), err)
	}
}

// TestScaffoldIntegration_EndToEnd is the flagship integration test. It
// reproduces the QA-rejection scenario (`beluga init` then `beluga new <kind>`
// then `go build ./...` + `go test ./...`) and fails loudly if any step
// regresses. The test intentionally builds the CLI binary rather than
// dispatching in-process so the ldflags Version injection path is also
// covered — the (devel)-version branch inside postProcessGoMod fires only
// when version.Get() is actually "(devel)".
func TestScaffoldIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: skipping under -short")
	}

	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go toolchain not on PATH: %v", err)
	}

	frameworkRoot := findFrameworkRoot(t)
	tmp := t.TempDir()
	belugaBin := filepath.Join(tmp, "beluga")

	// Build the CLI with the ldflags used by the QA-repro scenario: the
	// version string is injected as "(devel)" so the replace-directive
	// branch of postProcessGoMod fires. This also guarantees the pin
	// fix (v2.0.0-unknown) is exercised inside go mod tidy.
	buildCmd := exec.Command(goBin, "build",
		"-ldflags", "-X github.com/lookatitude/beluga-ai/v2/cmd/beluga/internal/version.Version=(devel)",
		"-o", belugaBin,
		"./cmd/beluga",
	)
	buildCmd.Dir = frameworkRoot
	buildCmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	var buildOut, buildErr bytes.Buffer
	buildCmd.Stdout = &buildOut
	buildCmd.Stderr = &buildErr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("build beluga CLI: %v\nstdout: %s\nstderr: %s",
			err, buildOut.String(), buildErr.String())
	}

	// Scaffold a project. The CLI discovers the workspace root by
	// ancestor-walking from cwd, so we chdir into the framework root's
	// tmp-adjacent directory — any directory inside the framework tree
	// works. Using `tmp` directly fails to find the workspace, so we
	// scaffold into a subdirectory of frameworkRoot we clean up
	// afterwards.
	scaffoldParent := filepath.Join(frameworkRoot, "tmp-integration-"+filepath.Base(tmp))
	if err := os.MkdirAll(scaffoldParent, 0o750); err != nil {
		t.Fatalf("mkdir scaffold parent: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(scaffoldParent) })

	runCmd(t, scaffoldParent, belugaBin, "init", "testproject")
	projectDir := filepath.Join(scaffoldParent, "testproject")

	// Sanity: generated main.go must exist and declare package main.
	mainGo := filepath.Join(projectDir, "main.go")
	data, err := os.ReadFile(mainGo) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(data), "package main") {
		t.Fatalf("generated main.go must declare package main; got:\n%s", data)
	}

	// go.mod must have the v2.0.0-unknown pin (Defect C) and the replace
	// directive pointing at the framework root.
	goModData, err := os.ReadFile(filepath.Join(projectDir, "go.mod")) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if !strings.Contains(string(goModData), "v2.0.0-unknown") {
		t.Errorf("go.mod must carry v2.0.0-unknown pin (Defect C regression); got:\n%s", goModData)
	}
	if !strings.Contains(string(goModData), "replace github.com/lookatitude/beluga-ai/v2") {
		t.Errorf("go.mod must carry replace directive for (devel) build; got:\n%s", goModData)
	}

	// First build/test gate: stock scaffolded project compiles + tests
	// pass (main.go references the echo tool, the .env.example file
	// sentinel, etc.).
	runCmd(t, projectDir, goBin, "mod", "tidy")
	runCmd(t, projectDir, goBin, "build", "./...")
	runCmd(t, projectDir, goBin, "test", "./...")

	// Second step: scaffold stubs in-tree with `beluga new <kind> <Name>`
	// and re-run the gate. This is the scenario that failed pre-fix:
	// mixing `package main` (main.go) with `package testproject` (new
	// stubs) produced a Go compilation error.
	for _, spec := range []struct {
		kind string
		name string
	}{
		{"agent", "SmokeAgent"},
		{"tool", "SmokeTool"},
		{"planner", "SmokePlanner"},
	} {
		runCmd(t, projectDir, belugaBin, "new", spec.kind, spec.name)
		runCmd(t, projectDir, goBin, "build", "./...")
		runCmd(t, projectDir, goBin, "test", "./...")
	}
}
