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
		OnIdle: func(ctx context.Context, state SessionState) {
			for _, h := range hooks {
				if h.OnIdle != nil {
					h.OnIdle(ctx, state)
				}
			}
		},
		OnWake: func(ctx context.Context, state SessionState) {
			for _, h := range hooks {
				if h.OnWake != nil {
					h.OnWake(ctx, state)
				}
			}
		},
		BeforeTask: func(ctx context.Context, taskName string, state SessionState) error {
			for _, h := range hooks {
				if h.BeforeTask != nil {
					if err := h.BeforeTask(ctx, taskName, state); err != nil {
						return err
					}
				}
			}
			return nil
		},
		AfterTask: func(ctx context.Context, result TaskResult) {
			for _, h := range hooks {
				if h.AfterTask != nil {
					h.AfterTask(ctx, result)
				}
			}
		},
	}
}
