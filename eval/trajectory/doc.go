// Package trajectory provides agent trajectory evaluation for the Beluga AI
// eval framework. A trajectory is an ordered sequence of steps an agent takes
// to accomplish a task, including planning, tool calls, responses, and handoffs.
//
// # Core Types
//
// Trajectory represents a complete agent execution trace with ordered Steps.
// Each Step has a type (plan, tool_call, respond, handoff, finish), an action,
// a result, latency, and metadata.
//
// # Metrics
//
// TrajectoryMetric is the interface for scoring trajectories. Built-in metrics
// are available in the eval/trajectory/metrics sub-package:
//
//   - tool_selection: F1 score comparing actual vs expected tool usage.
//   - planning_quality: step efficiency, redundancy detection, goal achievement.
//   - trajectory_faithfulness: LLM-as-judge faithfulness scoring.
//   - step_efficiency: ratio of productive steps to total steps.
//   - cost_per_task: normalized cost score based on token usage metadata.
//
// # Recorder
//
// Recorder captures agent execution into a Trajectory via agent.Hooks. It is
// thread-safe and composable with user hooks via agent.ComposeHooks.
//
// # Runner
//
// Runner evaluates a set of trajectories against configured metrics with
// bounded concurrency. It produces a Report with per-trajectory and aggregate
// scores.
//
// # Registry
//
// Metrics are extensible via the standard Register/New/List pattern.
//
// # Usage
//
//	// Record an agent's execution
//	rec := trajectory.NewRecorder()
//	agent.SetHooks(rec.Hooks())
//	agent.Run(ctx, "solve the problem")
//	traj := rec.Trajectory()
//
//	// Evaluate trajectories
//	runner := trajectory.NewRunner(
//	    trajectory.WithMetrics(
//	        toolselection.New(),
//	        stepefficiency.New(),
//	    ),
//	    trajectory.WithTrajectories([]trajectory.Trajectory{*traj}),
//	    trajectory.WithParallel(4),
//	)
//	report, err := runner.Run(ctx)
package trajectory
