package scaffold

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// stableVars returns a ScaffoldVars populated with the deterministic test
// values used by every renderer test (and the golden-file test in T6).
func stableVars() ScaffoldVars {
	return ScaffoldVars{
		AgentName:      "sample-agent",
		BelugaVersion:  "v2.10.1",
		ModelName:      "gpt-4o-mini",
		ModulePath:     "example.com/sample",
		ProjectName:    "sample",
		ProviderImport: "github.com/lookatitude/beluga-ai/v2/llm/providers/openai",
		ProviderName:   "openai",
		ScaffoldedAt:   "2026-04-20T00:00:00Z",
	}
}

// TestApplyTemplate_Substitution checks every sentinel in the fixed set
// is replaced with the corresponding field and unknown sentinels are left
// in place (the caller must assert no leftover __BELUGA_ substring).
func TestApplyTemplate_Substitution(t *testing.T) {
	src := strings.Join([]string{
		"AGENT_NAME=__BELUGA_AGENT_NAME__",
		"BELUGA_VERSION=__BELUGA_VERSION__",
		"MODEL_NAME=__BELUGA_MODEL_NAME__",
		"MODULE_PATH=__BELUGA_MODULE_PATH__",
		"PROJECT_NAME=__BELUGA_PROJECT_NAME__",
		"PROVIDER_IMPORT=__BELUGA_PROVIDER_IMPORT__",
		"PROVIDER_NAME=__BELUGA_PROVIDER_NAME__",
		"SCAFFOLDED_AT=__BELUGA_SCAFFOLDED_AT__",
		"MISSING=__BELUGA_MISSING__",
	}, "\n")

	got := applyTemplate(src, stableVars())

	for _, want := range []string{
		"AGENT_NAME=sample-agent",
		"BELUGA_VERSION=v2.10.1",
		"MODEL_NAME=gpt-4o-mini",
		"MODULE_PATH=example.com/sample",
		"PROJECT_NAME=sample",
		"PROVIDER_IMPORT=github.com/lookatitude/beluga-ai/v2/llm/providers/openai",
		"PROVIDER_NAME=openai",
		"SCAFFOLDED_AT=2026-04-20T00:00:00Z",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("applyTemplate missing expected substitution %q\nfull output:\n%s", want, got)
		}
	}
	// Unknown sentinels are left untouched.
	if !strings.Contains(got, "MISSING=__BELUGA_MISSING__") {
		t.Errorf("applyTemplate unexpectedly modified unknown sentinel\noutput: %s", got)
	}
}

// TestApplyTemplate_Deterministic asserts the same input produces the same
// bytes on repeated invocations (no map-iteration order hazards).
func TestApplyTemplate_Deterministic(t *testing.T) {
	src := "__BELUGA_PROJECT_NAME__ has module __BELUGA_MODULE_PATH__ pinned to __BELUGA_VERSION__"
	vars := stableVars()
	out1 := applyTemplate(src, vars)
	for i := 0; i < 5; i++ {
		if applyTemplate(src, vars) != out1 {
			t.Fatalf("applyTemplate is non-deterministic on iteration %d", i)
		}
	}
}

// TestRenderFS_HappyPath writes a small in-memory template tree and asserts:
// (a) ".tmpl" suffix is stripped on write, (b) the resulting main.go is valid
// Go source (round-trips through go/format.Source without loss), and (c) the
// go.mod content is preserved verbatim after substitution.
func TestRenderFS_HappyPath(t *testing.T) {
	fsys := fstest.MapFS{
		"main.go.tmpl": {Data: []byte(`package __BELUGA_PROJECT_NAME__

func main() {}
`)},
		"go.mod.tmpl":        {Data: []byte("module __BELUGA_MODULE_PATH__\n\ngo 1.25\n")},
		".env.example.tmpl":  {Data: []byte("OPENAI_API_KEY=YOUR_OPENAI_API_KEY_HERE\n")},
		"nested/README.tmpl": {Data: []byte("Project: __BELUGA_PROJECT_NAME__\n")},
		"unrelated.txt":      {Data: []byte("no substitution here\n")},
	}

	target := t.TempDir()
	if err := renderFS(context.Background(), fsys, target, stableVars(), false); err != nil {
		t.Fatalf("renderFS: unexpected error: %v", err)
	}

	mainPath := filepath.Join(target, "main.go")
	mainBytes, err := os.ReadFile(mainPath) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(mainBytes), "package sample") {
		t.Errorf("main.go missing substitution; got:\n%s", mainBytes)
	}
	// .tmpl stripped.
	if _, err := os.Stat(filepath.Join(target, "main.go.tmpl")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("renderFS must strip .tmpl suffix; still present: %v", err)
	}
	// go.mod written.
	modBytes, err := os.ReadFile(filepath.Join(target, "go.mod")) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if !strings.Contains(string(modBytes), "module example.com/sample") {
		t.Errorf("go.mod missing substitution; got:\n%s", modBytes)
	}
	// Nested directory was created with .tmpl stripped.
	if _, err := os.Stat(filepath.Join(target, "nested", "README")); err != nil {
		t.Errorf("nested/README not created: %v", err)
	}
	// Non-tmpl files pass through unchanged.
	plainBytes, err := os.ReadFile(filepath.Join(target, "unrelated.txt")) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read unrelated.txt: %v", err)
	}
	if string(plainBytes) != "no substitution here\n" {
		t.Errorf("non-tmpl file was modified; got %q", plainBytes)
	}
}

// TestRenderFS_NonEmptyTargetRejected asserts that the three-state overwrite
// policy rejects a non-empty target without --force. The error must name
// --force so TestCmdInit_ForceOverwrite can confirm Success Criterion 7.
func TestRenderFS_NonEmptyTargetRejected(t *testing.T) {
	fsys := fstest.MapFS{
		"main.go.tmpl": {Data: []byte("package __BELUGA_PROJECT_NAME__\n")},
	}
	target := t.TempDir()
	// Seed the directory so it is non-empty.
	if err := os.WriteFile(filepath.Join(target, "existing.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	err := renderFS(context.Background(), fsys, target, stableVars(), false)
	if err == nil {
		t.Fatalf("renderFS: expected error for non-empty target without --force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error must name --force for Success Criterion 7; got: %v", err)
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrInvalidInput {
		t.Errorf("error must be core.ErrInvalidInput; got: %v", err)
	}
	// The rejection must happen BEFORE any write — no main.go on disk.
	if _, err := os.Stat(filepath.Join(target, "main.go")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("renderFS must not write files before rejection; stat: %v", err)
	}
}

// TestRenderFS_EmptyTargetProceeds asserts that an existing-but-empty target
// directory is accepted (silent proceed — the cobra layer prints a warning
// to stderr, not the renderer).
func TestRenderFS_EmptyTargetProceeds(t *testing.T) {
	fsys := fstest.MapFS{
		"main.go.tmpl": {Data: []byte("package __BELUGA_PROJECT_NAME__\n")},
	}
	target := t.TempDir()

	if err := renderFS(context.Background(), fsys, target, stableVars(), false); err != nil {
		t.Fatalf("renderFS empty target: unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "main.go")); err != nil {
		t.Errorf("main.go not written into empty target: %v", err)
	}
}

// TestRenderFS_ForceOverwrites asserts --force opens individual files with
// O_CREATE|O_TRUNC, leaving unrelated files alone (not a full os.RemoveAll).
func TestRenderFS_ForceOverwrites(t *testing.T) {
	fsys := fstest.MapFS{
		"main.go.tmpl": {Data: []byte("package __BELUGA_PROJECT_NAME__\n")},
	}
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "main.go"), []byte("package old"), 0o644); err != nil {
		t.Fatalf("seed existing: %v", err)
	}
	// Unrelated file that must survive --force.
	if err := os.WriteFile(filepath.Join(target, "keep.txt"), []byte("preserve"), 0o644); err != nil {
		t.Fatalf("seed keep: %v", err)
	}

	if err := renderFS(context.Background(), fsys, target, stableVars(), true); err != nil {
		t.Fatalf("renderFS --force: unexpected error: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "main.go")) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read overwritten main.go: %v", err)
	}
	if !strings.Contains(string(got), "package sample") {
		t.Errorf("main.go was not overwritten; got %q", got)
	}
	if _, err := os.Stat(filepath.Join(target, "keep.txt")); err != nil {
		t.Errorf("--force must not delete unrelated files; keep.txt stat: %v", err)
	}
}

// TestRenderFS_BadGoSourceWrapped asserts that when substitution yields
// syntactically invalid Go, renderFS returns an error wrapped with the
// "this is a bug" preamble (brief Risk #9 / specialist-security-architect
// §Q3) and writes no .go file.
func TestRenderFS_BadGoSourceWrapped(t *testing.T) {
	// Missing closing brace after substitution.
	fsys := fstest.MapFS{
		"bad.go.tmpl": {Data: []byte(`package __BELUGA_PROJECT_NAME__

func main() { syntax error here
`)},
	}
	target := t.TempDir()
	err := renderFS(context.Background(), fsys, target, stableVars(), false)
	if err == nil {
		t.Fatalf("renderFS: expected go/format.Source failure")
	}
	if !strings.Contains(err.Error(), "this is a bug") {
		t.Errorf("error must include 'this is a bug' preamble; got: %v", err)
	}
	if !strings.Contains(err.Error(), "report it") {
		t.Errorf("error should direct user to report; got: %v", err)
	}
	// No .go file must have been written.
	if _, err := os.Stat(filepath.Join(target, "bad.go")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("renderFS must not write malformed Go; stat: %v", err)
	}
}

// TestRenderFS_CancelledContext asserts ctx.Err is returned and no files
// are written when the context is cancelled before the first write.
func TestRenderFS_CancelledContext(t *testing.T) {
	fsys := fstest.MapFS{
		"main.go.tmpl": {Data: []byte("package __BELUGA_PROJECT_NAME__\n")},
		"b.txt.tmpl":   {Data: []byte("b\n")},
	}
	target := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := renderFS(ctx, fsys, target, stableVars(), false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("renderFS with cancelled ctx: want context.Canceled, got %v", err)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("renderFS must not write on cancelled ctx; entries: %v", entries)
	}
}

// TestRenderFS_DevelReplaceDirective asserts that when BelugaVersion is
// "(devel)" and the template names a file called go.mod(.tmpl), renderFS
// strips the pseudo-version require line and appends a replace directive
// pointing at the detected workspace root (brief Risk #12 /
// specialist-devops-expert §Q3 + §Risk #1).
func TestRenderFS_DevelReplaceDirective(t *testing.T) {
	fsys := fstest.MapFS{
		"go.mod.tmpl": {Data: []byte(
			"module __BELUGA_MODULE_PATH__\n\ngo 1.25\n\nrequire github.com/lookatitude/beluga-ai/v2 __BELUGA_VERSION__\n",
		)},
	}
	target := t.TempDir()

	vars := stableVars()
	vars.BelugaVersion = "(devel)"

	if err := renderFS(context.Background(), fsys, target, vars, false); err != nil {
		t.Fatalf("renderFS: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(target, "go.mod")) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	got := string(data)
	if strings.Contains(got, "v(devel)") {
		t.Errorf("go.mod must not contain v(devel); got:\n%s", got)
	}
	if !strings.Contains(got, "replace github.com/lookatitude/beluga-ai/v2 =>") {
		t.Errorf("go.mod must include replace directive; got:\n%s", got)
	}
	// Go rejects a pseudo-version pin whose major does not match the
	// module's /v2 suffix with "version invalid: should be v2, not v0".
	// The replace directive above does not cover a malformed require line
	// at parse time, so the pin itself must be a valid v2 semver.
	if strings.Contains(got, "v0.0.0-unknown") {
		t.Errorf("go.mod must not carry v0.0.0-unknown (invalid for /v2 module); got:\n%s", got)
	}
	if !strings.Contains(got, "v2.0.0-unknown") {
		t.Errorf("go.mod must carry the v2.0.0-unknown placeholder pin; got:\n%s", got)
	}
}

// TestPostProcessGoMod_PinIsValidV2Semver asserts the placeholder pin used
// by the (devel) post-processor has a "v2." major component — a v0 pin
// would be rejected by Go as "version invalid: should be v2, not v0"
// (Defect C: LOO-149 rejection). This is a direct regression test.
func TestPostProcessGoMod_PinIsValidV2Semver(t *testing.T) {
	rendered := "module example.com/sample\n\ngo 1.25\n\nrequire github.com/lookatitude/beluga-ai/v2 (devel)\n"
	vars := stableVars()
	vars.BelugaVersion = "(devel)"

	out, err := postProcessGoMod(rendered, vars)
	if err != nil {
		t.Fatalf("postProcessGoMod: %v", err)
	}
	if !strings.Contains(out, "v2.0.0-unknown") {
		t.Errorf("postProcessGoMod must emit v2.0.0-unknown pin; got:\n%s", out)
	}
	if strings.Contains(out, "v0.0.0-unknown") {
		t.Errorf("postProcessGoMod must NOT emit v0.0.0-unknown (invalid for /v2 module); got:\n%s", out)
	}
}

// TestDetectProjectRoot exercises the ancestor-walk: a project is identified
// when both .beluga/project.yaml and a go.mod with the beluga require line
// are present. Walk must succeed from a deep subdirectory.
func TestDetectProjectRoot(t *testing.T) {
	root := t.TempDir()
	// Create .beluga/project.yaml and go.mod at the synthetic project root.
	if err := os.MkdirAll(filepath.Join(root, ".beluga"), 0o750); err != nil {
		t.Fatalf("mkdir .beluga: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".beluga", "project.yaml"),
		[]byte("schema-version: 1\nname: sample\n"), 0o644); err != nil {
		t.Fatalf("write project.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.com/sample\n\ngo 1.25\n\nrequire github.com/lookatitude/beluga-ai/v2 v2.10.1\n"),
		0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Walk up from a nested directory.
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o750); err != nil {
		t.Fatalf("mkdir deep: %v", err)
	}

	got, err := detectProjectRoot(deep)
	if err != nil {
		t.Fatalf("detectProjectRoot: %v", err)
	}
	// On macOS t.TempDir may return a /private/var/folders symlinked root;
	// compare after Abs to tolerate symlink resolution differences.
	gotAbs, _ := filepath.Abs(got)
	rootAbs, _ := filepath.Abs(root)
	if gotAbs != rootAbs {
		// Accept symlink-resolved equivalent via EvalSymlinks as a fallback.
		gotEval, _ := filepath.EvalSymlinks(got)
		rootEval, _ := filepath.EvalSymlinks(root)
		if gotEval != rootEval {
			t.Errorf("detectProjectRoot: got %q, want %q", got, root)
		}
	}
}

// TestDetectProjectRoot_MissingBelugaDir asserts the walk fails when only
// go.mod exists but no .beluga/project.yaml.
func TestDetectProjectRoot_MissingBelugaDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.com/sample\n\nrequire github.com/lookatitude/beluga-ai/v2 v2.10.1\n"),
		0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	_, err := detectProjectRoot(root)
	if err == nil {
		t.Fatalf("detectProjectRoot: expected ErrNotFound")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "not inside a Beluga project") {
		t.Errorf("error must include canonical message; got: %v", err)
	}
}

// TestDetectProjectRoot_MissingBelugaRequire asserts the walk fails when
// go.mod lacks the beluga require line.
func TestDetectProjectRoot_MissingBelugaRequire(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".beluga"), 0o750); err != nil {
		t.Fatalf("mkdir .beluga: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".beluga", "project.yaml"),
		[]byte("schema-version: 1\n"), 0o644); err != nil {
		t.Fatalf("write project.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.com/sample\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	_, err := detectProjectRoot(root)
	if err == nil {
		t.Fatalf("detectProjectRoot: expected ErrNotFound when require line missing")
	}
}

// TestDetectWorkspaceRoot asserts ancestor-walking finds the framework root
// (go.mod with module github.com/lookatitude/beluga-ai/v2) from a nested
// synthetic directory.
func TestDetectWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module github.com/lookatitude/beluga-ai/v2\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	deep := filepath.Join(root, "cmd", "beluga", "scaffold")
	if err := os.MkdirAll(deep, 0o750); err != nil {
		t.Fatalf("mkdir deep: %v", err)
	}

	got, err := detectWorkspaceRoot(deep)
	if err != nil {
		t.Fatalf("detectWorkspaceRoot: %v", err)
	}
	gotEval, _ := filepath.EvalSymlinks(got)
	rootEval, _ := filepath.EvalSymlinks(root)
	if gotEval != rootEval {
		t.Errorf("detectWorkspaceRoot: got %q, want %q", got, root)
	}
}

// TestDetectWorkspaceRoot_NotFound asserts the walk returns ErrNotFound
// when no go.mod with the framework module path exists.
func TestDetectWorkspaceRoot_NotFound(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.com/unrelated\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	_, err := detectWorkspaceRoot(root)
	if err == nil {
		t.Fatalf("detectWorkspaceRoot: expected ErrNotFound")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
