package sleeptime

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// alwaysIdleDetector always reports idle.
type alwaysIdleDetector struct{}

func (d *alwaysIdleDetector) IsIdle() bool { return true }
func (d *alwaysIdleDetector) OnActivity()  {}

// neverIdleDetector never reports idle.
type neverIdleDetector struct{}

func (d *neverIdleDetector) IsIdle() bool { return false }
func (d *neverIdleDetector) OnActivity()  {}

// toggleDetector can be switched between idle and active.
type toggleDetector struct {
	mu   sync.Mutex
	idle bool
}

func (d *toggleDetector) IsIdle() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.idle
}

func (d *toggleDetector) OnActivity() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.idle = false
}

func (d *toggleDetector) SetIdle(v bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.idle = v
}

// countingTask counts how many times Run is called.
type countingTask struct {
	name      string
	priority  Priority
	shouldRun bool
	count     atomic.Int32
	delay     time.Duration
}

var _ Task = (*countingTask)(nil)

func (t *countingTask) Name() string       { return t.name }
func (t *countingTask) Priority() Priority { return t.priority }
func (t *countingTask) ShouldRun(_ context.Context, _ SessionState) bool {
	return t.shouldRun
}

func (t *countingTask) Run(ctx context.Context, _ SessionState) (TaskResult, error) {
	t.count.Add(1)
	if t.delay > 0 {
		select {
		case <-time.After(t.delay):
		case <-ctx.Done():
			return TaskResult{Success: false}, ctx.Err()
		}
	}
	return TaskResult{Success: true, ItemsProcessed: 1}, nil
}

func TestScheduler_StartStop(t *testing.T) {
	s := NewScheduler(&alwaysIdleDetector{}, WithPollInterval(50*time.Millisecond))

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	health := s.Health()
	if health.Status != "healthy" {
		t.Errorf("Health().Status = %q, want %q", health.Status, "healthy")
	}

	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	health = s.Health()
	if health.Status != "degraded" {
		t.Errorf("Health().Status = %q after stop, want %q", health.Status, "degraded")
	}
}

func TestScheduler_DoubleStart(t *testing.T) {
	s := NewScheduler(&alwaysIdleDetector{}, WithPollInterval(50*time.Millisecond))
	ctx := context.Background()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_ = s.Stop(stopCtx)
	}()

	// Second start should be a no-op.
	if err := s.Start(ctx); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
}

func TestScheduler_RunsTasks(t *testing.T) {
	task := &countingTask{name: "counter", priority: PriorityNormal, shouldRun: true}

	s := NewScheduler(&alwaysIdleDetector{},
		WithTasks(task),
		WithPollInterval(50*time.Millisecond),
		WithMaxConcurrentTasks(1),
	)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for at least one tick to fire and task to execute.
	time.Sleep(200 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if task.count.Load() == 0 {
		t.Error("expected task to run at least once")
	}
}

func TestScheduler_SkipsTaskWhenShouldRunFalse(t *testing.T) {
	task := &countingTask{name: "skip", priority: PriorityNormal, shouldRun: false}

	s := NewScheduler(&alwaysIdleDetector{},
		WithTasks(task),
		WithPollInterval(50*time.Millisecond),
	)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_ = s.Stop(stopCtx)

	if task.count.Load() != 0 {
		t.Errorf("expected task not to run, but ran %d times", task.count.Load())
	}
}

func TestScheduler_Hooks(t *testing.T) {
	var (
		mu          sync.Mutex
		idleCalled  bool
		wakeCalled  bool
		beforeCalls []string
		afterCalls  []string
	)

	hooks := Hooks{
		OnIdle: func(_ context.Context, _ SessionState) {
			mu.Lock()
			idleCalled = true
			mu.Unlock()
		},
		OnWake: func(_ context.Context, _ SessionState) {
			mu.Lock()
			wakeCalled = true
			mu.Unlock()
		},
		BeforeTask: func(_ context.Context, name string, _ SessionState) error {
			mu.Lock()
			beforeCalls = append(beforeCalls, name)
			mu.Unlock()
			return nil
		},
		AfterTask: func(_ context.Context, r TaskResult) {
			mu.Lock()
			afterCalls = append(afterCalls, r.TaskName)
			mu.Unlock()
		},
	}

	task := &countingTask{name: "hooked", priority: PriorityNormal, shouldRun: true}
	det := &toggleDetector{idle: true}

	s := NewScheduler(det,
		WithTasks(task),
		WithPollInterval(50*time.Millisecond),
		WithHooks(hooks),
	)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for idle detection and task execution.
	time.Sleep(200 * time.Millisecond)

	// Wake the scheduler.
	s.Wake(ctx)

	time.Sleep(50 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_ = s.Stop(stopCtx)

	mu.Lock()
	defer mu.Unlock()

	if !idleCalled {
		t.Error("expected OnIdle hook to be called")
	}
	if !wakeCalled {
		t.Error("expected OnWake hook to be called")
	}
	if len(beforeCalls) == 0 {
		t.Error("expected BeforeTask hook to be called")
	}
	if len(afterCalls) == 0 {
		t.Error("expected AfterTask hook to be called")
	}
}

func TestScheduler_PreemptOnWake(t *testing.T) {
	det := &toggleDetector{idle: true}
	task := &countingTask{
		name:      "slow",
		priority:  PriorityNormal,
		shouldRun: true,
		delay:     2 * time.Second,
	}

	s := NewScheduler(det,
		WithTasks(task),
		WithPollInterval(50*time.Millisecond),
		WithMaxConcurrentTasks(1),
	)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for task to start.
	time.Sleep(150 * time.Millisecond)

	// Simulate user returning — wake the scheduler.
	det.SetIdle(false)

	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestScheduler_SetSessionState(t *testing.T) {
	s := NewScheduler(&alwaysIdleDetector{})

	state := SessionState{
		SessionID: "sess-123",
		AgentID:   "agent-1",
		TurnCount: 10,
	}
	s.SetSessionState(state)

	s.mu.Lock()
	got := s.state
	s.mu.Unlock()

	if got.SessionID != "sess-123" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "sess-123")
	}
	if got.TurnCount != 10 {
		t.Errorf("TurnCount = %d, want %d", got.TurnCount, 10)
	}
}

func TestScheduler_ContextCancellation(t *testing.T) {
	s := NewScheduler(&alwaysIdleDetector{}, WithPollInterval(50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Cancel context should stop the loop.
	cancel()
	time.Sleep(200 * time.Millisecond)

	// Stop should complete quickly since loop already exited.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	if err := s.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestScheduler_Options(t *testing.T) {
	t.Run("WithMaxConcurrentTasks clamps to 1", func(t *testing.T) {
		s := NewScheduler(&alwaysIdleDetector{}, WithMaxConcurrentTasks(0))
		if s.opts.maxConcurrentTasks != 1 {
			t.Errorf("maxConcurrentTasks = %d, want 1", s.opts.maxConcurrentTasks)
		}
	})

	t.Run("WithPollInterval clamps to 10ms", func(t *testing.T) {
		s := NewScheduler(&alwaysIdleDetector{}, WithPollInterval(1*time.Millisecond))
		if s.opts.pollInterval != 10*time.Millisecond {
			t.Errorf("pollInterval = %v, want 10ms", s.opts.pollInterval)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		s := NewScheduler(&alwaysIdleDetector{})
		if s.opts.maxConcurrentTasks != defaultMaxConcurrentTasks {
			t.Errorf("maxConcurrentTasks = %d, want %d", s.opts.maxConcurrentTasks, defaultMaxConcurrentTasks)
		}
		if s.opts.pollInterval != defaultPollInterval {
			t.Errorf("pollInterval = %v, want %v", s.opts.pollInterval, defaultPollInterval)
		}
	})
}

func TestScheduler_TaskPriorityOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string

	low := &countingTask{name: "low", priority: PriorityLow, shouldRun: true}
	high := &countingTask{name: "high", priority: PriorityHigh, shouldRun: true}
	normal := &countingTask{name: "normal", priority: PriorityNormal, shouldRun: true}

	hooks := Hooks{
		BeforeTask: func(_ context.Context, name string, _ SessionState) error {
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			return nil
		},
	}

	s := NewScheduler(&alwaysIdleDetector{},
		WithTasks(low, high, normal),
		WithPollInterval(50*time.Millisecond),
		WithMaxConcurrentTasks(1), // sequential to preserve order
		WithHooks(hooks),
	)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_ = s.Stop(stopCtx)

	mu.Lock()
	defer mu.Unlock()

	// Verify that at least one tick ran all three and in priority order.
	if len(order) >= 3 {
		// First three should be in priority order.
		if order[0] != "high" || order[1] != "normal" || order[2] != "low" {
			t.Errorf("task order = %v, want [high, normal, low, ...]", order)
		}
	}
}
