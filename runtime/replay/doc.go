// Package replay provides agent replay and time-travel debugging capabilities.
//
// It captures checkpoints of agent sessions and allows re-execution from any
// checkpoint using recorded LLM responses. A divergence detector compares
// replay events against original recordings to identify behavioral differences.
//
// Key types:
//   - Checkpoint: Captures session state at a point in time
//   - CheckpointStore: Interface for persisting and retrieving checkpoints
//   - Replayer: Re-executes from a checkpoint with recorded responses
//   - DivergenceDetector: Compares replay events vs original events
package replay
