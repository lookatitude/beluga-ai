package devloop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeEnv(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadProjectEnv_Basic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeEnv(t, dir, "# header comment\nFOO=bar\nBAZ=qux quux\n\n# trailing\n")
	out, err := LoadProjectEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	wantAll := []string{"FOO=bar", "BAZ=qux quux"}
	if len(out) != len(wantAll) {
		t.Fatalf("len=%d want %d: %q", len(out), len(wantAll), out)
	}
	for i, w := range wantAll {
		if out[i] != w {
			t.Fatalf("out[%d]=%q want %q", i, out[i], w)
		}
	}
}

func TestLoadProjectEnv_Missing(t *testing.T) {
	t.Parallel()
	out, err := LoadProjectEnv(t.TempDir())
	if err != nil {
		t.Fatalf("missing .env should not error: %v", err)
	}
	if out != nil {
		t.Fatalf("want nil got %q", out)
	}
}

func TestLoadProjectEnv_Quoting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeEnv(t, dir, `SINGLE='value with spaces'
DOUBLE="line1\nline2"
INLINE=val # comment
`)
	out, err := LoadProjectEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"SINGLE=value with spaces",
		"DOUBLE=line1\nline2",
		"INLINE=val",
	}
	for i, w := range want {
		if i >= len(out) || out[i] != w {
			t.Fatalf("out[%d]=%q want %q (all=%q)", i, out, w, out)
		}
	}
}

func TestLoadProjectEnv_InvalidKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeEnv(t, dir, "1BAD=value\n")
	_, err := LoadProjectEnv(dir)
	if err == nil || !strings.Contains(err.Error(), "invalid key") {
		t.Fatalf("want invalid-key error, got %v", err)
	}
}

func TestLoadProjectEnv_MalformedLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeEnv(t, dir, "no equals sign\n")
	_, err := LoadProjectEnv(dir)
	if err == nil || !strings.Contains(err.Error(), "expected KEY=value") {
		t.Fatalf("want malformed-line error, got %v", err)
	}
}

func TestLoadProjectEnv_SymlinkEscape(t *testing.T) {
	t.Parallel()
	outside := t.TempDir()
	target := filepath.Join(outside, "secrets.env")
	if err := os.WriteFile(target, []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	proj := t.TempDir()
	if err := os.Symlink(target, filepath.Join(proj, ".env")); err != nil {
		t.Skipf("platform does not support symlinks: %v", err)
	}
	_, err := LoadProjectEnv(proj)
	if err == nil || !strings.Contains(err.Error(), "escapes project root") {
		t.Fatalf("want escape error, got %v", err)
	}
}

func TestMergeEnv_LastWins(t *testing.T) {
	t.Parallel()
	got := MergeEnv(
		[]string{"A=os1", "B=os2", "KEEP=1"},
		[]string{"A=env", "C=env"},
		[]string{"B=flag"},
	)
	want := []string{"A=env", "B=flag", "KEEP=1", "C=env"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %q", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("got[%d]=%q want %q", i, got[i], w)
		}
	}
}

func TestMergeEnv_IgnoresMalformedEntry(t *testing.T) {
	t.Parallel()
	got := MergeEnv([]string{"OK=1", "malformed", "B=2"}, nil, nil)
	want := []string{"OK=1", "B=2"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %q", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("got[%d]=%q want %q", i, got[i], w)
		}
	}
}

func TestEnvKeyAllowed(t *testing.T) {
	t.Parallel()
	ok := []string{"A", "_FOO", "FOO_BAR", "abc123"}
	bad := []string{"", "1FOO", "FOO-BAR", "FOO BAR", "FOO$"}
	for _, k := range ok {
		if !envKeyAllowed(k) {
			t.Fatalf("want ok: %q", k)
		}
	}
	for _, k := range bad {
		if envKeyAllowed(k) {
			t.Fatalf("want bad: %q", k)
		}
	}
}
