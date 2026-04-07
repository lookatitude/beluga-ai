package runtime

import (
	"context"
	"errors"
	"iter"
	"log/slog"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

// Runner is the lifecycle manager for a single agent.
// It handles session management, plugin execution, guard enforcement,
// event streaming, and graceful shutdown.
//
// Create a Runner with [NewRunner] and configure it with functional options.
// Call [Runner.Run] to execute the agent for a given session and input.
// Call [Runner.Shutdown] to gracefully drain in-flight sessions.
type Runner struct {
	agent          agent.Agent
	sessionService SessionService
	pluginChain    *PluginChain
	workerPool     *WorkerPool
	config         RunnerConfig

	mu       sync.Mutex
	shutdown bool
}

// RunnerOption is a functional option for configuring a Runner.
type RunnerOption func(*Runner)

// WithPlugins adds plugins to the Runner's plugin chain.
// Plugins execute in the order provided: first plugin's BeforeTurn runs first,
// and its AfterTurn runs first after the agent completes.
func WithPlugins(plugins ...Plugin) RunnerOption {
	return func(r *Runner) {
		r.pluginChain = NewPluginChain(plugins...)
	}
}

// WithSessionService sets the session service used by the Runner to
// create and retrieve sessions. If not provided, the Runner uses an
// InMemorySessionService.
func WithSessionService(s SessionService) RunnerOption {
	return func(r *Runner) {
		r.sessionService = s
	}
}

// WithRunnerConfig sets the full RunnerConfig for the Runner.
// Individual options like WithWorkerPoolSize override the corresponding
// field in the config.
func WithRunnerConfig(cfg RunnerConfig) RunnerOption {
	return func(r *Runner) {
		r.config = cfg
	}
}

// WithWorkerPoolSize sets the number of concurrent workers in the Runner's
// pool. A value less than 1 is normalized to 1. This overrides the
// WorkerPoolSize in RunnerConfig.
func WithWorkerPoolSize(size int) RunnerOption {
	return func(r *Runner) {
		r.config.WorkerPoolSize = size
	}
}

// NewRunner creates a new Runner hosting the given agent. The Runner is
// configured with sensible defaults that can be overridden with functional
// options. The agent must not be nil.
func NewRunner(a agent.Agent, opts ...RunnerOption) *Runner {
	r := &Runner{
		agent:  a,
		config: defaults(),
	}

	for _, opt := range opts {
		opt(r)
	}

	// Apply defaults for unset fields.
	if r.sessionService == nil {
		var sessionOpts []SessionOption
		if r.config.SessionTTL > 0 {
			sessionOpts = append(sessionOpts, WithSessionTTL(r.config.SessionTTL))
		}
		r.sessionService = NewInMemorySessionService(sessionOpts...)
	}

	if r.pluginChain == nil {
		r.pluginChain = NewPluginChain()
	}

	poolSize := r.config.WorkerPoolSize
	if poolSize < 1 {
		poolSize = defaultWorkerPoolSize
	}
	r.workerPool = NewWorkerPool(poolSize)

	return r
}

// Run executes the agent for the given session and input message, returning
// a stream of events. The flow is:
//
//  1. Load or create the session identified by sessionID.
//  2. Run the plugin chain's BeforeTurn hooks.
//  3. Stream the agent's response.
//  4. Collect events and run the plugin chain's AfterTurn hooks.
//  5. Yield the (potentially modified) events to the caller.
//
// The provided context controls the lifetime of the execution. Agents MUST
// respect context cancellation. If an agent ignores a cancelled context, the
// goroutine will complete when the agent's stream terminates naturally.
//
// Run respects context cancellation at every stage. If the Runner has been
// shut down, Run returns a single error event.
func (r *Runner) Run(ctx context.Context, sessionID string, input schema.Message) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		// Check if shut down.
		r.mu.Lock()
		isShutdown := r.shutdown
		r.mu.Unlock()
		if isShutdown {
			yield(agent.Event{}, core.NewError("runtime.runner.run", core.ErrInvalidInput,
				"runner has been shut down", nil))
			return
		}

		// Use context for cancellation throughout.
		if err := ctx.Err(); err != nil {
			yield(agent.Event{}, err)
			return
		}

		// Use a channel to synchronize worker pool execution with the iterator.
		type result struct {
			events []agent.Event
			err    error
		}
		resultCh := make(chan result, 1)

		// Submit work to the worker pool.
		submitErr := r.workerPool.Submit(ctx, func(ctx context.Context) {
			events, err := r.executeTurn(ctx, sessionID, input)
			resultCh <- result{events: events, err: err}
		})
		if submitErr != nil {
			yield(agent.Event{}, core.NewError("runtime.runner.run", core.ErrInvalidInput,
				"failed to submit work to pool", submitErr))
			return
		}

		// Wait for the result or context cancellation.
		select {
		case <-ctx.Done():
			yield(agent.Event{}, ctx.Err())
			return
		case res := <-resultCh:
			if res.err != nil {
				// Run error through plugin chain.
				pluginErr := r.pluginChain.RunOnError(ctx, res.err)
				yield(agent.Event{}, pluginErr)
				return
			}
			// Yield all events to the caller.
			for _, evt := range res.events {
				if !yield(evt, nil) {
					return
				}
			}
		}
	}
}

// executeTurn performs the full turn lifecycle: session load/create,
// BeforeTurn plugins, agent streaming, AfterTurn plugins, and session update.
func (r *Runner) executeTurn(ctx context.Context, sessionID string, input schema.Message) ([]agent.Event, error) {
	// Step 1: Load or create session.
	session, err := r.getOrCreateSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Step 2: Run BeforeTurn plugins.
	modifiedInput, err := r.pluginChain.RunBeforeTurn(ctx, session, input)
	if err != nil {
		return nil, core.NewError("runtime.runner.beforeTurn", core.ErrInvalidInput,
			"plugin BeforeTurn failed", err)
	}

	// Step 3: Stream agent events and collect them.
	inputText := extractMessageText(modifiedInput)
	var events []agent.Event
	for evt, streamErr := range r.agent.Stream(ctx, inputText) {
		if streamErr != nil {
			return nil, streamErr
		}
		events = append(events, evt)
	}

	// Step 4: Run AfterTurn plugins.
	events, err = r.pluginChain.RunAfterTurn(ctx, session, events)
	if err != nil {
		return nil, core.NewError("runtime.runner.afterTurn", core.ErrInvalidInput,
			"plugin AfterTurn failed", err)
	}

	// Step 5: Record the turn in the session and persist.
	outputMsg := buildOutputMessage(events)
	session.Turns = append(session.Turns, schema.Turn{
		Input:     modifiedInput,
		Output:    outputMsg,
		Timestamp: time.Now().UTC(),
	})
	if updateErr := r.sessionService.Update(ctx, session); updateErr != nil {
		// Log but do not fail the turn — the events were already produced.
		slog.WarnContext(ctx, "failed to update session", "error", updateErr)
	}

	return events, nil
}

// getOrCreateSession attempts to load an existing session. If the sessionID
// is empty or the session is not found, it creates a new one.
func (r *Runner) getOrCreateSession(ctx context.Context, sessionID string) (*Session, error) {
	if sessionID != "" {
		session, err := r.sessionService.Get(ctx, sessionID)
		if err == nil {
			return session, nil
		}
		// If the error is not a "not found" error, propagate it.
		var coreErr *core.Error
		if !errors.As(err, &coreErr) || coreErr.Code != core.ErrNotFound {
			return nil, err
		}
		// Fall through to create a new session.
	}

	session, err := r.sessionService.Create(ctx, r.agent.ID())
	if err != nil {
		return nil, core.NewError("runtime.runner.session", core.ErrInvalidInput,
			"failed to create session", err)
	}
	return session, nil
}

// Shutdown gracefully shuts down the Runner by draining the worker pool.
// It waits for all in-flight sessions to complete, respecting the provided
// context deadline. If no deadline is set on ctx, the Runner's configured
// GracefulShutdownTimeout is used.
func (r *Runner) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	r.shutdown = true
	r.mu.Unlock()

	// Apply default timeout if none on context.
	if _, ok := ctx.Deadline(); !ok && r.config.GracefulShutdownTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.GracefulShutdownTimeout)
		defer cancel()
	}

	return r.workerPool.Drain(ctx)
}

// extractMessageText extracts the text content from a schema.Message.
func extractMessageText(msg schema.Message) string {
	if msg == nil {
		return ""
	}
	parts := msg.GetContent()
	for _, p := range parts {
		if tp, ok := p.(schema.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}

// buildOutputMessage constructs an AIMessage from the collected agent events
// by concatenating all text events.
func buildOutputMessage(events []agent.Event) schema.Message {
	var text string
	for _, evt := range events {
		if evt.Type == agent.EventText {
			text += evt.Text
		}
	}
	return schema.NewAIMessage(text)
}
