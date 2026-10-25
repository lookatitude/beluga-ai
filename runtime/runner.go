package runtime

import (
	"context"
	"errors"
	"iter"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
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

	// sessionLocks serializes turns per session ID. Entries are reference
	// counted and removed when no turn holds or waits for them.
	locksMu      sync.Mutex
	sessionLocks map[string]*sessionLock
}

// sessionLock is a reference-counted per-session turn lock.
type sessionLock struct {
	mu   sync.Mutex
	refs int
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

// MetadataSessionID is the agent.Event Metadata key under which Runner.Run
// reports the ID of the session the turn ran in. When Run is called with an
// empty or unknown sessionID a new session is created, and this is how the
// caller learns its ID for the next turn.
const MetadataSessionID = "session_id"

// Run executes the agent for the given session and input message, returning
// a stream of events. The flow is:
//
//  1. Load or create the session identified by sessionID.
//  2. Run the plugin chain's BeforeTurn hooks.
//  3. Stream the agent's response. The agent receives the session's earlier
//     turns as history (agent.WithHistory).
//  4. Run the plugin chain's AfterTurn hooks.
//  5. Record the turn in the session.
//
// When the Runner has no plugins, events are yielded as the agent produces
// them. When plugins are configured, AfterTurn may rewrite the event list, so
// the turn is collected first and the (potentially modified) events are
// yielded afterwards.
//
// Every yielded event carries the session ID under [MetadataSessionID].
// Turns on the same session are serialized, so concurrent Run calls for one
// session do not lose each other's updates.
//
// The provided context controls the lifetime of the execution. Stopping the
// iteration early cancels the turn. If the Runner has been shut down, Run
// returns a single error event.
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

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		eventCh := make(chan agent.Event)
		doneCh := make(chan error, 1)
		emit := func(wctx context.Context, evt agent.Event) bool {
			select {
			case eventCh <- evt:
				return true
			case <-wctx.Done():
				return false
			}
		}

		// Submit work to the worker pool.
		submitErr := r.workerPool.Submit(ctx, func(wctx context.Context) {
			doneCh <- r.executeTurn(wctx, sessionID, input, func(evt agent.Event) bool {
				return emit(wctx, evt)
			})
		})
		if submitErr != nil {
			yield(agent.Event{}, core.NewError("runtime.runner.run", core.CodeOf(submitErr, core.ErrInvalidInput),
				"failed to submit work to pool", submitErr))
			return
		}

		for {
			select {
			case <-ctx.Done():
				yield(agent.Event{}, ctx.Err())
				return
			case evt := <-eventCh:
				if !yield(evt, nil) {
					return
				}
			case err := <-doneCh:
				if err != nil {
					// Run error through plugin chain.
					yield(agent.Event{}, r.pluginChain.RunOnError(ctx, err))
				}
				return
			}
		}
	}
}

// executeTurn performs the full turn lifecycle: session load/create,
// BeforeTurn plugins, agent streaming, AfterTurn plugins, and session update.
// Events are passed to emit; emit returning false means the caller stopped
// and the turn is abandoned without being recorded.
func (r *Runner) executeTurn(ctx context.Context, sessionID string, input schema.Message, emit func(agent.Event) bool) error {
	// Step 1: Load or create session, holding the per-session lock for the
	// whole turn so concurrent turns cannot overwrite each other.
	session, unlock, err := r.lockSession(ctx, sessionID)
	if err != nil {
		return err
	}
	defer unlock()

	// Step 2: Run BeforeTurn plugins.
	modifiedInput, err := r.pluginChain.RunBeforeTurn(ctx, session, input)
	if err != nil {
		return core.NewError("runtime.runner.beforeTurn", core.CodeOf(err, core.ErrInvalidInput),
			"plugin BeforeTurn failed", err)
	}

	// Step 3: Stream agent events. Without plugins nothing can rewrite the
	// events, so they are forwarded as they arrive.
	live := r.pluginChain.Len() == 0
	inputText := extractMessageText(modifiedInput)
	history := turnsToMessages(session.Turns)
	var events []agent.Event
	for evt, streamErr := range r.agent.Stream(ctx, inputText, agent.WithHistory(history)) {
		if streamErr != nil {
			return streamErr
		}
		evt = withSessionID(evt, session.ID)
		events = append(events, evt)
		if live && !emit(evt) {
			return ctx.Err()
		}
	}

	// Step 4: Run AfterTurn plugins.
	if !live {
		events, err = r.pluginChain.RunAfterTurn(ctx, session, events)
		if err != nil {
			return core.NewError("runtime.runner.afterTurn", core.CodeOf(err, core.ErrInvalidInput),
				"plugin AfterTurn failed", err)
		}
		for _, evt := range events {
			if !emit(withSessionID(evt, session.ID)) {
				return ctx.Err()
			}
		}
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

	return nil
}

// lockSession loads or creates the session and acquires its turn lock. The
// returned unlock func must be called when the turn ends.
func (r *Runner) lockSession(ctx context.Context, sessionID string) (*Session, func(), error) {
	if sessionID != "" {
		unlock := r.acquireSessionLock(sessionID)
		session, err := r.getOrCreateSession(ctx, sessionID)
		if err != nil {
			unlock()
			return nil, nil, err
		}
		if session.ID == sessionID {
			return session, unlock, nil
		}
		// A new session was created; switch to its lock.
		unlock()
		return session, r.acquireSessionLock(session.ID), nil
	}
	session, err := r.getOrCreateSession(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	return session, r.acquireSessionLock(session.ID), nil
}

// acquireSessionLock locks the per-session mutex and returns its unlock func.
func (r *Runner) acquireSessionLock(id string) func() {
	r.locksMu.Lock()
	if r.sessionLocks == nil {
		r.sessionLocks = make(map[string]*sessionLock)
	}
	l, ok := r.sessionLocks[id]
	if !ok {
		l = &sessionLock{}
		r.sessionLocks[id] = l
	}
	l.refs++
	r.locksMu.Unlock()

	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		r.locksMu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(r.sessionLocks, id)
		}
		r.locksMu.Unlock()
	}
}

// withSessionID returns evt with the session ID added to a copy of its
// metadata.
func withSessionID(evt agent.Event, id string) agent.Event {
	md := make(map[string]any, len(evt.Metadata)+1)
	for k, v := range evt.Metadata {
		md[k] = v
	}
	md[MetadataSessionID] = id
	evt.Metadata = md
	return evt
}

// turnsToMessages flattens session turns into the input/output message
// history passed to the agent.
func turnsToMessages(turns []schema.Turn) []schema.Message {
	msgs := make([]schema.Message, 0, len(turns)*2)
	for _, t := range turns {
		if t.Input != nil {
			msgs = append(msgs, t.Input)
		}
		if t.Output != nil {
			msgs = append(msgs, t.Output)
		}
	}
	return msgs
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
		return nil, core.NewError("runtime.runner.session", core.CodeOf(err, core.ErrInvalidInput),
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

// extractMessageText extracts the text content from a schema.Message,
// joining multiple text parts with newlines.
func extractMessageText(msg schema.Message) string {
	if msg == nil {
		return ""
	}
	var texts []string
	for _, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			texts = append(texts, tp.Text)
		}
	}
	return strings.Join(texts, "\n")
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
