//go:build unix

package devloop

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"syscall"
	"time"
)

// newChildCmd constructs the exec.Cmd for a compiled project binary on
// unix systems. Setpgid=true places the child in its own process group
// so SIGTERM to the negative PGID hits the child and any grandchildren
// it spawns (e.g. tool subprocesses) without escaping to the
// supervisor's own group.
func newChildCmd(ctx context.Context, binPath string, env []string, stdin io.Reader, stdout, stderr io.Writer) *exec.Cmd {
	// #nosec G204 -- binPath is the absolute temp-file path produced
	// by BuildBinary; it is never directly supplied by a remote caller.
	cmd := exec.CommandContext(ctx, binPath) //nolint:gosec // G204: see nosec justification above
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// signalProcessGroup sends sig to the entire process group of cmd.
// It must be called only after cmd.Start has returned successfully.
func signalProcessGroup(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return errors.New("signalProcessGroup: process not started")
	}
	return syscall.Kill(-cmd.Process.Pid, sig)
}

// terminateGracefully sends SIGTERM to the child process group, waits
// up to grace for the child to exit, and escalates to SIGKILL on
// timeout. It blocks until the child has been waited on (or the wait
// returned an error). Returns the exit error, if any.
func terminateGracefully(cmd *exec.Cmd, waitCh <-chan error, grace time.Duration) error {
	_ = signalProcessGroup(cmd, syscall.SIGTERM)
	select {
	case err := <-waitCh:
		return err
	case <-time.After(grace):
		_ = signalProcessGroup(cmd, syscall.SIGKILL)
		return <-waitCh
	}
}
