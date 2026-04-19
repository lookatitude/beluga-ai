package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// executeArgs runs the cobra root with the given args and returns captured
// stdout/stderr plus the exit code from Execute. It is the post-T2 replacement
// for the pre-cobra run() helper.
func executeArgs(args []string) (stdout, stderr string, code int) {
	var out, errBuf bytes.Buffer
	cmd := newRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	if err := cmd.Execute(); err != nil {
		// Match the Execute() function's formatting so tests see the same
		// stderr contract they would in production.
		_, _ = errBuf.WriteString("error: " + err.Error() + "\n")
		code = 1
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

func TestCmdInit(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	projDir := filepath.Join(dir, "myproject")

	err := executeSubcommand(newInitCmd(), []string{"--name", "test-project", "--dir", projDir})
	if err != nil {
		t.Fatalf("newInitCmd: %v", err)
	}

	// Verify directories were created.
	for _, sub := range []string{"agents", "tools", "config"} {
		path := filepath.Join(projDir, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("directory %s not created: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", sub)
		}
	}

	// Verify config file.
	configPath := filepath.Join(projDir, "config", "agent.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if len(data) == 0 {
		t.Error("config file is empty")
	}
	if !strings.Contains(string(data), "test-project-agent") {
		t.Errorf("config missing agent id: %s", data)
	}

	// Verify main.go.
	mainPath := filepath.Join(projDir, "main.go")
	data, err = os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if len(data) == 0 {
		t.Error("main.go is empty")
	}
}

func TestCmdInit_DefaultName(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	projDir := filepath.Join(dir, "derived-name")

	if err := executeSubcommand(newInitCmd(), []string{"--dir", projDir}); err != nil {
		t.Fatalf("newInitCmd: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(projDir, "config", "agent.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "derived-name-agent") {
		t.Errorf("expected default name to derive from dir, got: %s", data)
	}
}

func TestCmdInit_PathTraversal(t *testing.T) {
	t.Chdir(t.TempDir())
	err := executeSubcommand(newInitCmd(), []string{"--dir", "/tmp/../etc/passwd"})
	if err == nil {
		t.Error("expected error for absolute path traversal")
	}
	if err != nil && !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("expected path traversal error, got: %v", err)
	}
}

func TestCmdInit_RelativeTraversal(t *testing.T) {
	t.Chdir(t.TempDir())
	err := executeSubcommand(newInitCmd(), []string{"--dir", "../escape"})
	if err == nil {
		t.Error("expected error for relative path traversal")
	}
}

func TestCmdDev(t *testing.T) {
	err := executeSubcommand(newDevCmd(), []string{"--port", "9090"})
	if err != nil {
		t.Errorf("newDevCmd: %v", err)
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
	_, errBuf, code := executeArgs([]string{"init", "--name", "runtest", "--dir", filepath.Join(dir, "p")})
	if code != 0 {
		t.Errorf("want exit 0, got %d; stderr=%s", code, errBuf)
	}
}

func TestRoot_Dev(t *testing.T) {
	_, errBuf, code := executeArgs([]string{"dev", "--port", "7777"})
	if code != 0 {
		t.Errorf("want exit 0, got %d; stderr=%s", code, errBuf)
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
