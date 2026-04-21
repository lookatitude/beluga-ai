//go:build windows

package devloop

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
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

// generateConsoleCtrlEvent caches the kernel32 DLL handle and the
// GenerateConsoleCtrlEvent procedure pointer for the lifetime of the
// process. Loading kernel32 per call — which the original implementation
// did — is wasteful and rules out any optimisation at the syscall
// boundary; sync.OnceValue makes the cache trivially goroutine-safe.
// A nil return means the DLL or proc lookup failed; callers must guard
// against it before calling into the proc.
var generateConsoleCtrlEvent = sync.OnceValue(func() *syscall.Proc {
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return nil
	}
	proc, err := dll.FindProc("GenerateConsoleCtrlEvent")
	if err != nil {
		return nil
	}
	return proc
})

// signalProcessGroup delivers CTRL_BREAK_EVENT to the child's process
// group. The generic signal argument is accepted for parity with the
// unix implementation but is ignored — Windows consoles only support
// CTRL_C / CTRL_BREAK events, not arbitrary POSIX signals.
func signalProcessGroup(cmd *exec.Cmd, _ syscall.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return errors.New("signalProcessGroup: process not started")
	}
	proc := generateConsoleCtrlEvent()
	if proc == nil {
		return errors.New("signalProcessGroup: kernel32.GenerateConsoleCtrlEvent unavailable")
	}
	r1, _, callErr := proc.Call(uintptr(ctrlBreakEvent), uintptr(cmd.Process.Pid))
	if r1 == 0 {
		return callErr
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
