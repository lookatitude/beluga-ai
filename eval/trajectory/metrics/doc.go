// Package metrics provides built-in trajectory evaluation metrics for the
// Beluga AI eval framework. Each metric implements the
// trajectory.TrajectoryMetric interface, returning an overall score in [0, 1]
// along with per-step details.
//
// # Available Metrics
//
//   - tool_selection: F1 score comparing actual vs expected tool usage.
//   - planning_quality: step efficiency, redundancy detection, goal achievement.
//   - trajectory_faithfulness: LLM-as-judge faithfulness scoring.
//   - step_efficiency: ratio of productive steps to total steps.
//   - cost_per_task: normalized cost score based on token metadata.
//
// All metrics register themselves via init() and can be created through the
// trajectory.New() factory or instantiated directly.
package metrics
