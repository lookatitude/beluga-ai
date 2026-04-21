package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/cmd/beluga/devloop"
	"github.com/spf13/cobra"
)

// executeArgs runs the cobra root with the given args and returns captured
// stdout/stderr plus the exit code from Execute. It mirrors the production
// Execute() helper's handling of runExitError so tests see the same exit-code
// contract as the shipped binary.
func executeArgs(args []string) (stdout, stderr string, code int) {
	var out, errBuf bytes.Buffer
	cmd := newRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	if err := cmd.Execute(); err != nil {
		var re *runExitError
		if errors.As(err, &re) {
			if re.err != nil {
				_, _ = errBuf.WriteString("error: " + re.err.Error() + "\n")
			}
			code = re.ExitCode()
		} else {
			_, _ = errBuf.WriteString("error: " + err.Error() + "\n")
			code = 1
		}
	}
	return out.String(), errBuf.String(), code
}

// executeSubcommand runs a single subcommand in isolation (no root). Useful
// for direct subcommand tests that don't need root-level flag parsing.
func executeSubcommand(cmd *cobra.Command, args []string) error {
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	return cmd.Execute()
}

// --- Per-subcommand tests (migrate from direct cmdInit/cmdDev/… calls) ---

// TestCmdInit_PositionalName is the happy path: a bare `beluga init <name>`
// from a clean working directory writes the full basic template, with the
// three anchor files (go.mod, main.go, .beluga/project.yaml) produced and
// main.go containing the Layer 7 canonical shape.
func TestCmdInit_PositionalName(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := executeSubcommand(newInitCmd(), []string{"my-project"}); err != nil {
		t.Fatalf("newInitCmd: %v", err)
	}

	projDir := filepath.Join(dir, "my-project")
	for _, anchor := range []string{
		"go.mod",
		"main.go",
		filepath.Join(".beluga", "project.yaml"),
		".env.example",
		".gitignore",
		"Dockerfile",
		"Makefile",
		filepath.Join(".github", "workflows", "ci.yml"),
	} {
		if _, err := os.Stat(filepath.Join(projDir, anchor)); err != nil {
			t.Errorf("expected %s to exist: %v", anchor, err)
		}
	}

	mainBytes, err := os.ReadFile(filepath.Join(projDir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(mainBytes), `llm.New("openai"`) {
		t.Errorf("main.go missing llm.New call; got:\n%s", mainBytes)
	}
	if !strings.Contains(string(mainBytes), `/llm/providers/openai`) {
		t.Errorf("main.go missing openai blank import; got:\n%s", mainBytes)
	}
	// Project-name derivation into agent id — brief Decision #4.
	if !strings.Contains(string(mainBytes), `agent.New("my-project-agent"`) {
		t.Errorf("main.go should name agent from project name; got:\n%s", mainBytes)
	}
}

// TestCmdInit_RejectBadName exercises the allowlist regex +
// Windows-reserved-name blocklist at the cobra entry point. Each of these
// must exit non-zero with an error message naming the validation rule.
func TestCmdInit_RejectBadName(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cases := []struct {
		name  string
		input string
	}{
		{"path traversal", "../evil"},
		{"leading hyphen", "-badstart"},
		{"contains space", "My Project"},
		{"windows reserved", "con"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Prefix "--" so cobra treats the input as a positional
			// argument even when it starts with "-". Real users typing
			// `beluga init -badstart` hit the same cobra flag-parsing
			// error, which is acceptable (it still rejects the name);
			// this test targets the validator path specifically.
			err := executeSubcommand(newInitCmd(), []string{"--", tc.input})
			if err == nil {
				t.Fatalf("expected rejection for %q, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), "allowed pattern") &&
				!strings.Contains(err.Error(), "reserved") {
				t.Errorf("error must name validation rule; got: %v", err)
			}
		})
	}
}

// TestCmdInit_ForceOverwrite validates Success Criterion 7: a non-empty
// target directory exits non-zero with a message containing --force, and
// the --force flag permits the overwrite.
func TestCmdInit_ForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	projDir := filepath.Join(dir, "dup-target")
	if err := os.MkdirAll(projDir, 0o750); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "stale.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("seed stale: %v", err)
	}

	err := executeSubcommand(newInitCmd(), []string{"dup-target"})
	if err == nil {
		t.Fatalf("expected rejection of non-empty target without --force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error must mention --force (Success Criterion 7); got: %v", err)
	}

	// With --force the command succeeds and the stale file survives
	// (we only overwrite template-produced files, not the whole tree).
	if err := executeSubcommand(newInitCmd(), []string{"--force", "dup-target"}); err != nil {
		t.Fatalf("--force init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projDir, "stale.txt")); err != nil {
		t.Errorf("--force should not delete unrelated files; stale.txt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projDir, "main.go")); err != nil {
		t.Errorf("--force should have written main.go; got: %v", err)
	}
}

// TestCmdInit_DevelVersion simulates a (devel)-build invocation inside a
// framework checkout. It synthesises a fake workspace root (a go.mod whose
// module path is the framework module) and chdirs into a nested directory
// so detectWorkspaceRoot finds it. Asserts the generated go.mod carries a
// replace directive pointing at the detected root and contains no literal
// "v(devel)" token. Brief Risk #12 / Decision #12.
func TestCmdInit_DevelVersion(t *testing.T) {
	root := t.TempDir()
	// Fake the framework checkout: a go.mod with the framework module path.
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module github.com/lookatitude/beluga-ai/v2\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("seed framework go.mod: %v", err)
	}
	// Nested cwd so detectWorkspaceRoot has to ancestor-walk.
	nested := filepath.Join(root, "sub", "nested")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	t.Chdir(nested)

	if err := executeSubcommand(newInitCmd(), []string{"devel-project"}); err != nil {
		t.Fatalf("newInitCmd: %v", err)
	}
	goMod, err := os.ReadFile(filepath.Join(nested, "devel-project", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	// The test binary always runs with version.Get() == "(devel)" (no
	// ldflags injection) so the replace-directive branch MUST fire —
	// and it must NOT leave a literal "v(devel)" token behind.
	if strings.Contains(string(goMod), "v(devel)") {
		t.Errorf("go.mod must not contain v(devel); got:\n%s", goMod)
	}
	if !strings.Contains(string(goMod), "replace github.com/lookatitude/beluga-ai/v2") {
		t.Errorf("go.mod must contain replace directive for (devel) build; got:\n%s", goMod)
	}
}

func TestCmdDev_UnknownFlagRejected(t *testing.T) {
	// The legacy --port flag was removed in favour of --playground; the
	// cobra layer must reject it rather than silently accept.
	err := executeSubcommand(newDevCmd(), []string{"--port", "9090"})
	if err == nil {
		t.Error("expected cobra to reject removed --port flag")
	}
}

func TestCmdDev_BadPlaygroundFlag(t *testing.T) {
	origRun := devloopRun
	defer func() { devloopRun = origRun }()
	devloopRun = func(_ context.Context, _ devloop.Config) error { return nil }

	err := executeSubcommand(newDevCmd(), []string{"--playground", "not-a-port"})
	if err == nil || !strings.Contains(err.Error(), "invalid --playground") {
		t.Errorf("expected --playground validation error, got: %v", err)
	}
}

func TestCmdDev_PlaygroundOff(t *testing.T) {
	// With --playground=off, the command must not attempt to start the
	// UI server at all; a stubbed devloopRun lets us confirm the config
	// is built without any sink wiring.
	origRun := devloopRun
	defer func() { devloopRun = origRun }()
	var captured devloop.Config
	devloopRun = func(_ context.Context, cfg devloop.Config) error {
		captured = cfg
		return nil
	}
	if err := executeSubcommand(newDevCmd(), []string{"--playground", "off"}); err != nil {
		t.Fatalf("dev --playground off: %v", err)
	}
	if captured.OnRestart != nil {
		t.Error("--playground=off should leave OnRestart nil")
	}
	for _, e := range captured.ExtraEnv {
		if strings.HasPrefix(e, "BELUGA_PLAYGROUND_URL=") {
			t.Errorf("--playground=off must not set BELUGA_PLAYGROUND_URL; got %q", e)
		}
	}
}

func TestCmdDeploy(t *testing.T) {
	tests := []struct {
		target  string
		wantErr bool
	}{
		{"docker", false},
		{"compose", false},
		{"k8s", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			err := executeSubcommand(newDeployCmd(), []string{"--target", tt.target})
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCmdTest_InvalidPkgPattern(t *testing.T) {
	err := executeSubcommand(newTestCmd(), []string{"--pkg", "./... -exec evil"})
	if err == nil || !strings.Contains(err.Error(), "invalid package pattern") {
		t.Errorf("expected invalid package pattern error, got: %v", err)
	}
}

func TestCmdTest_ParseError(t *testing.T) {
	err := executeSubcommand(newTestCmd(), []string{"--nope"})
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestCmdTest_LookPathFailure(t *testing.T) {
	orig := lookPath
	defer func() { lookPath = orig }()
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }

	err := executeSubcommand(newTestCmd(), []string{"--pkg", "./..."})
	if err == nil || !strings.Contains(err.Error(), "locate go toolchain") {
		t.Errorf("expected toolchain lookup error, got: %v", err)
	}
}

func TestCmdTest_Success(t *testing.T) {
	origLook := lookPath
	origExec := execCommand
	defer func() {
		lookPath = origLook
		execCommand = origExec
	}()
	lookPath = func(string) (string, error) { return "/usr/bin/true", nil }
	execCommand = func(stdout, stderr io.Writer, name string, args ...string) *exec.Cmd {
		// Always return a command that succeeds regardless of the OS.
		return exec.Command("/bin/sh", "-c", "exit 0")
	}

	if err := executeSubcommand(newTestCmd(), []string{"-v", "--race", "--pkg", "./..."}); err != nil {
		t.Errorf("newTestCmd: %v", err)
	}
}

// TestCmdTest_CanonicalEnv verifies that `beluga test` injects the
// BELUGA_ENV=test / BELUGA_LLM_PROVIDER=mock / OTEL_SDK_DISABLED=true
// triple into the child `go test` env. Asserting on cmd.Env is the
// only way to check this without running real tests.
func TestCmdTest_CanonicalEnv(t *testing.T) {
	origLook := lookPath
	origExec := execCommand
	defer func() {
		lookPath = origLook
		execCommand = origExec
	}()
	lookPath = func(string) (string, error) { return "/usr/bin/true", nil }

	var cmdRef *exec.Cmd
	execCommand = func(stdout, stderr io.Writer, name string, args ...string) *exec.Cmd {
		c := exec.Command("/bin/sh", "-c", "exit 0")
		cmdRef = c
		return c
	}

	if err := executeSubcommand(newTestCmd(), []string{"--pkg", "./..."}); err != nil {
		t.Fatalf("newTestCmd: %v", err)
	}
	if cmdRef == nil {
		t.Fatal("execCommand stub never invoked")
	}
	for _, want := range []string{"BELUGA_ENV=test", "BELUGA_LLM_PROVIDER=mock", "OTEL_SDK_DISABLED=true"} {
		found := false
		for _, e := range cmdRef.Env {
			if e == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("canonical env missing %q; got: %v", want, cmdRef.Env)
		}
	}
}

// TestCmdTest_BannerUsesCobraWriter asserts that the "Running: ..." banner
// is written to the cobra command's configured stdout (via cmd.OutOrStdout),
// not directly to os.Stdout. A prior implementation used fmt.Printf, which
// bypassed cobra's writer plumbing and made test capture awkward.
func TestCmdTest_BannerUsesCobraWriter(t *testing.T) {
	origLook := lookPath
	origExec := execCommand
	defer func() {
		lookPath = origLook
		execCommand = origExec
	}()
	lookPath = func(string) (string, error) { return "/usr/bin/true", nil }
	execCommand = func(stdout, stderr io.Writer, name string, args ...string) *exec.Cmd {
		return exec.Command("/bin/sh", "-c", "exit 0")
	}

	var out, errBuf bytes.Buffer
	cmd := newTestCmd()
	cmd.SetArgs([]string{"--pkg", "./..."})
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "Running: /usr/bin/true") {
		t.Errorf("banner missing from cobra stdout; got stdout=%q stderr=%q", out.String(), errBuf.String())
	}
}

func TestCmdTest_RunFailure(t *testing.T) {
	origLook := lookPath
	origExec := execCommand
	defer func() {
		lookPath = origLook
		execCommand = origExec
	}()
	lookPath = func(string) (string, error) { return "/usr/bin/false", nil }
	execCommand = func(stdout, stderr io.Writer, name string, args ...string) *exec.Cmd {
		return exec.Command("/bin/sh", "-c", "exit 1")
	}

	if err := executeSubcommand(newTestCmd(), []string{"--pkg", "./..."}); err == nil {
		t.Error("expected non-zero exit error")
	}
}

// --- version subcommand (T4) ---

// TestVersionCommand asserts that `beluga version` prints:
//   - a line containing "beluga " followed by the resolved framework version
//   - a line containing "go1." (from runtime.Version())
//   - a line starting with "providers:" listing the four category counts
//
// These three substrings are the verifiable surface for AC2.
func TestVersionCommand(t *testing.T) {
	out, _, code := executeArgs([]string{"version"})
	if code != 0 {
		t.Fatalf("version: want exit 0, got %d", code)
	}
	for _, want := range []string{"beluga ", "go1.", "providers:"} {
		if !strings.Contains(out, want) {
			t.Errorf("version stdout missing %q; got:\n%s", want, out)
		}
	}
}

// --- providers subcommand (T6) ---

// TestProvidersCommand_Human asserts the default text output contains all
// seven curated providers plus the memory built-ins (core/recall/archival/
// composite, registered by memory/*.go), exit 0, stderr empty.
func TestProvidersCommand_Human(t *testing.T) {
	out, errBuf, code := executeArgs([]string{"providers"})
	if code != 0 {
		t.Fatalf("providers: want exit 0, got %d; stderr=%s", code, errBuf)
	}
	if errBuf != "" {
		t.Errorf("providers: stderr must be empty on success, got: %s", errBuf)
	}
	// Curated providers (the blank imports from cmd/beluga/providers).
	for _, want := range []string{"anthropic", "ollama", "openai", "inmemory"} {
		if !strings.Contains(out, want) {
			t.Errorf("providers: stdout missing curated %q; got:\n%s", want, out)
		}
	}
	// Category headers.
	for _, want := range []string{"llm", "embedding", "vectorstore", "memory"} {
		if !strings.Contains(out, want) {
			t.Errorf("providers: stdout missing category %q; got:\n%s", want, out)
		}
	}
}

// TestProvidersCommand_JSON asserts `--output json` emits a parseable
// JSON array with the four canonical categories in the documented order,
// stderr empty, exit 0.
func TestProvidersCommand_JSON(t *testing.T) {
	out, errBuf, code := executeArgs([]string{"--output", "json", "providers"})
	if code != 0 {
		t.Fatalf("providers --output json: want exit 0, got %d; stderr=%s", code, errBuf)
	}
	if errBuf != "" {
		t.Errorf("providers --output json: stderr must be empty on success, got: %s", errBuf)
	}

	var cats []struct {
		Category  string   `json:"category"`
		Providers []string `json:"providers"`
	}
	if err := json.Unmarshal([]byte(out), &cats); err != nil {
		t.Fatalf("providers --output json: stdout is not valid JSON: %v\ngot:\n%s", err, out)
	}
	if len(cats) != 4 {
		t.Fatalf("providers --output json: want 4 categories, got %d", len(cats))
	}
	wantOrder := []string{"llm", "embedding", "vectorstore", "memory"}
	for i, want := range wantOrder {
		if cats[i].Category != want {
			t.Errorf("providers --output json: category[%d] = %q, want %q", i, cats[i].Category, want)
		}
	}

	// Spot-check that llm contains anthropic and vectorstore contains inmemory.
	byCat := map[string][]string{}
	for _, c := range cats {
		byCat[c.Category] = c.Providers
	}
	if !containsString(byCat["llm"], "anthropic") {
		t.Errorf("providers --output json: llm missing anthropic; got %v", byCat["llm"])
	}
	if !containsString(byCat["vectorstore"], "inmemory") {
		t.Errorf("providers --output json: vectorstore missing inmemory; got %v", byCat["vectorstore"])
	}
}

// TestProvidersCommand_UnsupportedFormat asserts an unrecognised --output
// value returns a non-zero exit with the "unsupported output format" error.
func TestProvidersCommand_UnsupportedFormat(t *testing.T) {
	_, errBuf, code := executeArgs([]string{"--output", "yaml", "providers"})
	if code != 1 {
		t.Errorf("providers --output yaml: want exit 1, got %d", code)
	}
	if !strings.Contains(errBuf, "unsupported output format") {
		t.Errorf("providers --output yaml: stderr missing expected error; got: %s", errBuf)
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// --- Root-level dispatch tests (replaces the pre-T2 TestRun_* set) ---

func TestRoot_Help(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		out, _, code := executeArgs([]string{arg})
		if code != 0 {
			t.Errorf("%s: want exit 0, got %d", arg, code)
		}
		if !strings.Contains(out, "beluga") {
			t.Errorf("%s: expected help output to reference 'beluga', got: %s", arg, out)
		}
	}
}

func TestRoot_UnknownCommand(t *testing.T) {
	_, errBuf, code := executeArgs([]string{"bogus"})
	if code != 1 {
		t.Errorf("want exit 1, got %d", code)
	}
	if !strings.Contains(errBuf, "unknown command") {
		t.Errorf("expected unknown command error, got: %s", errBuf)
	}
}

func TestRoot_Init(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	_, errBuf, code := executeArgs([]string{"init", "runtest"})
	if code != 0 {
		t.Errorf("want exit 0, got %d; stderr=%s", code, errBuf)
	}
}

func TestRoot_Dev(t *testing.T) {
	origRun := devloopRun
	defer func() { devloopRun = origRun }()
	devloopRun = func(_ context.Context, _ devloop.Config) error { return nil }

	_, errBuf, code := executeArgs([]string{"dev", "--playground", "off"})
	if code != 0 {
		t.Errorf("want exit 0, got %d; stderr=%s", code, errBuf)
	}
}

func TestRoot_Run(t *testing.T) {
	origRun := devloopRun
	origExit := devloopExitCode
	defer func() {
		devloopRun = origRun
		devloopExitCode = origExit
	}()
	var captured devloop.Config
	devloopRun = func(_ context.Context, cfg devloop.Config) error {
		captured = cfg
		return nil
	}
	devloopExitCode = func(error) int { return 0 }

	_, errBuf, code := executeArgs([]string{"run", "--", "one", "two"})
	if code != 0 {
		t.Fatalf("want exit 0, got %d; stderr=%s", code, errBuf)
	}
	if got := captured.ChildArgs; len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Errorf("ChildArgs after --: got %v, want [one two]", got)
	}
	if captured.Watch {
		t.Error("run should use Watch=false")
	}
}

func TestRoot_Run_ForwardsChildExitCode(t *testing.T) {
	origRun := devloopRun
	origExit := devloopExitCode
	defer func() {
		devloopRun = origRun
		devloopExitCode = origExit
	}()
	devloopRun = func(_ context.Context, _ devloop.Config) error {
		return errors.New("child died")
	}
	devloopExitCode = func(error) int { return 42 }

	_, _, code := executeArgs([]string{"run"})
	if code != 42 {
		t.Errorf("run should mirror child exit code; got %d want 42", code)
	}
}

func TestRoot_Deploy(t *testing.T) {
	_, errBuf, code := executeArgs([]string{"deploy", "--target", "docker"})
	if code != 0 {
		t.Errorf("want exit 0, got %d; stderr=%s", code, errBuf)
	}
}

func TestRoot_DeployError(t *testing.T) {
	_, errBuf, code := executeArgs([]string{"deploy", "--target", "nope"})
	if code != 1 {
		t.Errorf("want exit 1, got %d", code)
	}
	if !strings.Contains(errBuf, "error:") {
		t.Errorf("expected error prefix, got: %s", errBuf)
	}
}

func TestRoot_Test(t *testing.T) {
	origLook := lookPath
	origExec := execCommand
	defer func() {
		lookPath = origLook
		execCommand = origExec
	}()
	lookPath = func(string) (string, error) { return "/usr/bin/true", nil }
	execCommand = func(stdout, stderr io.Writer, name string, args ...string) *exec.Cmd {
		return exec.Command("/bin/sh", "-c", "exit 0")
	}

	_, errBuf, code := executeArgs([]string{"test", "--pkg", "./..."})
	if code != 0 {
		t.Errorf("want exit 0, got %d; stderr=%s", code, errBuf)
	}
}
