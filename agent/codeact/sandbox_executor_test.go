package codeact

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/tool/sandbox"
)

type fakeSandbox struct {
	gotCode string
	gotCfg  sandbox.SandboxConfig
	err     error
}

func (f *fakeSandbox) Execute(_ context.Context, code string, cfg sandbox.SandboxConfig) (sandbox.ExecutionResult, error) {
	f.gotCode, f.gotCfg = code, cfg
	return sandbox.ExecutionResult{Output: "out", Error: "warn", ExitCode: 3, Duration: time.Second}, f.err
}

func (f *fakeSandbox) Close(context.Context) error { return nil }

// TestSandboxExecutor is a regression test for FWC-33: codeact had its own
// execution path with no way to use the shared sandbox backend.
func TestSandboxExecutor(t *testing.T) {
	fs := &fakeSandbox{}
	e := NewSandboxExecutor(fs, sandbox.SandboxConfig{Language: "bash", Timeout: time.Minute, NetworkPolicy: sandbox.NetworkIsolated})
	res, err := e.Execute(context.Background(), CodeAction{Language: "python", Code: "print(1)", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if fs.gotCode != "print(1)" || fs.gotCfg.Language != "python" || fs.gotCfg.Timeout != 2*time.Second || fs.gotCfg.NetworkPolicy != sandbox.NetworkIsolated {
		t.Errorf("sandbox got code=%q cfg=%+v", fs.gotCode, fs.gotCfg)
	}
	if res.Output != "out" || res.Error != "warn" || res.ExitCode != 3 || res.Duration != time.Second {
		t.Errorf("result = %+v", res)
	}
	boom := errors.New("boom")
	fs.err = boom
	if _, err := e.Execute(context.Background(), CodeAction{Code: "x"}); !errors.Is(err, boom) || fs.gotCfg.Language != "bash" {
		t.Errorf("err = %v lang = %q", err, fs.gotCfg.Language)
	}
}
