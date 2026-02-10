// Package a2a implements the Agent-to-Agent (A2A) protocol for multi-agent
// collaboration. It provides an A2A server that exposes a Beluga agent as a
// remote agent with an Agent Card, task lifecycle management, and HTTP endpoints,
// as well as a client for connecting to remote A2A agents.
//
// The A2A protocol enables agents to discover each other via Agent Cards served
// at /.well-known/agent.json, submit tasks asynchronously, poll for task status,
// and cancel running tasks. Tasks follow a lifecycle: submitted -> working ->
// completed/failed/canceled.
//
// # Server
//
// A2AServer exposes a Beluga agent as a remote A2A agent via HTTP. It provides
// endpoints for the Agent Card, task creation, status polling, and cancellation.
//
//	card := a2a.AgentCard{
//	    Name:     "assistant",
//	    Endpoint: "http://localhost:9090",
//	}
//	srv := a2a.NewServer(myAgent, card)
//	srv.Serve(ctx, ":9090")
//
// The server exposes the following HTTP endpoints:
//
//   - GET  /.well-known/agent.json — returns the Agent Card
//   - POST /tasks — creates a new task
//   - GET  /tasks/{id} — returns task status
//   - POST /tasks/{id}/cancel — cancels a running task
//
// # Client
//
// A2AClient connects to a remote A2A agent and provides methods for retrieving
// the Agent Card, creating tasks, polling status, and cancellation.
//
//	client := a2a.NewClient("http://localhost:9090")
//	card, err := client.GetCard(ctx)
//	task, err := client.CreateTask(ctx, a2a.TaskRequest{Input: "Hello"})
//	task, err = client.GetTask(ctx, task.ID)
//
// # Remote Agent
//
// NewRemoteAgent wraps an A2A endpoint as a local agent.Agent, enabling
// transparent use of remote agents in local orchestration:
//
//	remote, err := a2a.NewRemoteAgent("http://localhost:9090")
//	result, err := remote.Invoke(ctx, "Hello")
//
// # Key Types
//
//   - A2AServer — serves a Beluga agent via the A2A protocol
//   - A2AClient — connects to remote A2A agents
//   - AgentCard — describes a remote agent's identity and capabilities
//   - Task — represents an A2A task with lifecycle state
//   - TaskStatus — lifecycle state (submitted, working, completed, failed, canceled)
//   - TaskRequest / TaskResponse / ErrorResponse — API message types
package a2a
