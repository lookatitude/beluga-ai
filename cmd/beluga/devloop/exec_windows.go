//go:build windows

package devloop

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"syscall"
	"time"
)

// createNewProcessGroup is the Windows CREATE_NEW_PROCESS_GROUP flag.
// Declared here so it does not require importing x/sys/windows.
const createNewProcessGroup = 0x00000200

// ctrlBreakEvent is the Windows CTRL_BREAK_EVENT signal code passed to
// GenerateConsoleCtrlEvent. Using CTRL_C_EVENT (value 0) would be
// silently ignored by a process in a new process group — only
// CTRL_BREAK_EVENT is delivered, which is the cross-platform footgun
// the DX-1 S3 brief calls out explicitly.
const ctrlBreakEvent = 1

// newChildCmd constructs the exec.Cmd for a compiled project binary on
// windows. CREATE_NEW_PROCESS_GROUP lets the supervisor deliver a
// CTRL_BREAK_EVENT to the child without also hitting itself.
func newChildCmd(ctx context.Context, binPath string, env []string, stdin io.Reader, stdout, stderr io.Writer, args ...string) *exec.Cmd {
	// #nosec G204 -- binPath is the absolute temp-file path produced
	// by BuildBinary; it is never directly supplied by a remote caller.
	// args are caller-supplied argv tail (passthrough from `beluga
	// run -- ...`), never shell-interpreted.
	cmd := exec.CommandContext(ctx, binPath, args...) //nolint:gosec // G204: see nosec justification above
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
	return cmd
}

// signalProcessGroup delivers CTRL_BREAK_EVENT to the child's process
// group. The generic signal argument is accepted for parity with the
// unix implementation but is ignored — Windows consoles only support
// CTRL_C / CTRL_BREAK events, not arbitrary POSIX signals.
func signalProcessGroup(cmd *exec.Cmd, _ syscall.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return errors.New("signalProcessGroup: process not started")
	}
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}
	defer func() { _ = dll.Release() }()
	proc, err := dll.FindProc("GenerateConsoleCtrlEvent")
	if err != nil {
		return err
	}
	r1, _, err := proc.Call(uintptr(ctrlBreakEvent), uintptr(cmd.Process.Pid))
	if r1 == 0 {
		return err
	}
	return nil
}

// terminateGracefully sends CTRL_BREAK_EVENT to the child, waits up to
// grace for it to exit, and escalates to Process.Kill on timeout.
func terminateGracefully(cmd *exec.Cmd, waitCh <-chan error, grace time.Duration) error {
	_ = signalProcessGroup(cmd, syscall.SIGTERM) // signal arg ignored on Windows
	select {
	case err := <-waitCh:
		return err
	case <-time.After(grace):
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return <-waitCh
	}
}
