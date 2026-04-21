package devloop

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildBinary_StubbedExec(t *testing.T) {
	origLook, origExec := lookPath, execCommand
	defer func() { lookPath, execCommand = origLook, origExec }()

	lookPath = func(bin string) (string, error) {
		if bin != "go" {
			t.Fatalf("unexpected lookPath %q", bin)
		}
		return "/usr/bin/go", nil
	}
	var gotArgs []string
	var gotDir string
	execCommand = func(_ context.Context, _, _ io.Writer, _ string, args ...string) *exec.Cmd {
		gotArgs = append([]string(nil), args...)
		// produce a cmd that reports gotDir via its Dir after Run
		c := exec.Command("true") // runnable on unix; windows path below
		if runtime.GOOS == "windows" {
			c = exec.Command("cmd", "/c", "exit", "0")
		}
		return c
	}

	projDir := t.TempDir()
	res, err := BuildBinary(context.Background(), projDir, 7, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	// record: exec.Cmd.Dir is set inside BuildBinary on the returned
	// cmd, not our stubbed cmd — so we assert via the filename scheme.
	if len(gotArgs) < 4 || gotArgs[0] != "build" || gotArgs[1] != "-o" {
		t.Fatalf("want [build -o <out> .], got %q", gotArgs)
	}
	outArg := gotArgs[2]
	base := filepath.Base(outArg)
	if !strings.HasPrefix(base, "beluga-app-") {
		t.Fatalf("output filename prefix wrong: %q", base)
	}
	if !strings.HasSuffix(base, "-7") && !strings.HasSuffix(base, "-7.exe") {
		t.Fatalf("output filename seq wrong: %q", base)
	}
	if res.OutputPath != outArg {
		t.Fatalf("OutputPath %q != invoked output %q", res.OutputPath, outArg)
	}
	_ = gotDir
}

func TestBuildBinary_DeterministicFilenamePerRoot(t *testing.T) {
	origLook, origExec := lookPath, execCommand
	defer func() { lookPath, execCommand = origLook, origExec }()

	lookPath = func(string) (string, error) { return "/usr/bin/go", nil }
	captured := make([]string, 0, 2)
	execCommand = func(_ context.Context, _, _ io.Writer, _ string, args ...string) *exec.Cmd {
		captured = append(captured, args[2])
		if runtime.GOOS == "windows" {
			return exec.Command("cmd", "/c", "exit", "0")
		}
		return exec.Command("true")
	}

	proj := t.TempDir()
	if _, err := BuildBinary(context.Background(), proj, 1, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildBinary(context.Background(), proj, 2, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if len(captured) != 2 {
		t.Fatalf("want 2 invocations, got %d", len(captured))
	}
	// Same root → same SHA prefix → only seq differs.
	b1 := filepath.Base(captured[0])
	b2 := filepath.Base(captured[1])
	p1 := strings.TrimSuffix(strings.TrimSuffix(b1, ".exe"), "-1")
	p2 := strings.TrimSuffix(strings.TrimSuffix(b2, ".exe"), "-2")
	if p1 != p2 {
		t.Fatalf("hash prefix differs across builds of same root: %q vs %q", p1, p2)
	}
}

func TestBuildBinary_GoToolchainMissing(t *testing.T) {
	origLook := lookPath
	defer func() { lookPath = origLook }()
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }

	var buf bytes.Buffer
	_, err := BuildBinary(context.Background(), t.TempDir(), 1, &buf, &buf)
	if err == nil || !strings.Contains(err.Error(), "locate go toolchain") {
		t.Fatalf("want toolchain-missing error, got %v", err)
	}
}
