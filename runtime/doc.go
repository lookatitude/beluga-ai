// Package runtime provides the agent lifecycle management layer for the
// Beluga AI framework. It includes the Runner, Team composition, Plugin
// system, Session management, and WorkerPool.
//
// # Runner
//
// [Runner] is the lifecycle manager for a single agent. It handles session
// management, plugin execution, bounded concurrency, and graceful shutdown.
//
// Create a Runner with [NewRunner] and configure it with functional options:
//
//	import (
//	    "context"
//	    "fmt"
//
//	    "github.com/lookatitude/beluga-ai/v2/runtime"
//	    "github.com/lookatitude/beluga-ai/v2/runtime/plugins"
//	    "github.com/lookatitude/beluga-ai/v2/schema"
//	)
//
//	runner := runtime.NewRunner(myAgent,
//	    runtime.WithWorkerPoolSize(20),
//	    runtime.WithPlugins(
//	        plugins.NewRateLimit(60),
//	        plugins.NewAuditPlugin(auditStore),
//	        plugins.NewCostTracking(tracker, budget),
//	    ),
//	)
//	defer func() {
//	    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	    defer cancel()
//	    if err := runner.Shutdown(ctx); err != nil {
//	        fmt.Println("shutdown error:", err)
//	    }
//	}()
//
//	for evt, err := range runner.Run(ctx, "session-1", schema.NewHumanMessage("hello")) {
//	    if err != nil {
//	        break
//	    }
//	    fmt.Print(evt.Text)
//	}
//
// # Team
//
// [Team] groups agents and coordinates them with an [OrchestrationPattern].
// Teams implement [agent.Agent], enabling recursive composition — a Team can
// contain other Teams.
//
// Three patterns are built in:
//
//   - [PipelinePattern]: sequential; text output of each agent feeds the next.
//   - [SupervisorPattern]: coordinator LLM delegates to member agents.
//   - [ScatterGatherPattern]: all agents run in parallel; aggregator synthesizes.
//
// Example — pipeline team hosted by a Runner:
//
//	team := runtime.NewTeam(
//	    runtime.WithTeamID("draft-edit-review"),
//	    runtime.WithAgents(drafterAgent, editorAgent, reviewerAgent),
//	    runtime.WithPattern(runtime.PipelinePattern()),
//	)
//	runner := runtime.NewRunner(team)
//
// # Plugin
//
// [Plugin] is the cross-cutting concern interface. The [PluginChain] calls
// each plugin's BeforeTurn, AfterTurn, and OnError in registration order.
//
// To implement a plugin:
//
//	type myPlugin struct{}
//
//	func (p *myPlugin) Name() string { return "my-plugin" }
//
//	func (p *myPlugin) BeforeTurn(ctx context.Context, session *runtime.Session, input schema.Message) (schema.Message, error) {
//	    // optionally modify input
//	    return input, nil
//	}
//
//	func (p *myPlugin) AfterTurn(ctx context.Context, session *runtime.Session, events []agent.Event) ([]agent.Event, error) {
//	    // optionally modify events
//	    return events, nil
//	}
//
//	func (p *myPlugin) OnError(_ context.Context, err error) error {
//	    return err // return nil to suppress the error
//	}
//
// See package [github.com/lookatitude/beluga-ai/v2/runtime/plugins] for built-in
// plugin implementations.
//
// # Session
//
// [Session] holds the conversation state for one agent interaction. [SessionService]
// manages session lifecycle. The default [InMemorySessionService] is suitable
// for development. Replace it with a persistent backend via [WithSessionService]:
//
//	runner := runtime.NewRunner(myAgent,
//	    runtime.WithSessionService(myRedisSessionService),
//	)
//
// # WorkerPool
//
// [WorkerPool] provides bounded concurrency for agent execution. It ensures
// that no more than N agent tasks run concurrently, preventing resource
// exhaustion when many tasks arrive simultaneously:
//
//	pool := runtime.NewWorkerPool(8)
//
//	for _, task := range tasks {
//	    task := task // capture loop variable
//	    if err := pool.Submit(ctx, func(ctx context.Context) {
//	        if err := task.Execute(ctx); err != nil {
//	            slog.ErrorContext(ctx, "task failed", "error", err)
//	        }
//	    }); err != nil {
//	        // context cancelled while waiting for a slot
//	        break
//	    }
//	}
//
//	pool.Wait() // block until all submitted work completes
//
// [WorkerPool.Drain] stops accepting new work and waits for all in-flight
// tasks to finish, respecting the provided context deadline:
//
//	if err := pool.Drain(shutdownCtx); err != nil {
//	    // shutdown context expired before all tasks drained
//	}
package runtime
