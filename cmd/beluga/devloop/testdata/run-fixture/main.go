// Package main is a current-shape (v2.12+) scaffold fixture used by the
// devloop-integration CI job. It mirrors what `beluga init --template basic`
// produces after the DX-1 S3 dev-loop feature landed: the project calls
// o11y.BootstrapFromEnv at startup and wraps its agent with the tracing
// middleware. The fixture uses the mock LLM provider so `beluga run` can
// execute offline in CI with no API key, and it prints the literal marker
// line "FIXTURE_RAN:" so the smoke job can count how many times the child
// ran without parsing supervisor stderr.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/o11y"

	_ "github.com/lookatitude/beluga-ai/v2/llm/providers/mock"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shutdown, err := o11y.BootstrapFromEnv(ctx, "runfixture")
	if err != nil {
		log.Fatalf("o11y.BootstrapFromEnv: %v", err)
	}
	defer shutdown()

	model, err := llm.New("mock", config.ProviderConfig{
		Provider: "mock",
		Model:    "mock-fixture",
	})
	if err != nil {
		log.Fatalf("llm.New: %v", err)
	}

	a := agent.ApplyMiddleware(
		agent.New("runfixture-agent",
			agent.WithLLM(model),
			agent.WithPersona(agent.Persona{
				Role:      "fixture",
				Goal:      "exit cleanly in CI",
				Backstory: "Always return the canned mock response.",
			}),
		),
		agent.WithTracing(),
	)

	answer, err := a.Invoke(ctx, "hello")
	if err != nil {
		log.Fatalf("agent.Invoke: %v", err)
	}
	fmt.Println("FIXTURE_RAN:", answer)
}
