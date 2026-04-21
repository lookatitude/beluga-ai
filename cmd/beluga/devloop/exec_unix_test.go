//go:build unix

package devloop

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestSignalProcessGroup_NilCmd documents the defensive-guard contract.
func TestSignalProcessGroup_NilCmd(t *testing.T) {
	t.Parallel()
	if err := signalProcessGroup(nil, syscall.SIGTERM); err == nil {
		t.Fatal("nil cmd must error, got nil")
	}
	if err := signalProcessGroup(&exec.Cmd{}, syscall.SIGTERM); err == nil || !strings.Contains(err.Error(), "not started") {
		t.Fatalf("unstarted cmd must error with 'not started', got %v", err)
	}
}

// TestTerminateGracefully_ExitsOnSIGTERM spawns `/bin/sleep 60` in its
// own process group and calls terminateGracefully. The child must exit
// before the grace window and never reach the SIGKILL branch.
func TestTerminateGracefully_ExitsOnSIGTERM(t *testing.T) {
	if _, err := exec.LookPath("/bin/sleep"); err != nil {
		t.Skip("/bin/sleep required")
	}
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := newChildCmd(ctx, "/bin/sleep", nil, nil, io.Discard, io.Discard, "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	waitCh := waitAsync(cmd)
	start := time.Now()
	err := terminateGracefully(cmd, waitCh, 3*time.Second)
	took := time.Since(start)
	if err == nil {
		t.Fatal("sleep terminated without an error — signal path lost")
	}
	// Sleep exits in tens of milliseconds after SIGTERM; any jump to
	// the SIGKILL path would push took past the 3s grace.
	if took > 2*time.Second {
		t.Fatalf("terminateGracefully took %v; grace path may have fired", took)
	}
}

// TestTerminateGracefully_EscalatesOnIgnoredSIGTERM covers the SIGKILL
// escalation branch by running a shell that traps SIGTERM. With a tight
// grace window, terminateGracefully must escalate to SIGKILL and still
// return a non-nil error.
func TestTerminateGracefully_EscalatesOnIgnoredSIGTERM(t *testing.T) {
	if _, err := exec.LookPath("/bin/sh"); err != nil {
		t.Skip("/bin/sh required")
	}
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Trap SIGTERM so the child only exits on SIGKILL.
	cmd := newChildCmd(ctx, "/bin/sh", nil, nil, io.Discard, io.Discard, "-c", "trap '' TERM; while :; do sleep 1; done")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	waitCh := waitAsync(cmd)
	err := terminateGracefully(cmd, waitCh, 250*time.Millisecond)
	if err == nil {
		t.Fatal("SIGKILL path must produce a non-nil exit error")
	}
}
