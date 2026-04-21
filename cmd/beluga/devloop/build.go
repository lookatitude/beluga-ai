package devloop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// lookPath is indirected so tests can stub binary resolution.
var lookPath = exec.LookPath

// execCommand is indirected so tests can stub command construction.
// The production implementation resolves the binary via an absolute
// path (so execution does not rely on PATH lookup at Run time) and
// wires stdout/stderr through to the caller.
var execCommand = func(ctx context.Context, stdout, stderr io.Writer, name string, args ...string) *exec.Cmd {
	// #nosec G204 -- `name` is always an absolute path resolved via
	// exec.LookPath("go"). `args` is built from a fixed template
	// (-o <tmp path>, ".") with no caller-supplied values. No shell.
	c := exec.CommandContext(ctx, name, args...) //nolint:gosec // G204: see nosec justification above
	c.Stdout = stdout
	c.Stderr = stderr
	return c
}

// BuildResult is returned by [BuildBinary] on success. OutputPath is the
// absolute path of the compiled binary; the caller owns cleanup.
type BuildResult struct {
	OutputPath string
}

// BuildBinary compiles the scaffolded project at projectRoot into a
// binary placed under os.TempDir(). The filename is
// `beluga-app-<sha256prefix>-<seq>[.exe]` where <sha256prefix> is a
// stable hash of the absolute projectRoot so concurrent `beluga dev`
// sessions on different projects do not collide on the same temp path,
// and <seq> is a monotonic counter supplied by the supervisor for
// unambiguous cleanup of prior binaries.
//
// The caller is responsible for deleting OutputPath after the
// corresponding child has exited.
func BuildBinary(ctx context.Context, projectRoot string, seq int, stdout, stderr io.Writer) (*BuildResult, error) {
	goBin, err := lookPath("go")
	if err != nil {
		return nil, fmt.Errorf("locate go toolchain: %w", err)
	}
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}

	sum := sha256.Sum256([]byte(absRoot))
	prefix := hex.EncodeToString(sum[:6])
	name := fmt.Sprintf("beluga-app-%s-%d", prefix, seq)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	out := filepath.Join(os.TempDir(), name)

	cmd := execCommand(ctx, stdout, stderr, goBin, "build", "-o", out, ".")
	cmd.Dir = absRoot
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go build: %w", err)
	}
	return &BuildResult{OutputPath: out}, nil
}
