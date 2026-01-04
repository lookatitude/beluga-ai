package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

func main() {
	fmt.Println("🎯 Beluga AI - Specialized Multi-Agent System Example")
	fmt.Println("=======================================================")

	ctx := context.Background()

	// Step 1: Create specialized agents with different roles
	agents := createSpecializedAgents(ctx)
	fmt.Printf("✅ Created %d specialized agents\n", len(agents))

	// Step 2: Demonstrate hierarchical agent structure
	fmt.Println("\n📊 Agent Hierarchy:")
	fmt.Println("  Manager Agent")
	fmt.Println("    ├── Research Agent")
	fmt.Println("    ├── Analysis Agent")
	fmt.Println("    └── Report Agent")

	// Step 3: Execute hierarchical task delegation
	fmt.Println("\n🚀 Executing hierarchical task...")
	task := "Research and analyze market trends for AI technology"

	// Step 3a: Manager delegates to research agent
	fmt.Println("\n👔 Manager delegating to Research Agent...")
	researchTask := map[string]interface{}{
		"input": fmt.Sprintf("Research: %s", task),
	}
	researchResult, err := agents["research"].Invoke(ctx, researchTask)
	if err != nil {
		log.Fatalf("Research agent failed: %v", err)
	}
	fmt.Printf("  Research result: %v\n", researchResult)

	// Step 3b: Manager delegates to analysis agent
	fmt.Println("\n👔 Manager delegating to Analysis Agent...")
	analysisTask := map[string]interface{}{
		"input": fmt.Sprintf("Analyze research findings: %v", researchResult),
	}
	analysisResult, err := agents["analysis"].Invoke(ctx, analysisTask)
	if err != nil {
		log.Fatalf("Analysis agent failed: %v", err)
	}
	fmt.Printf("  Analysis result: %v\n", analysisResult)

	// Step 3c: Manager delegates to report agent
	fmt.Println("\n👔 Manager delegating to Report Agent...")
	reportTask := map[string]interface{}{
		"input": fmt.Sprintf("Create report from research and analysis: Research=%v, Analysis=%v", researchResult, analysisResult),
	}
	reportResult, err := agents["report"].Invoke(ctx, reportTask)
	if err != nil {
		log.Fatalf("Report agent failed: %v", err)
	}
	fmt.Printf("  Report result: %v\n", reportResult)

	// Step 4: Manager synthesizes final result
	fmt.Println("\n👔 Manager synthesizing final result...")
	finalResult := map[string]interface{}{
		"task":           task,
		"research":       researchResult,
		"analysis":       analysisResult,
		"report":         reportResult,
		"status":         "completed",
		"agents_used":    []string{"research", "analysis", "report"},
	}

	// Step 5: Display final result
	fmt.Printf("\n✅ Final Hierarchical Result:\n")
	fmt.Printf("  Task: %s\n", finalResult["task"])
	fmt.Printf("  Research: %v\n", finalResult["research"])
	fmt.Printf("  Analysis: %v\n", finalResult["analysis"])
	fmt.Printf("  Report: %v\n", finalResult["report"])
	fmt.Printf("  Status: %v\n", finalResult["status"])

	// Step 6: Demonstrate agent delegation patterns
	fmt.Println("\n🔄 Demonstrating delegation patterns...")
	demonstrateDelegation(ctx, agents)

	fmt.Println("\n✨ Specialized multi-agent system example completed successfully!")
}

// createSpecializedAgents creates agents with specialized roles
func createSpecializedAgents(ctx context.Context) map[string]interface{} {
	agents := make(map[string]interface{})

	// Research agent - specializes in gathering information
	researchLLM, _ := createLLM(ctx, "research")
	research, _ := agents.NewBaseAgent("research-specialist", researchLLM, nil)
	research.Initialize(map[string]interface{}{
		"specialization": "research",
		"expertise":       "information-gathering",
	})
	agents["research"] = research

	// Analysis agent - specializes in data analysis
	analysisLLM, _ := createLLM(ctx, "analysis")
	analysis, _ := agents.NewBaseAgent("analysis-specialist", analysisLLM, nil)
	analysis.Initialize(map[string]interface{}{
		"specialization": "analysis",
		"expertise":       "data-analysis",
	})
	agents["analysis"] = analysis

	// Report agent - specializes in report generation
	reportLLM, _ := createLLM(ctx, "report")
	report, _ := agents.NewBaseAgent("report-specialist", reportLLM, nil)
	report.Initialize(map[string]interface{}{
		"specialization": "reporting",
		"expertise":       "document-generation",
	})
	agents["report"] = report

	return agents
}

// demonstrateDelegation shows different delegation patterns
func demonstrateDelegation(ctx context.Context, agents map[string]interface{}) {
	// Pattern 1: Sequential delegation
	fmt.Println("\n  Pattern 1: Sequential Delegation")
	fmt.Println("    Agent A -> Agent B -> Agent C")

	// Pattern 2: Parallel delegation
	fmt.Println("\n  Pattern 2: Parallel Delegation")
	fmt.Println("    Agent A -> Agent B (parallel)")
	fmt.Println("    Agent A -> Agent C (parallel)")

	// Pattern 3: Conditional delegation
	fmt.Println("\n  Pattern 3: Conditional Delegation")
	fmt.Println("    If condition X: Agent B")
	fmt.Println("    Else: Agent C")
}

// createLLM creates an LLM instance
func createLLM(ctx context.Context, name string) (llmsiface.LLM, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return &mockLLM{
			modelName:    fmt.Sprintf("mock-%s", name),
			providerName: "mock-provider",
		}, nil
	}

	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-3.5-turbo"),
		llms.WithAPIKey(apiKey),
	)

	factory := llms.NewFactory()
	llm, err := factory.CreateProvider("openai", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	return llm, nil
}

// mockLLM is a simple mock implementation
type mockLLM struct {
	modelName    string
	providerName string
}

func (m *mockLLM) Invoke(ctx context.Context, prompt string, callOptions ...interface{}) (string, error) {
	return fmt.Sprintf("Mock response from %s", m.modelName), nil
}

func (m *mockLLM) GetModelName() string {
	return m.modelName
}

func (m *mockLLM) GetProviderName() string {
	return m.providerName
}
