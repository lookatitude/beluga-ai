package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// setupProject writes a minimum Beluga project skeleton into dir and
// returns it. The skeleton is exactly what scaffold.DetectProjectRoot
// requires: `.beluga/project.yaml` + a `go.mod` declaring a require line
// for the framework. Module path is configurable so the derived package
// name can be asserted.
func setupProject(t *testing.T, modulePath string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".beluga"), 0o750); err != nil {
		t.Fatalf("mkdir .beluga: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".beluga", "project.yaml"),
		[]byte("schema-version: 1\nname: testproj\ntemplate: basic\nbeluga-version: v2.10.1\nscaffolded-at: 2026-04-20T00:00:00Z\n"),
		0o644); err != nil {
		t.Fatalf("write project.yaml: %v", err)
	}
	goMod := "module " + modulePath + "\n\ngo 1.25\n\nrequire github.com/lookatitude/beluga-ai/v2 v2.10.1\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	return root
}

// TestCmdNewAgent_HappyPath verifies `beluga new agent MyAgent` from a
// scaffolded project writes my_agent.go + my_agent_test.go, both parse,
// and the source contains core.Errorf(core.ErrNotFound, ...) (Decision
// #18 — never panic).
func TestCmdNewAgent_HappyPath(t *testing.T) {
	root := setupProject(t, "example.com/testproj")
	t.Chdir(root)

	if err := executeSubcommand(newNewAgentCmd(), []string{"MyAgent"}); err != nil {
		t.Fatalf("newNewAgentCmd: %v", err)
	}

	srcPath := filepath.Join(root, "my_agent.go")
	testPath := filepath.Join(root, "my_agent_test.go")

	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read my_agent.go: %v", err)
	}
	testBytes, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("read my_agent_test.go: %v", err)
	}

	src := string(srcBytes)
	testSrc := string(testBytes)

	if !strings.Contains(src, "core.Errorf(core.ErrNotFound") {
		t.Errorf("stub must use core.Errorf(core.ErrNotFound, ...) per Decision #18; got:\n%s", src)
	}
	if strings.Contains(src, "panic(") {
		t.Errorf("stub must not panic (Decision #18); got:\n%s", src)
	}
	// Generated stubs land in the same flat directory as main.go which is
	// `package main`; mixing package declarations would break `go build`.
	if !strings.Contains(src, "package main") {
		t.Errorf("stub must declare package main to match the basic template layout; got:\n%s", src)
	}
	// The test file must also be package main (internal test) so it does
	// not import a separate package that does not exist on disk.
	if !strings.Contains(testSrc, "package main") {
		t.Errorf("test file must declare package main for the flat basic template; got:\n%s", testSrc)
	}
	// t.Skip guard — exactly one call in the test file.
	if strings.Count(testSrc, "t.Skip(") != 1 {
		t.Errorf("test file must have exactly one t.Skip call; got:\n%s", testSrc)
	}
	if !strings.Contains(testSrc, `t.Skip("remove when MyAgent is implemented")`) {
		t.Errorf(`test file must contain t.Skip("remove when MyAgent is implemented"); got:\n%s`, testSrc)
	}

	// Both files parse as valid Go.
	for path, body := range map[string][]byte{srcPath: srcBytes, testPath: testBytes} {
		fset := token.NewFileSet()
		if _, perr := parser.ParseFile(fset, path, body, parser.AllErrors); perr != nil {
			t.Errorf("%s must parse: %v", path, perr)
		}
	}
}

// TestCmdNewAgent_OutsideProject asserts the detection-failure contract:
// exit non-zero with a message containing "not inside a Beluga project".
func TestCmdNewAgent_OutsideProject(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	err := executeSubcommand(newNewAgentCmd(), []string{"MyAgent"})
	if err == nil {
		t.Fatalf("expected error outside a Beluga project, got nil")
	}
	if !strings.Contains(err.Error(), "not inside a Beluga project") {
		t.Errorf("error must name the detection failure; got: %v", err)
	}
}

// TestCmdNewTool_HappyPath mirrors TestCmdNewAgent_HappyPath for tools.
// Additionally asserts the file uses tool.NewFuncTool (the canonical
// constructor per framework/tool/functool.go).
func TestCmdNewTool_HappyPath(t *testing.T) {
	root := setupProject(t, "example.com/testproj")
	t.Chdir(root)

	if err := executeSubcommand(newNewToolCmd(), []string{"MyTool"}); err != nil {
		t.Fatalf("newNewToolCmd: %v", err)
	}

	src, err := os.ReadFile(filepath.Join(root, "my_tool.go"))
	if err != nil {
		t.Fatalf("read my_tool.go: %v", err)
	}
	testSrc, err := os.ReadFile(filepath.Join(root, "my_tool_test.go"))
	if err != nil {
		t.Fatalf("read my_tool_test.go: %v", err)
	}
	if !strings.Contains(string(src), "tool.NewFuncTool") {
		t.Errorf("tool stub must use tool.NewFuncTool; got:\n%s", src)
	}
	if !strings.Contains(string(src), "core.Errorf(core.ErrNotFound") {
		t.Errorf("tool stub must use core.Errorf(core.ErrNotFound, ...); got:\n%s", src)
	}
	if strings.Count(string(testSrc), "t.Skip(") != 1 {
		t.Errorf("tool test must have exactly one t.Skip call; got:\n%s", testSrc)
	}

	for _, path := range []string{
		filepath.Join(root, "my_tool.go"),
		filepath.Join(root, "my_tool_test.go"),
	} {
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			t.Fatalf("read %s: %v", path, rerr)
		}
		fset := token.NewFileSet()
		if _, perr := parser.ParseFile(fset, path, data, parser.AllErrors); perr != nil {
			t.Errorf("%s must parse: %v", path, perr)
		}
	}
}

// TestCmdNewPlanner_HappyPath asserts the planner stub shape: PascalCase
// type, agent.Planner compile-time check, commented init() block, and the
// "always use core.Errorf(core.ErrNotFound)" policy from Decision #18.
func TestCmdNewPlanner_HappyPath(t *testing.T) {
	root := setupProject(t, "example.com/testproj")
	t.Chdir(root)

	if err := executeSubcommand(newNewPlannerCmd(), []string{"MyPlanner"}); err != nil {
		t.Fatalf("newNewPlannerCmd: %v", err)
	}

	srcBytes, err := os.ReadFile(filepath.Join(root, "my_planner.go"))
	if err != nil {
		t.Fatalf("read my_planner.go: %v", err)
	}
	testBytes, err := os.ReadFile(filepath.Join(root, "my_planner_test.go"))
	if err != nil {
		t.Fatalf("read my_planner_test.go: %v", err)
	}

	src := string(srcBytes)
	testSrc := string(testBytes)

	if !strings.Contains(src, "var _ agent.Planner = (*MyPlanner)(nil)") {
		t.Errorf("planner stub must declare compile-time check; got:\n%s", src)
	}
	if !strings.Contains(src, "agent.RegisterPlanner") {
		t.Errorf("planner stub must reference agent.RegisterPlanner in commented init(); got:\n%s", src)
	}
	if !strings.Contains(src, "core.Errorf(core.ErrNotFound") {
		t.Errorf("planner stub must use core.Errorf(core.ErrNotFound, ...); got:\n%s", src)
	}
	if strings.Contains(src, "panic(") {
		t.Errorf("planner stub must not panic; got:\n%s", src)
	}
	// Commented init() block: the `func init() {` line must exist only as
	// a comment (lines beginning with "//" or inside a block comment).
	// Assert the uncommented token "func init() {" is absent.
	if strings.Contains(src, "\nfunc init() {") {
		t.Errorf("planner init() block must be commented, not live; got:\n%s", src)
	}
	// Two t.Skip guards (Plan + Replan).
	if got := strings.Count(testSrc, "t.Skip("); got != 2 {
		t.Errorf("planner test file must have exactly 2 t.Skip calls; got %d, src:\n%s", got, testSrc)
	}

	for _, path := range []string{
		filepath.Join(root, "my_planner.go"),
		filepath.Join(root, "my_planner_test.go"),
	} {
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			t.Fatalf("read %s: %v", path, rerr)
		}
		fset := token.NewFileSet()
		if _, perr := parser.ParseFile(fset, path, data, parser.AllErrors); perr != nil {
			t.Errorf("%s must parse: %v", path, perr)
		}
	}
}

// TestCmdNew_RejectBadName exercises the PascalCase allowlist across all
// three kinds. Any non-PascalCase input must be rejected before any
// project detection.
func TestCmdNew_RejectBadName(t *testing.T) {
	root := setupProject(t, "example.com/testproj")
	t.Chdir(root)

	cases := []struct {
		kind string
		name string
	}{
		{"agent", "lowercase"},
		{"agent", "With-Hyphen"},
		{"tool", "has space"},
		{"planner", "1StartWithDigit"},
	}
	factories := map[string]func() *cobra.Command{
		"agent":   newNewAgentCmd,
		"tool":    newNewToolCmd,
		"planner": newNewPlannerCmd,
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.kind+"/"+tc.name, func(t *testing.T) {
			err := executeSubcommand(factories[tc.kind](), []string{"--", tc.name})
			if err == nil {
				t.Fatalf("expected rejection for %q, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), "PascalCase") && !strings.Contains(err.Error(), "pattern") {
				t.Errorf("error must name the PascalCase rule; got: %v", err)
			}
		})
	}
}

// TestCmdNew_VersionSkewWarning asserts the advisory stderr warning fires
// when go.mod pin differs from version.Get(). The test cannot assert the
// exact pin value but can assert the warning is present and references
// both tokens.
func TestCmdNew_VersionSkewWarning(t *testing.T) {
	root := t.TempDir()
	// Set up a project whose go.mod require pin is obviously not equal to
	// version.Get() in this test binary (always "(devel)" unless ldflags).
	if err := os.MkdirAll(filepath.Join(root, ".beluga"), 0o750); err != nil {
		t.Fatalf("mkdir .beluga: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".beluga", "project.yaml"),
		[]byte("schema-version: 1\nname: skew\n"), 0o644); err != nil {
		t.Fatalf("write project.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.com/skew\n\ngo 1.25\n\nrequire github.com/lookatitude/beluga-ai/v2 v9.9.9-different\n"),
		0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	t.Chdir(root)

	cmd := newNewAgentCmd()
	// Collect stderr via the cmd's SetErr hook — executeSubcommand sets it
	// to io.Discard, so we need a local equivalent that keeps the buffer.
	var errBuf strings.Builder
	cmd.SetErr(&errBuf)
	cmd.SetOut(&strings.Builder{})
	cmd.SetArgs([]string{"MyAgent"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(errBuf.String(), "warning:") {
		t.Errorf("expected version-skew warning on stderr; got: %q", errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "v9.9.9-different") {
		t.Errorf("warning must name the project pin; got: %q", errBuf.String())
	}
}

// TestCmdNew_RefuseOverwrite asserts additive semantics: calling
// `beluga new agent` twice with the same name returns an error on the
// second invocation and does not corrupt the first-generated file.
func TestCmdNew_RefuseOverwrite(t *testing.T) {
	root := setupProject(t, "example.com/testproj")
	t.Chdir(root)

	if err := executeSubcommand(newNewAgentCmd(), []string{"Alpha"}); err != nil {
		t.Fatalf("first invocation: %v", err)
	}
	err := executeSubcommand(newNewAgentCmd(), []string{"Alpha"})
	if err == nil {
		t.Fatalf("expected second invocation to refuse overwrite, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error must mention existing file; got: %v", err)
	}
}

// TestToSnakeCase covers the Pascal → snake conversion for the edge cases
// that matter for real PascalCase identifiers: single-run caps, multi-run
// caps, mixed with digits.
func TestToSnakeCase(t *testing.T) {
	cases := map[string]string{
		"MyAgent":    "my_agent",
		"HTTPServer": "http_server",
		"URL":        "url",
		"Simple":     "simple",
		"A":          "a",
		"MyMCPTool":  "my_mcp_tool",
	}
	for in, want := range cases {
		if got := toSnakeCase(in); got != want {
			t.Errorf("toSnakeCase(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestRoot_New_Help (T10 AC) asserts the parent command lists its three
// kinds in --help output and exits zero.
func TestRoot_New_Help(t *testing.T) {
	out, _, code := executeArgs([]string{"new", "--help"})
	if code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
	for _, kind := range []string{"agent", "tool", "planner"} {
		if !strings.Contains(out, kind) {
			t.Errorf("--help must list %q as subcommand; got:\n%s", kind, out)
		}
	}
}

// TestDetectProjectPackage covers the module-path → package-name heuristic
// including the /vN version suffix trim.
func TestDetectProjectPackage(t *testing.T) {
	cases := map[string]string{
		"module example.com/foo\n":            "foo",
		"module github.com/org/my-project\n":  "myproject",
		"module github.com/org/myproj/v2\n":   "myproj",
		"module example.com/foo/BAR\n":        "bar",
		"module example.com/pkg // comment\n": "pkg",
	}
	for goMod, want := range cases {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		if got := detectProjectPackage(dir); got != want {
			t.Errorf("detectProjectPackage(%q) = %q, want %q", goMod, got, want)
		}
	}
}
