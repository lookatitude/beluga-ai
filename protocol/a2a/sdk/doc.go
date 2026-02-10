// Package sdk provides integration between the official A2A Go SDK
// (github.com/a2aproject/a2a-go) and Beluga's A2A protocol layer.
// It bridges Beluga's agent.Agent interface with the SDK's server and client,
// enabling exposure of Beluga agents as A2A remote agents and consumption
// of remote A2A agents as Beluga agents.
//
// This package is useful when you need full compliance with the official A2A
// SDK behavior, including JSON-RPC messaging, event queues, and the standard
// AgentCard format.
//
// # Server
//
// NewServer creates an A2A request handler and agent card from a Beluga agent.
// The returned handler should be mounted on an HTTP server. The agent's tools
// are automatically converted to A2A skills.
//
//	handler, card := sdk.NewServer(myAgent, sdk.ServerConfig{
//	    Name:        "my-agent",
//	    Version:     "1.0.0",
//	    Description: "A helpful assistant",
//	    URL:         "http://localhost:9090",
//	})
//	http.ListenAndServe(":9090", handler)
//
// # Client
//
// NewRemoteAgent creates a Beluga agent.Agent that delegates to a remote A2A
// agent via the official SDK client. It fetches the AgentCard to populate the
// agent's identity.
//
//	remote, err := sdk.NewRemoteAgent(ctx, "http://remote-agent:9090")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := remote.Invoke(ctx, "Hello")
//
// # Key Types
//
//   - ServerConfig — configuration for creating an A2A SDK server
//   - NewServer — creates handler and card from a Beluga agent
//   - NewRemoteAgent — wraps a remote A2A agent as a local agent.Agent
package sdk
