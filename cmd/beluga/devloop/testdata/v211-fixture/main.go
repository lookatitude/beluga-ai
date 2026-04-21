// Package main is a v2.11.0-shape scaffold fixture used by the
// devloop-integration CI job to guarantee backward compatibility (brief
// Risk 7): `beluga run` / `beluga dev` must still work on projects that
// were scaffolded by v2.11.0, which do NOT call o11y.BootstrapFromEnv
// and do NOT wrap the agent with agent.WithTracing(). The fixture uses
// the mock LLM provider so the binary can run offline in CI with no API
// key, and prints "FIXTURE_RAN:" so the smoke job can count executions.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/llm"

	_ "github.com/lookatitude/beluga-ai/v2/llm/providers/mock"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	model, err := llm.New("mock", config.ProviderConfig{
		Provider: "mock",
		Model:    "mock-fixture",
	})
	if err != nil {
		log.Fatalf("llm.New: %v", err)
	}

	a := agent.New("v211-agent",
		agent.WithLLM(model),
		agent.WithPersona(agent.Persona{
			Role:      "fixture",
			Goal:      "exit cleanly in CI",
			Backstory: "Always return the canned mock response.",
		}),
	)

	answer, err := a.Invoke(ctx, "hello")
	if err != nil {
		log.Fatalf("agent.Invoke: %v", err)
	}
	fmt.Println("FIXTURE_RAN:", answer)
}
