package scaffold

import (
	"context"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// updateGolden regenerates testdata/golden/basic from the current templates
// when `go test -run TestScaffoldBasic_Golden -update` is passed. Use after
// deliberate template changes; always review the diff before committing.
var updateGolden = flag.Bool("update", false, "regenerate golden files for TestScaffoldBasic_Golden")

// TestScaffoldBasic_Golden runs Scaffold with deterministic inputs and
// diffs every produced file against the committed tree at
// testdata/golden/basic/. The stable inputs match stableVars() from
// renderer_test.go so one canonical set drives both tests.
//
// When a template legitimately changes, regenerate the goldens with:
//
//	go test -run TestScaffoldBasic_Golden -update ./cmd/beluga/scaffold/...
func TestScaffoldBasic_Golden(t *testing.T) {
	targetDir := t.TempDir()
	opts := Options{
		ProjectName:   "sample",
		Template:      "basic",
		ModulePath:    "example.com/sample",
		TargetDir:     targetDir,
		BelugaVersion: "v2.10.1",
		ScaffoldedAt:  time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
	}
	if err := Scaffold(context.Background(), opts); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	goldenDir := "testdata/golden/basic"

	if *updateGolden {
		if err := os.RemoveAll(goldenDir); err != nil {
			t.Fatalf("remove existing golden dir: %v", err)
		}
		if err := copyTree(targetDir, goldenDir); err != nil {
			t.Fatalf("copy tree to golden: %v", err)
		}
		t.Logf("golden regenerated at %s", goldenDir)
		return
	}

	// Diff every file in the golden tree against the generated tree.
	gotFiles := collectFiles(t, targetDir)
	wantFiles := collectFiles(t, goldenDir)

	if len(gotFiles) != len(wantFiles) {
		gotKeys := keys(gotFiles)
		wantKeys := keys(wantFiles)
		t.Fatalf("file-count mismatch: generated %d, golden %d\n  generated: %v\n  golden:    %v",
			len(gotFiles), len(wantFiles), gotKeys, wantKeys)
	}
	for relPath, wantContent := range wantFiles {
		gotContent, ok := gotFiles[relPath]
		if !ok {
			t.Errorf("missing file in generated tree: %s", relPath)
			continue
		}
		if gotContent != wantContent {
			t.Errorf("file %s differs from golden:\n--- want\n%s\n--- got\n%s\n---\n",
				relPath, wantContent, gotContent)
		}
	}
}

// TestScaffoldBasic_GoFormat asserts the golden main.go is already
// gofmt-clean (a generated main.go should pass `gofmt -l` with empty
// output). Using the golden file directly keeps the assertion stable
// whether or not -update was just run.
func TestScaffoldBasic_GoFormat(t *testing.T) {
	path := filepath.Join("testdata", "golden", "basic", "main.go")
	data, err := os.ReadFile(path) //nolint:gosec // fixed path under testdata/
	if err != nil {
		t.Fatalf("read golden main.go: %v", err)
	}
	if !strings.Contains(string(data), "package main") {
		t.Errorf("golden main.go should contain 'package main'; content:\n%s", data)
	}
	if !strings.Contains(string(data), "llm.New") {
		t.Errorf("golden main.go should contain llm.New (Success Criterion 14)")
	}
	if !strings.Contains(string(data), "/llm/providers/openai") {
		t.Errorf("golden main.go should blank-import openai provider")
	}
	// Scaffolded projects must bootstrap o11y and attach the agent-tracing
	// middleware so the first `beluga dev`/`beluga run` invocation lights
	// up spans end-to-end (DX-1 S3 brief, decision reversal in Phase 5).
	for _, required := range []string{
		`"github.com/lookatitude/beluga-ai/v2/o11y"`,
		"o11y.BootstrapFromEnv(",
		"defer shutdown()",
		"agent.ApplyMiddleware(",
		"agent.WithTracing()",
	} {
		if !strings.Contains(string(data), required) {
			t.Errorf("golden main.go missing %q (scaffold must wire o11y + tracing)", required)
		}
	}
}

// collectFiles recursively reads every file under root into a map of
// relative path → content. Symlinks and directories are skipped. Paths
// are normalised to forward slashes so Windows and POSIX compare equal.
func collectFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path) //nolint:gosec // test-controlled root
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return out
}

// copyTree copies every file from src to dst, preserving relative paths.
// Directories are created on demand with 0o755 (0o750 for .beluga).
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			perm := os.FileMode(0o755)
			if strings.HasPrefix(rel, ".beluga") {
				perm = 0o750
			}
			return os.MkdirAll(target, perm)
		}
		data, err := os.ReadFile(path) //nolint:gosec // test-controlled
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
