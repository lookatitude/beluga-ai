package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// --- test helpers -----------------------------------------------------------

// sleepTool sleeps for a fixed duration then returns its name, or respects
// context cancellation.
type sleepTool struct {
	name     string
	duration time.Duration
}

func (s *sleepTool) Name() string                { return s.name }
func (s *sleepTool) Description() string         { return "sleep tool" }
func (s *sleepTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (s *sleepTool) Execute(ctx context.Context, _ map[string]any) (*tool.Result, error) {
	select {
	case <-time.After(s.duration):
		return tool.TextResult(s.name), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// echoTool echoes the "value" argument back as a text result.
type echoTool struct{}

func (e *echoTool) Name() string                { return "echo" }
func (e *echoTool) Description() string         { return "echo" }
func (e *echoTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (e *echoTool) Execute(_ context.Context, args map[string]any) (*tool.Result, error) {
	v, _ := args["value"].(string)
	return tool.TextResult(v), nil
}

// errorTool always returns an error.
type errorTool struct{ msg string }

func (f *errorTool) Name() string                { return "errortool" }
func (f *errorTool) Description() string         { return "always fails" }
func (f *errorTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (f *errorTool) Execute(_ context.Context, _ map[string]any) (*tool.Result, error) {
	return nil, fmt.Errorf("%s", f.msg)
}

// counterTool counts concurrent executions and tracks the peak.
type counterTool struct {
	name    string
	dur     time.Duration
	current atomic.Int64
	peak    atomic.Int64
}

func (c *counterTool) Name() string                { return c.name }
func (c *counterTool) Description() string         { return "counter" }
func (c *counterTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (c *counterTool) Execute(ctx context.Context, _ map[string]any) (*tool.Result, error) {
	cur := c.current.Add(1)
	for {
		p := c.peak.Load()
		if cur <= p || c.peak.CompareAndSwap(p, cur) {
			break
		}
	}
	select {
	case <-time.After(c.dur):
	case <-ctx.Done():
	}
	c.current.Add(-1)
	return tool.TextResult(c.name), nil
}

// orderedTool records its execution order in a shared slice.
type orderedTool struct {
	name  string
	order *[]string
	mu    *sync.Mutex
}

func (o *orderedTool) Name() string                { return o.name }
func (o *orderedTool) Description() string         { return "ordered" }
func (o *orderedTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (o *orderedTool) Execute(_ context.Context, _ map[string]any) (*tool.Result, error) {
	o.mu.Lock()
	*o.order = append(*o.order, o.name)
	o.mu.Unlock()
	return tool.TextResult(o.name), nil
}

// newRegistryWith builds a *tool.Registry pre-loaded with the given tools.
func newRegistryWith(tools ...tool.Tool) *tool.Registry {
	r := tool.NewRegistry()
	for _, t := range tools {
		_ = r.Add(t)
	}
	return r
}

// textContent extracts text from the first TextPart in a result.
func textContent(r *tool.Result) string {
	if r == nil {
		return ""
	}
	for _, c := range r.Content {
		if tp, ok := c.(schema.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}

// --- unit tests -------------------------------------------------------------

func TestToolDAGExecutor_EmptyInput(t *testing.T) {
	e := NewToolDAGExecutor()
	results := e.Execute(context.Background(), nil, tool.NewRegistry())
	if results != nil {
		t.Errorf("expected nil results for empty input, got %v", results)
	}
}

func TestToolDAGExecutor_SingleCall(t *testing.T) {
	reg := newRegistryWith(&echoTool{})
	args, _ := json.Marshal(map[string]any{"value": "hello"})

	e := NewToolDAGExecutor()
	results := e.Execute(context.Background(), []schema.ToolCall{
		{ID: "c1", Name: "echo", Arguments: string(args)},
	}, reg)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if textContent(results[0]) != "hello" {
		t.Errorf("expected 'hello', got %q", textContent(results[0]))
	}
}

func TestToolDAGExecutor_ToolNotFound(t *testing.T) {
	e := NewToolDAGExecutor()
	results := e.Execute(context.Background(), []schema.ToolCall{
		{ID: "c1", Name: "nonexistent"},
	}, tool.NewRegistry())

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].IsError {
		t.Error("expected IsError=true for missing tool")
	}
}

func TestToolDAGExecutor_ToolExecutionError(t *testing.T) {
	reg := newRegistryWith(&errorTool{msg: "boom"})
	e := NewToolDAGExecutor()
	results := e.Execute(context.Background(), []schema.ToolCall{
		{ID: "c1", Name: "errortool"},
	}, reg)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].IsError {
		t.Error("expected IsError=true when tool returns an error")
	}
}

func TestToolDAGExecutor_InvalidJSON(t *testing.T) {
	reg := newRegistryWith(&echoTool{})
	e := NewToolDAGExecutor()
	results := e.Execute(context.Background(), []schema.ToolCall{
		{ID: "c1", Name: "echo", Arguments: "{invalid json"},
	}, reg)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].IsError {
		t.Error("expected IsError=true for invalid JSON arguments")
	}
}

func TestToolDAGExecutor_ParallelExecution(t *testing.T) {
	// Two 100 ms tools run in parallel, total should be ~100 ms, not ~200 ms.
	reg := newRegistryWith(
		&sleepTool{name: "t1", duration: 100 * time.Millisecond},
		&sleepTool{name: "t2", duration: 100 * time.Millisecond},
	)

	e := NewToolDAGExecutor(WithMaxConcurrency(2))

	start := time.Now()
	results := e.Execute(context.Background(), []schema.ToolCall{
		{ID: "c1", Name: "t1"},
		{ID: "c2", Name: "t2"},
	}, reg)
	elapsed := time.Since(start)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if r.IsError {
			t.Errorf("results[%d] unexpected error: %s", i, textContent(r))
		}
	}
	if elapsed > 180*time.Millisecond {
		t.Errorf("parallel execution too slow: %v (expected < 180ms)", elapsed)
	}
}

func TestToolDAGExecutor_MaxConcurrencyRespected(t *testing.T) {
	ct := &counterTool{name: "ct", dur: 50 * time.Millisecond}
	reg := newRegistryWith(ct)

	const maxConc = 2
	calls := make([]schema.ToolCall, 6)
	for i := range calls {
		calls[i] = schema.ToolCall{ID: fmt.Sprintf("c%d", i), Name: "ct"}
	}

	e := NewToolDAGExecutor(WithMaxConcurrency(maxConc))
	e.Execute(context.Background(), calls, reg)

	if peak := ct.peak.Load(); peak > int64(maxConc) {
		t.Errorf("peak concurrency %d exceeded max %d", peak, maxConc)
	}
}

func TestToolDAGExecutor_ContextCancellation(t *testing.T) {
	reg := newRegistryWith(&sleepTool{name: "slow", duration: 10 * time.Second})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	calls := []schema.ToolCall{
		{ID: "c1", Name: "slow"},
		{ID: "c2", Name: "slow"},
	}

	e := NewToolDAGExecutor(WithMaxConcurrency(4))
	start := time.Now()
	results := e.Execute(ctx, calls, reg)
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("cancellation did not stop execution promptly: %v", elapsed)
	}
	if len(results) != len(calls) {
		t.Errorf("expected %d results, got %d", len(calls), len(results))
	}
	for i, r := range results {
		if r == nil {
			t.Errorf("results[%d] is nil after context cancellation", i)
		}
	}
}

func TestToolDAGExecutor_ResultOrder(t *testing.T) {
	// Tools with different sleep durations — results must match input order.
	reg := newRegistryWith(
		&sleepTool{name: "slow", duration: 80 * time.Millisecond},
		&sleepTool{name: "medium", duration: 40 * time.Millisecond},
		&sleepTool{name: "fast", duration: 10 * time.Millisecond},
	)

	e := NewToolDAGExecutor(WithMaxConcurrency(3))
	results := e.Execute(context.Background(), []schema.ToolCall{
		{ID: "c1", Name: "slow"},
		{ID: "c2", Name: "medium"},
		{ID: "c3", Name: "fast"},
	}, reg)

	want := []string{"slow", "medium", "fast"}
	for i, w := range want {
		if got := textContent(results[i]); got != w {
			t.Errorf("results[%d]: want %q, got %q", i, w, got)
		}
	}
}

// --- dependency detection unit tests ----------------------------------------

func TestDetectDependencies_NoDeps(t *testing.T) {
	idToIdx := map[string]int{"call-1": 0, "call-2": 1}
	args, _ := json.Marshal(map[string]any{"x": 42, "y": "hello"})
	deps := detectDependencies(string(args), idToIdx, 2)
	if len(deps) != 0 {
		t.Errorf("expected no deps, got %v", deps)
	}
}

func TestDetectDependencies_WithDep(t *testing.T) {
	idToIdx := map[string]int{"call-1": 0, "call-2": 1}
	args, _ := json.Marshal(map[string]any{"input": "call-1"})
	deps := detectDependencies(string(args), idToIdx, 1)
	if len(deps) != 1 || deps[0] != 0 {
		t.Errorf("expected dep on index 0, got %v", deps)
	}
}

func TestDetectDependencies_SelfReferenceIgnored(t *testing.T) {
	idToIdx := map[string]int{"call-1": 0}
	args, _ := json.Marshal(map[string]any{"input": "call-1"})
	deps := detectDependencies(string(args), idToIdx, 0)
	if len(deps) != 0 {
		t.Errorf("expected self-reference to be ignored, got %v", deps)
	}
}

func TestDetectDependencies_InvalidJSON(t *testing.T) {
	idToIdx := map[string]int{"x": 0}
	deps := detectDependencies("{bad json", idToIdx, 1)
	if deps != nil {
		t.Errorf("expected nil deps for invalid JSON, got %v", deps)
	}
}

func TestDetectDependencies_EmptyArgs(t *testing.T) {
	deps := detectDependencies("", map[string]int{"x": 0}, 1)
	if deps != nil {
		t.Errorf("expected nil for empty args, got %v", deps)
	}
}

func TestDetectDependencies_NestedDep(t *testing.T) {
	idToIdx := map[string]int{"call-1": 0}
	args, _ := json.Marshal(map[string]any{"nested": map[string]any{"ref": "call-1"}})
	deps := detectDependencies(string(args), idToIdx, 1)
	if len(deps) != 1 || deps[0] != 0 {
		t.Errorf("expected dep on index 0 from nested arg, got %v", deps)
	}
}

// --- DAG ordering integration test ------------------------------------------

func TestToolDAGExecutor_DAGOrdering(t *testing.T) {
	// call-2 references call-1's ID → must execute after call-1.
	var (
		order []string
		mu    sync.Mutex
	)

	reg := newRegistryWith(
		&orderedTool{name: "t1", order: &order, mu: &mu},
		&orderedTool{name: "t2", order: &order, mu: &mu},
	)

	args2, _ := json.Marshal(map[string]any{"dep": "call-1"})
	calls := []schema.ToolCall{
		{ID: "call-1", Name: "t1"},
		{ID: "call-2", Name: "t2", Arguments: string(args2)},
	}

	e := NewToolDAGExecutor(WithDependencyDetection(true), WithMaxConcurrency(4))
	results := e.Execute(context.Background(), calls, reg)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if r.IsError {
			t.Errorf("results[%d] unexpected error: %s", i, textContent(r))
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "t1" || order[1] != "t2" {
		t.Errorf("expected execution order [t1, t2], got %v", order)
	}
}

// --- functional option tests ------------------------------------------------

func TestNewToolDAGExecutor_Defaults(t *testing.T) {
	e := NewToolDAGExecutor()
	if e.maxConcurrency != 8 {
		t.Errorf("default maxConcurrency: want 8, got %d", e.maxConcurrency)
	}
	if e.dependencyDetection {
		t.Error("default dependencyDetection should be false")
	}
}

func TestWithMaxConcurrency_Zero(t *testing.T) {
	// Zero should be ignored; default preserved.
	e := NewToolDAGExecutor(WithMaxConcurrency(0))
	if e.maxConcurrency != 8 {
		t.Errorf("zero maxConcurrency should be ignored, got %d", e.maxConcurrency)
	}
}

func TestWithMaxConcurrency_Positive(t *testing.T) {
	e := NewToolDAGExecutor(WithMaxConcurrency(3))
	if e.maxConcurrency != 3 {
		t.Errorf("want 3, got %d", e.maxConcurrency)
	}
}

func TestWithDependencyDetection_True(t *testing.T) {
	e := NewToolDAGExecutor(WithDependencyDetection(true))
	if !e.dependencyDetection {
		t.Error("dependencyDetection should be true")
	}
}

func TestWithDependencyDetection_False(t *testing.T) {
	e := NewToolDAGExecutor(WithDependencyDetection(true), WithDependencyDetection(false))
	if e.dependencyDetection {
		t.Error("dependencyDetection should be false after second option")
	}
}

// --- benchmarks -------------------------------------------------------------

func BenchmarkToolDAGExecutor_Parallel(b *testing.B) {
	reg := newRegistryWith(&echoTool{})
	args, _ := json.Marshal(map[string]any{"value": "bench"})
	calls := make([]schema.ToolCall, 16)
	for i := range calls {
		calls[i] = schema.ToolCall{
			ID:        fmt.Sprintf("c%d", i),
			Name:      "echo",
			Arguments: string(args),
		}
	}

	e := NewToolDAGExecutor(WithMaxConcurrency(8))
	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		e.Execute(context.Background(), calls, reg)
	}
}

func BenchmarkToolDAGExecutor_DAG(b *testing.B) {
	reg := newRegistryWith(&echoTool{})
	args, _ := json.Marshal(map[string]any{"value": "bench"})
	// Linear dependency chain.
	calls := make([]schema.ToolCall, 8)
	for i := range calls {
		var argStr string
		if i == 0 {
			argStr = string(args)
		} else {
			depArgs, _ := json.Marshal(map[string]any{"dep": fmt.Sprintf("c%d", i-1)})
			argStr = string(depArgs)
		}
		calls[i] = schema.ToolCall{
			ID:        fmt.Sprintf("c%d", i),
			Name:      "echo",
			Arguments: argStr,
		}
	}

	e := NewToolDAGExecutor(WithDependencyDetection(true), WithMaxConcurrency(4))
	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		e.Execute(context.Background(), calls, reg)
	}
}
