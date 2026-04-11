// Package simulation provides simulation-based testing for agents.
//
// It enables multi-turn evaluation episodes where a simulated user interacts
// with an agent in a controlled environment. The SimRunner extends eval
// capabilities for episodic evaluation with configurable environments and
// user personas.
//
// Key types:
//   - SimEnvironment: Interface for resettable simulation environments
//   - SimulatedUser: LLM-driven user persona with configurable goals
//   - WebSimulator: Mock web environment with pages and forms
//   - SimRunner: Runs multi-turn simulation episodes with metrics
package simulation
