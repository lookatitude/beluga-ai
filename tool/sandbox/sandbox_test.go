package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Registry Tests ---

func TestRegistryRoundtrip(t *testing.T) {
	// "process" is registered in init().
	names := ListSandboxes()
	assert.Contains(t, names, "process")

	sb, err := NewSandbox("process")
	require.NoError(t, err)
	require.NotNil(t, sb)
	assert.IsType(t, &ProcessSandbox{}, sb)
}

func TestNewSandboxUnknown(t *testing.T) {
	_, err := NewSandbox("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

// --- ProcessSandbox Tests ---

func hasPython() bool {
	_, err := exec.LookPath("python3")
	return err == nil
}

func hasBash() bool {
	_, err := exec.LookPath("bash")
	return err == nil
}

func TestProcessSandboxExecute(t *testing.T) {
	if !hasPython() {
		t.Skip("python3 not available")
	}

	sb := NewProcessSandbox()
	ctx := context.Background()

	result, err := sb.Execute(ctx, "print('hello world')", SandboxConfig{
		Language: "python",
		Timeout:  10 * time.Second,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "hello world\n", result.Output)
	assert.Empty(t, result.Error)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestProcessSandboxBash(t *testing.T) {
	if !hasBash() {
		t.Skip("bash not available")
	}

	sb := NewProcessSandbox()
	ctx := context.Background()

	result, err := sb.Execute(ctx, "echo hello from bash", SandboxConfig{
		Language: "bash",
		Timeout:  10 * time.Second,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "hello from bash\n", result.Output)
}

func TestProcessSandboxNonZeroExit(t *testing.T) {
	if !hasPython() {
		t.Skip("python3 not available")
	}

	sb := NewProcessSandbox()
	ctx := context.Background()

	result, err := sb.Execute(ctx, "import sys; sys.exit(42)", SandboxConfig{
		Language: "python",
		Timeout:  10 * time.Second,
	})
	// Non-zero exit is not a Go error.
	require.NoError(t, err)
	assert.Equal(t, 42, result.ExitCode)
}

func TestProcessSandboxStderr(t *testing.T) {
	if !hasPython() {
		t.Skip("python3 not available")
	}

	sb := NewProcessSandbox()
	ctx := context.Background()

	result, err := sb.Execute(ctx, "import sys; print('oops', file=sys.stderr); sys.exit(1)", SandboxConfig{
		Language: "python",
		Timeout:  10 * time.Second,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.ExitCode)
	assert.Contains(t, result.Error, "oops")
}

func TestProcessSandboxTimeout(t *testing.T) {
	if !hasPython() {
		t.Skip("python3 not available")
	}

	sb := NewProcessSandbox()
	ctx := context.Background()

	_, err := sb.Execute(ctx, "import time; time.sleep(60)", SandboxConfig{
		Language: "python",
		Timeout:  200 * time.Millisecond,
	})
	require.Error(t, err)
	var coreErr *core.Error
	require.True(t, errors.As(err, &coreErr))
	assert.Equal(t, core.ErrTimeout, coreErr.Code)
}

func TestProcessSandboxContextCancellation(t *testing.T) {
	if !hasPython() {
		t.Skip("python3 not available")
	}

	sb := NewProcessSandbox()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately.
	cancel()

	_, err := sb.Execute(ctx, "import time; time.sleep(60)", SandboxConfig{
		Language: "python",
		Timeout:  30 * time.Second,
	})
	require.Error(t, err)
}

func TestProcessSandboxEmptyCode(t *testing.T) {
	sb := NewProcessSandbox()
	ctx := context.Background()

	_, err := sb.Execute(ctx, "", SandboxConfig{Language: "python"})
	require.Error(t, err)
	var coreErr *core.Error
	require.True(t, errors.As(err, &coreErr))
	assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
}

func TestProcessSandboxEmptyLanguage(t *testing.T) {
	sb := NewProcessSandbox()
	ctx := context.Background()

	_, err := sb.Execute(ctx, "print('hi')", SandboxConfig{Language: ""})
	require.Error(t, err)
	var coreErr *core.Error
	require.True(t, errors.As(err, &coreErr))
	assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
}

func TestProcessSandboxUnsupportedLanguage(t *testing.T) {
	sb := NewProcessSandbox()
	ctx := context.Background()

	_, err := sb.Execute(ctx, "code", SandboxConfig{Language: "brainfuck"})
	require.Error(t, err)
	var coreErr *core.Error
	require.True(t, errors.As(err, &coreErr))
	assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
	assert.Contains(t, coreErr.Message, "unsupported language")
}

func TestProcessSandboxWithOptions(t *testing.T) {
	if !hasBash() {
		t.Skip("bash not available")
	}

	sb := NewProcessSandbox(
		WithWorkDir(t.TempDir()),
		WithEnv([]string{"PATH=/usr/bin:/bin", "MY_VAR=hello"}),
	)
	ctx := context.Background()

	result, err := sb.Execute(ctx, "echo $MY_VAR", SandboxConfig{
		Language: "bash",
		Timeout:  10 * time.Second,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "hello\n", result.Output)
}

func TestProcessSandboxClose(t *testing.T) {
	sb := NewProcessSandbox()
	err := sb.Close(context.Background())
	assert.NoError(t, err)
}

func TestProcessSandboxOutputLimit(t *testing.T) {
	if !hasPython() {
		t.Skip("python3 not available")
	}

	sb := NewProcessSandbox()
	ctx := context.Background()

	// Generate output larger than limit.
	result, err := sb.Execute(ctx, "print('A' * 2000)", SandboxConfig{
		Language: "python",
		Timeout:  10 * time.Second,
		Resources: ResourceLimits{
			MaxOutputBytes: 100,
		},
	})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result.Output), 100)
}

// --- Pool Tests ---

func TestPoolCheckoutCheckin(t *testing.T) {
	pool, err := NewSandboxPool("process", WithPoolSize(2))
	require.NoError(t, err)
	defer pool.Close(context.Background())

	ctx := context.Background()

	sb1, err := pool.Checkout(ctx)
	require.NoError(t, err)
	require.NotNil(t, sb1)

	sb2, err := pool.Checkout(ctx)
	require.NoError(t, err)
	require.NotNil(t, sb2)

	pool.Checkin(sb1)
	pool.Checkin(sb2)

	// Should be able to check out again.
	sb3, err := pool.Checkout(ctx)
	require.NoError(t, err)
	require.NotNil(t, sb3)
	pool.Checkin(sb3)
}

func TestPoolWarmup(t *testing.T) {
	pool, err := NewSandboxPool("process", WithPoolSize(2), WithWarmup(true))
	require.NoError(t, err)
	defer pool.Close(context.Background())

	ctx := context.Background()

	// Both should be available immediately from the warmed pool.
	sb1, err := pool.Checkout(ctx)
	require.NoError(t, err)
	sb2, err := pool.Checkout(ctx)
	require.NoError(t, err)

	pool.Checkin(sb1)
	pool.Checkin(sb2)
}

func TestPoolClose(t *testing.T) {
	pool, err := NewSandboxPool("process", WithPoolSize(2), WithWarmup(true))
	require.NoError(t, err)

	err = pool.Close(context.Background())
	require.NoError(t, err)

	// Checkout after close returns error.
	_, err = pool.Checkout(context.Background())
	require.Error(t, err)
}

func TestPoolCheckinAfterClose(t *testing.T) {
	pool, err := NewSandboxPool("process", WithPoolSize(2))
	require.NoError(t, err)

	sb, err := pool.Checkout(context.Background())
	require.NoError(t, err)

	err = pool.Close(context.Background())
	require.NoError(t, err)

	// Checkin after close should not panic — sandbox is closed.
	pool.Checkin(sb)
}

func TestPoolCheckinNil(t *testing.T) {
	pool, err := NewSandboxPool("process", WithPoolSize(2))
	require.NoError(t, err)
	defer pool.Close(context.Background())

	// Should not panic.
	pool.Checkin(nil)
}

func TestPoolInvalidSize(t *testing.T) {
	_, err := NewSandboxPool("process", WithPoolSize(0))
	require.Error(t, err)

	_, err = NewSandboxPool("process", WithPoolSize(-1))
	require.Error(t, err)
}

// --- SandboxTool Tests ---

func TestSandboxToolInterface(t *testing.T) {
	sb := NewProcessSandbox()
	st := NewSandboxTool(sb)

	assert.Equal(t, "code_sandbox", st.Name())
	assert.NotEmpty(t, st.Description())

	schema := st.InputSchema()
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, props, "code")
	assert.Contains(t, props, "language")

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "code")
	assert.Contains(t, required, "language")
}

func TestSandboxToolExecute(t *testing.T) {
	if !hasPython() {
		t.Skip("python3 not available")
	}

	sb := NewProcessSandbox()
	st := NewSandboxTool(sb)
	ctx := context.Background()

	result, err := st.Execute(ctx, map[string]any{
		"code":     "print('tool output')",
		"language": "python",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)
}

func TestSandboxToolMissingCode(t *testing.T) {
	sb := NewProcessSandbox()
	st := NewSandboxTool(sb)
	ctx := context.Background()

	_, err := st.Execute(ctx, map[string]any{
		"language": "python",
	})
	require.Error(t, err)
}

func TestSandboxToolMissingLanguage(t *testing.T) {
	sb := NewProcessSandbox()
	st := NewSandboxTool(sb)
	ctx := context.Background()

	_, err := st.Execute(ctx, map[string]any{
		"code": "print('hi')",
	})
	require.Error(t, err)
}

func TestSandboxToolNonZeroExit(t *testing.T) {
	if !hasPython() {
		t.Skip("python3 not available")
	}

	sb := NewProcessSandbox()
	st := NewSandboxTool(sb)
	ctx := context.Background()

	result, err := st.Execute(ctx, map[string]any{
		"code":     "import sys; sys.exit(1)",
		"language": "python",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestSandboxToolWithTimeout(t *testing.T) {
	sb := NewProcessSandbox()
	st := NewSandboxTool(sb, WithToolTimeout(5*time.Second))
	assert.Equal(t, 5*time.Second, st.timeout)
}

// --- Hooks Tests ---

func TestComposeHooksBeforeExecute(t *testing.T) {
	var calls []string

	h1 := Hooks{
		BeforeExecute: func(_ context.Context, code string, _ SandboxConfig) error {
			calls = append(calls, "h1:"+code)
			return nil
		},
	}
	h2 := Hooks{
		BeforeExecute: func(_ context.Context, code string, _ SandboxConfig) error {
			calls = append(calls, "h2:"+code)
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeExecute(context.Background(), "test", SandboxConfig{})
	require.NoError(t, err)
	assert.Equal(t, []string{"h1:test", "h2:test"}, calls)
}

func TestComposeHooksBeforeExecuteShortCircuit(t *testing.T) {
	sentinel := errors.New("stop")
	var calls []string

	h1 := Hooks{
		BeforeExecute: func(_ context.Context, _ string, _ SandboxConfig) error {
			calls = append(calls, "h1")
			return sentinel
		},
	}
	h2 := Hooks{
		BeforeExecute: func(_ context.Context, _ string, _ SandboxConfig) error {
			calls = append(calls, "h2")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeExecute(context.Background(), "test", SandboxConfig{})
	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, []string{"h1"}, calls)
}

func TestComposeHooksAfterExecute(t *testing.T) {
	var calls int

	h := ComposeHooks(
		Hooks{AfterExecute: func(_ context.Context, _ ExecutionResult, _ error) { calls++ }},
		Hooks{AfterExecute: func(_ context.Context, _ ExecutionResult, _ error) { calls++ }},
	)
	h.AfterExecute(context.Background(), ExecutionResult{}, nil)
	assert.Equal(t, 2, calls)
}

func TestComposeHooksOnTimeout(t *testing.T) {
	var calls int

	h := ComposeHooks(
		Hooks{OnTimeout: func(_ context.Context, _ string, _ SandboxConfig) { calls++ }},
		Hooks{OnTimeout: func(_ context.Context, _ string, _ SandboxConfig) { calls++ }},
	)
	h.OnTimeout(context.Background(), "code", SandboxConfig{})
	assert.Equal(t, 2, calls)
}

func TestComposeHooksOnError(t *testing.T) {
	original := errors.New("original")
	replaced := errors.New("replaced")

	h := ComposeHooks(
		Hooks{OnError: func(_ context.Context, _ error) error { return replaced }},
	)
	err := h.OnError(context.Background(), original)
	assert.ErrorIs(t, err, replaced)
}

func TestComposeHooksNilFields(t *testing.T) {
	// Should not panic with empty hooks.
	h := ComposeHooks(Hooks{}, Hooks{})
	assert.NotNil(t, h.BeforeExecute)
	assert.NotNil(t, h.AfterExecute)
	assert.NotNil(t, h.OnTimeout)
	assert.NotNil(t, h.OnError)

	// Calling them should be safe.
	err := h.BeforeExecute(context.Background(), "", SandboxConfig{})
	assert.NoError(t, err)
	h.AfterExecute(context.Background(), ExecutionResult{}, nil)
	h.OnTimeout(context.Background(), "", SandboxConfig{})
	err = h.OnError(context.Background(), errors.New("test"))
	assert.Error(t, err) // passthrough returns original.
}

// --- LimitedWriter Tests ---

func TestLimitedWriter(t *testing.T) {
	tests := []struct {
		name    string
		max     int
		writes  []string
		want    string
		wantLen int
	}{
		{
			name:    "within limit",
			max:     100,
			writes:  []string{"hello"},
			want:    "hello",
			wantLen: 5,
		},
		{
			name:    "exactly at limit",
			max:     5,
			writes:  []string{"hello"},
			want:    "hello",
			wantLen: 5,
		},
		{
			name:    "exceeds limit",
			max:     3,
			writes:  []string{"hello"},
			want:    "hel",
			wantLen: 3,
		},
		{
			name:    "multiple writes exceed limit",
			max:     5,
			writes:  []string{"hel", "lo world"},
			want:    "hello",
			wantLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &limitedWriter{buf: &buf, max: tt.max}
			for _, s := range tt.writes {
				_, err := w.Write([]byte(s))
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, buf.String())
			assert.Equal(t, tt.wantLen, buf.Len())
		})
	}
}
