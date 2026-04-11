package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdInit(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	projDir := filepath.Join(dir, "myproject")

	err := cmdInit([]string{"-name", "test-project", "-dir", projDir})
	if err != nil {
		t.Fatalf("cmdInit: %v", err)
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

	if err := cmdInit([]string{"-dir", projDir}); err != nil {
		t.Fatalf("cmdInit: %v", err)
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
	err := cmdInit([]string{"-dir", "/tmp/../etc/passwd"})
	if err == nil {
		t.Error("expected error for absolute path traversal")
	}
	if err != nil && !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("expected path traversal error, got: %v", err)
	}
}

func TestCmdInit_RelativeTraversal(t *testing.T) {
	t.Chdir(t.TempDir())
	err := cmdInit([]string{"-dir", "../escape"})
	if err == nil {
		t.Error("expected error for relative path traversal")
	}
}

func TestCmdDev(t *testing.T) {
	err := cmdDev([]string{"-port", "9090"})
	if err != nil {
		t.Errorf("cmdDev: %v", err)
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
			err := cmdDeploy([]string{"-target", tt.target})
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
	err := cmdTest([]string{"-pkg", "./... -exec evil"})
	if err == nil || !strings.Contains(err.Error(), "invalid package pattern") {
		t.Errorf("expected invalid package pattern error, got: %v", err)
	}
}

func TestCmdTest_ParseError(t *testing.T) {
	err := cmdTest([]string{"--nope"})
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestCmdTest_LookPathFailure(t *testing.T) {
	orig := lookPath
	defer func() { lookPath = orig }()
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }

	err := cmdTest([]string{"-pkg", "./..."})
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

	if err := cmdTest([]string{"-v", "-race", "-pkg", "./..."}); err != nil {
		t.Errorf("cmdTest: %v", err)
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

	if err := cmdTest([]string{"-pkg", "./..."}); err == nil {
		t.Error("expected non-zero exit error")
	}
}

func TestRun_NoArgs(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run(nil, &out, &errBuf)
	if code != 1 {
		t.Errorf("want exit 1, got %d", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("expected usage output, got: %s", out.String())
	}
}

func TestRun_Version(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"version"}, &out, &errBuf)
	if code != 0 {
		t.Errorf("want exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "beluga v") {
		t.Errorf("expected version output, got: %s", out.String())
	}
}

func TestRun_Help(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		var out, errBuf bytes.Buffer
		code := run([]string{arg}, &out, &errBuf)
		if code != 0 {
			t.Errorf("%s: want exit 0, got %d", arg, code)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("%s: expected usage output", arg)
		}
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"bogus"}, &out, &errBuf)
	if code != 1 {
		t.Errorf("want exit 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "unknown command") {
		t.Errorf("expected unknown command error, got: %s", errBuf.String())
	}
}

func TestRun_Init(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	var out, errBuf bytes.Buffer
	code := run([]string{"init", "-name", "runtest", "-dir", filepath.Join(dir, "p")}, &out, &errBuf)
	if code != 0 {
		t.Errorf("want exit 0, got %d; stderr=%s", code, errBuf.String())
	}
}

func TestRun_Dev(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"dev", "-port", "7777"}, &out, &errBuf)
	if code != 0 {
		t.Errorf("want exit 0, got %d", code)
	}
}

func TestRun_Deploy(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"deploy", "-target", "docker"}, &out, &errBuf)
	if code != 0 {
		t.Errorf("want exit 0, got %d", code)
	}
}

func TestRun_DeployError(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"deploy", "-target", "nope"}, &out, &errBuf)
	if code != 1 {
		t.Errorf("want exit 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "error:") {
		t.Errorf("expected error prefix, got: %s", errBuf.String())
	}
}

func TestRun_Test(t *testing.T) {
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
	code := run([]string{"test", "-pkg", "./..."}, &out, &errBuf)
	if code != 0 {
		t.Errorf("want exit 0, got %d; stderr=%s", code, errBuf.String())
	}
}

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	for _, want := range []string{"init", "dev", "test", "deploy", "version", "help"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("usage missing %q", want)
		}
	}
}
