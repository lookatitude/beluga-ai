package sleeptime

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// Default scheduler configuration values.
const (
	defaultPollInterval       = 5 * time.Second
	defaultMaxConcurrentTasks = 2
	minPollInterval           = 10 * time.Millisecond
)

// schedulerOptions holds the configuration for a Scheduler.
type schedulerOptions struct {
	tasks              []Task
	maxConcurrentTasks int
	pollInterval       time.Duration
	hooks              Hooks
}

// Option configures a Scheduler.
type Option func(*schedulerOptions)

// WithTasks sets the tasks that the scheduler may run during idle periods.
func WithTasks(tasks ...Task) Option {
	return func(o *schedulerOptions) {
		o.tasks = append(o.tasks, tasks...)
	}
}

// WithMaxConcurrentTasks sets the maximum number of tasks that may run
// concurrently during an idle period. The minimum is 1.
func WithMaxConcurrentTasks(n int) Option {
	return func(o *schedulerOptions) {
		if n < 1 {
			n = 1
		}
		o.maxConcurrentTasks = n
	}
}

// WithPollInterval sets how frequently the scheduler checks the idle
// detector. The minimum is 10ms.
func WithPollInterval(d time.Duration) Option {
	return func(o *schedulerOptions) {
		if d < minPollInterval {
			d = minPollInterval
		}
		o.pollInterval = d
	}
}

// WithHooks sets the hooks for observing scheduler events.
func WithHooks(h Hooks) Option {
	return func(o *schedulerOptions) {
		o.hooks = h
	}
}

// Scheduler orchestrates background task execution during idle periods.
// It implements core.Lifecycle for integration with the application
// lifecycle manager.
type Scheduler struct {
	detector IdleDetector
	opts     schedulerOptions

	mu      sync.Mutex
	state   SessionState
	running bool
	cancel  context.CancelFunc
	done    chan struct{}
	wasIdle bool
}

// Compile-time check that Scheduler implements core.Lifecycle.
var _ core.Lifecycle = (*Scheduler)(nil)

// NewScheduler creates a Scheduler that uses the given IdleDetector to
// determine when to run background tasks.
func NewScheduler(detector IdleDetector, opts ...Option) *Scheduler {
	o := schedulerOptions{
		maxConcurrentTasks: defaultMaxConcurrentTasks,
		pollInterval:       defaultPollInterval,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &Scheduler{
		detector: detector,
		opts:     o,
		done:     make(chan struct{}),
	}
}

// SetSessionState updates the session state that tasks will receive. This
// is called by the plugin after each turn to keep the scheduler informed.
func (s *Scheduler) SetSessionState(state SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

// Wake signals that user activity has occurred. It notifies the idle
// detector and fires the OnWake hook if the scheduler was in idle mode.
func (s *Scheduler) Wake(ctx context.Context) {
	s.detector.OnActivity()
	s.mu.Lock()
	wasIdle := s.wasIdle
	state := s.state
	s.wasIdle = false
	s.mu.Unlock()

	if wasIdle && s.opts.hooks.OnWake != nil {
		s.opts.hooks.OnWake(ctx, state)
	}
}

// Start begins the scheduler's polling loop. It blocks until start
// completes (immediately) and returns nil.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.done = make(chan struct{})
	loopCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()

	go s.loop(loopCtx)
	return nil
}

// Stop gracefully shuts down the scheduler, cancelling any in-flight tasks
// and waiting for the polling loop to exit.
func (s *Scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()

	cancel()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Health returns the current health status of the scheduler.
func (s *Scheduler) Health() core.HealthStatus {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	if running {
		return core.HealthStatus{
			Status:    core.HealthHealthy,
			Message:   "scheduler is running",
			Timestamp: time.Now(),
		}
	}
	return core.HealthStatus{
		Status:    core.HealthDegraded,
		Message:   "scheduler is not running",
		Timestamp: time.Now(),
	}
}

// loop is the main polling loop that checks for idle state and runs tasks.
func (s *Scheduler) loop(ctx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.opts.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

// tick performs a single check of the idle detector and runs tasks if idle.
func (s *Scheduler) tick(ctx context.Context) {
	if !s.detector.IsIdle() {
		return
	}

	s.mu.Lock()
	if !s.wasIdle {
		s.wasIdle = true
		state := s.state
		s.mu.Unlock()
		if s.opts.hooks.OnIdle != nil {
			s.opts.hooks.OnIdle(ctx, state)
		}
	} else {
		s.mu.Unlock()
	}

	s.runTasks(ctx)
}

// runTasks executes eligible tasks sorted by priority with bounded concurrency.
// Tasks are preempted if the detector reports the session is no longer idle.
func (s *Scheduler) runTasks(ctx context.Context) {
	s.mu.Lock()
	state := s.state
	s.mu.Unlock()

	// Sort tasks by priority (lower value = higher priority).
	eligible := s.eligibleTasks(ctx, state)
	if len(eligible) == 0 {
		return
	}

	// Create a cancellable context for preemption.
	taskCtx, taskCancel := context.WithCancel(ctx)
	defer taskCancel()

	// Semaphore for bounded concurrency.
	sem := make(chan struct{}, s.opts.maxConcurrentTasks)
	var wg sync.WaitGroup

loop:
	for _, task := range eligible {
		// Check if still idle before starting each task.
		if !s.detector.IsIdle() {
			taskCancel()
			break loop
		}

		// Check context cancellation.
		if ctx.Err() != nil {
			break loop
		}

		// Acquire semaphore slot.
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break loop
		}

		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			defer func() { <-sem }()
			s.executeTask(taskCtx, t, state)
		}(task)
	}

	wg.Wait()
}

// eligibleTasks returns tasks that should run, sorted by priority.
func (s *Scheduler) eligibleTasks(ctx context.Context, state SessionState) []Task {
	var eligible []Task
	for _, t := range s.opts.tasks {
		if t.ShouldRun(ctx, state) {
			eligible = append(eligible, t)
		}
	}
	sort.Slice(eligible, func(i, j int) bool {
		return eligible[i].Priority() < eligible[j].Priority()
	})
	return eligible
}

// executeTask runs a single task with hooks and error handling.
func (s *Scheduler) executeTask(ctx context.Context, t Task, state SessionState) {
	taskName := t.Name()

	// BeforeTask hook.
	if s.opts.hooks.BeforeTask != nil {
		if err := s.opts.hooks.BeforeTask(ctx, taskName, state); err != nil {
			slog.WarnContext(ctx, "sleeptime: BeforeTask hook rejected task",
				"task", taskName,
				"error", err,
			)
			return
		}
	}

	start := time.Now()
	result, err := t.Run(ctx, state)
	result.TaskName = taskName
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		slog.WarnContext(ctx, "sleeptime: task failed",
			"task", taskName,
			"duration", result.Duration,
			"error", err,
		)
	}

	// AfterTask hook.
	if s.opts.hooks.AfterTask != nil {
		s.opts.hooks.AfterTask(ctx, result)
	}
}
