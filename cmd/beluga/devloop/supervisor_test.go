package devloop

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  Config
		want string
	}{
		{"missing root", Config{Stdout: io.Discard, Stderr: io.Discard}, "ProjectRoot"},
		{"missing stdout", Config{ProjectRoot: "/x", Stderr: io.Discard}, "Stdout and Stderr"},
		{"watch without filter", Config{ProjectRoot: "/x", Stdout: io.Discard, Stderr: io.Discard, Watch: true}, "Filter is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfig(&tc.cfg)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestValidateConfig_AppliesDefaults(t *testing.T) {
	t.Parallel()
	cfg := Config{
		ProjectRoot: "/x",
		Stdout:      io.Discard,
		Stderr:      io.Discard,
	}
	if err := validateConfig(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Debounce != defaultDebounce {
		t.Fatalf("Debounce=%v want %v", cfg.Debounce, defaultDebounce)
	}
	if cfg.GraceTimeout != defaultGrace {
		t.Fatalf("GraceTimeout=%v want %v", cfg.GraceTimeout, defaultGrace)
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()
	if got := ExitCode(nil); got != 0 {
		t.Fatalf("nil → %d want 0", got)
	}
	if got := ExitCode(errors.New("other")); got != 1 {
		t.Fatalf("plain error → %d want 1", got)
	}
}

// fixtureProject builds a tiny `package main` that prints "hello" and
// exits 0. Used for end-to-end supervisor smoke.
func fixtureProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module devloop_fixture\n\ngo 1.25\n")
	write("main.go", `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`)
	return dir
}

func TestRun_OneShot_Success(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain required")
	}
	dir := fixtureProject(t)

	var out, errBuf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	err := Run(ctx, Config{
		ProjectRoot: dir,
		Stdout:      &out,
		Stderr:      &errBuf,
	})
	if err != nil {
		t.Fatalf("Run returned %v (stderr=%s)", err, errBuf.String())
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("stdout missing 'hello': %q", out.String())
	}
}

func TestRun_OneShot_ChildNonzeroExit(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain required")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module devloop_fixture_err\n\ngo 1.25\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import "os"

func main() { os.Exit(2) }
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var out, errBuf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	err := Run(ctx, Config{ProjectRoot: dir, Stdout: &out, Stderr: &errBuf})
	if err == nil {
		t.Fatal("want non-nil error, got nil")
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("ExitCode=%d want 2", got)
	}
}

func TestRun_BaseEnvIncludesDotEnv(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain required")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module devloop_fixture_env\n\ngo 1.25\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("val=" + os.Getenv("DEVLOOP_TEST_KEY"))
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DEVLOOP_TEST_KEY=from_dotenv\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out, errBuf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := Run(ctx, Config{ProjectRoot: dir, Stdout: &out, Stderr: &errBuf}); err != nil {
		t.Fatalf("Run returned %v", err)
	}
	if !strings.Contains(out.String(), "val=from_dotenv") {
		t.Fatalf("expected .env value, got %q", out.String())
	}
}
