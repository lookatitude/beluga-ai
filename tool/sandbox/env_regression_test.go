package sandbox

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// TestProcessSandbox_DoesNotLeakParentEnv is a regression test for FWC-16:
// with no WithEnv the child inherited the full parent environment, including
// provider API keys.
func TestProcessSandbox_DoesNotLeakParentEnv(t *testing.T) {
	if !hasBash() {
		t.Skip("bash not available")
	}
	t.Setenv("BELUGA_FAKE_API_KEY", "sk-should-not-leak")
	sb := NewProcessSandbox()
	res, err := sb.Execute(context.Background(), `echo "key=${BELUGA_FAKE_API_KEY:-unset}"; echo "home=$HOME"`, SandboxConfig{
		Language: "bash",
		Timeout:  10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute: %v (stderr %q)", err, res.Error)
	}
	if strings.Contains(res.Output, "sk-should-not-leak") {
		t.Fatalf("parent secret leaked into sandbox: %q", res.Output)
	}
	if !strings.Contains(res.Output, "key=unset") {
		t.Errorf("output = %q, want key=unset", res.Output)
	}
	if !strings.Contains(res.Output, "home=") || strings.Contains(res.Output, "home=\n") {
		t.Errorf("HOME not set to the temp dir: %q", res.Output)
	}
}

func TestProcessSandbox_ExplicitEnvReplacesDefault(t *testing.T) {
	if !hasBash() {
		t.Skip("bash not available")
	}
	sb := NewProcessSandbox(WithEnv(append(MinimalEnv(t.TempDir()), "GREETING=hi")))
	res, err := sb.Execute(context.Background(), `echo "$GREETING"`, SandboxConfig{Language: "bash", Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(res.Output) != "hi" {
		t.Errorf("output = %q", res.Output)
	}
}

// TestProcessSandbox_RejectsUnenforceableResources is a regression test for
// FWC-16: MemoryMB and CPUs were silently ignored.
func TestProcessSandbox_RejectsUnenforceableResources(t *testing.T) {
	sb := NewProcessSandbox()
	for name, res := range map[string]ResourceLimits{
		"memory": {MemoryMB: 128},
		"cpus":   {CPUs: 0.5},
	} {
		_, err := sb.Execute(context.Background(), "echo hi", SandboxConfig{Language: "bash", Resources: res})
		if core.CodeOf(err, "") != core.ErrInvalidInput {
			t.Errorf("%s: err = %v, want invalid_input", name, err)
		}
	}
}

// TestLimitedWriter_CombinedBudget is a regression test for FWC-16:
// MaxOutputBytes was applied to stdout and stderr separately.
func TestLimitedWriter_CombinedBudget(t *testing.T) {
	budget := &outputBudget{remaining: 6}
	var so, se bytes.Buffer
	wOut := &limitedWriter{buf: &so, budget: budget}
	wErr := &limitedWriter{buf: &se, budget: budget}
	if n, _ := wOut.Write([]byte("abcd")); n != 4 {
		t.Fatalf("n = %d", n)
	}
	if n, _ := wErr.Write([]byte("efgh")); n != 4 {
		t.Fatalf("n = %d, want full length reported", n)
	}
	if total := so.Len() + se.Len(); total != 6 {
		t.Errorf("combined captured = %d, want 6", total)
	}
}
