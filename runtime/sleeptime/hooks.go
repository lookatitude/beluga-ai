package sleeptime

import "context"

// Hooks provides optional callback functions that observe and augment
// scheduler behavior. All fields are optional; nil fields are skipped.
type Hooks struct {
	// OnIdle is called when the scheduler detects that the session has become
	// idle and background work may begin.
	OnIdle func(ctx context.Context, state SessionState)

	// OnWake is called when the scheduler detects that the session is no
	// longer idle and background work should be preempted.
	OnWake func(ctx context.Context, state SessionState)

	// BeforeTask is called before a task begins execution. It may modify the
	// session state or return an error to skip the task.
	BeforeTask func(ctx context.Context, taskName string, state SessionState) error

	// AfterTask is called after a task completes, regardless of success or
	// failure.
	AfterTask func(ctx context.Context, result TaskResult)
}

// ComposeHooks merges multiple Hooks into a single Hooks value. Each hook
// function in the result calls the corresponding function from each input
// Hooks in order. For BeforeTask, execution stops at the first error.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		OnIdle:     composeOnIdle(hooks),
		OnWake:     composeOnWake(hooks),
		BeforeTask: composeBeforeTask(hooks),
		AfterTask:  composeAfterTask(hooks),
	}
}

// composeOnIdle returns an OnIdle callback that invokes every non-nil OnIdle
// in hooks, in order.
func composeOnIdle(hooks []Hooks) func(ctx context.Context, state SessionState) {
	return func(ctx context.Context, state SessionState) {
		for _, h := range hooks {
			if h.OnIdle != nil {
				h.OnIdle(ctx, state)
			}
		}
	}
}

// composeOnWake returns an OnWake callback that invokes every non-nil OnWake
// in hooks, in order.
func composeOnWake(hooks []Hooks) func(ctx context.Context, state SessionState) {
	return func(ctx context.Context, state SessionState) {
		for _, h := range hooks {
			if h.OnWake != nil {
				h.OnWake(ctx, state)
			}
		}
	}
}

// composeBeforeTask returns a BeforeTask callback that invokes every non-nil
// BeforeTask in hooks, in order, stopping at the first error.
func composeBeforeTask(hooks []Hooks) func(ctx context.Context, taskName string, state SessionState) error {
	return func(ctx context.Context, taskName string, state SessionState) error {
		for _, h := range hooks {
			if h.BeforeTask == nil {
				continue
			}
			if err := h.BeforeTask(ctx, taskName, state); err != nil {
				return err
			}
		}
		return nil
	}
}

// composeAfterTask returns an AfterTask callback that invokes every non-nil
// AfterTask in hooks, in order.
func composeAfterTask(hooks []Hooks) func(ctx context.Context, result TaskResult) {
	return func(ctx context.Context, result TaskResult) {
		for _, h := range hooks {
			if h.AfterTask != nil {
				h.AfterTask(ctx, result)
			}
		}
	}
}
