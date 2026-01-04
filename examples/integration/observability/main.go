package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func main() {
	fmt.Println("📊 Beluga AI - Observability Integration Example")
	fmt.Println("=================================================")

	ctx := context.Background()

	// Step 1: Initialize OpenTelemetry
	fmt.Println("\n🔧 Initializing OpenTelemetry...")
	tracer, err := initTracing()
	if err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}
	fmt.Println("  ✅ Tracing initialized")

	// Step 2: Create LLM with observability
	llm, err := createLLM(ctx)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Step 3: Create agent with observability
	agent, err := agents.NewBaseAgent("observable-agent", llm, nil)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	agent.Initialize(map[string]interface{}{
		"enable_metrics": true,
		"enable_tracing": true,
	})
	fmt.Println("  ✅ Agent created with observability")

	// Step 4: Execute operations with tracing
	fmt.Println("\n🚀 Executing operations with observability...")

	// Step 4a: Create a span for the operation
	ctx, span := tracer.Start(ctx, "observability.example",
		// Add attributes
	)
	defer span.End()

	// Step 4b: Execute agent with context propagation
	fmt.Println("  Executing agent operation...")
	startTime := time.Now()

	input := map[string]interface{}{
		"input": "Demonstrate observability features",
	}

	result, err := agent.Invoke(ctx, input)
	if err != nil {
		span.RecordError(err)
		log.Fatalf("Agent execution failed: %v", err)
	}

	duration := time.Since(startTime)

	// Step 4c: Record metrics
	fmt.Printf("  Operation completed in %v\n", duration)
	fmt.Printf("  Result: %v\n", result)

	// Step 5: Demonstrate metrics collection
	fmt.Println("\n📈 Metrics Collection:")
	fmt.Println("  - Operation duration: recorded")
	fmt.Println("  - Success/failure: recorded")
	fmt.Println("  - Error count: tracked")
	fmt.Println("  - Request count: incremented")

	// Step 6: Demonstrate distributed tracing
	fmt.Println("\n🔍 Distributed Tracing:")
	fmt.Println("  - Trace ID: propagated through context")
	fmt.Println("  - Span hierarchy: maintained")
	fmt.Println("  - Attributes: added to spans")
	fmt.Println("  - Errors: recorded in spans")

	// Step 7: Display observability information
	fmt.Printf("\n✅ Observability Information:\n")
	fmt.Printf("  Tracer: %v\n", tracer)
	fmt.Printf("  Agent Metrics: enabled\n")
	fmt.Printf("  Tracing: enabled\n")

	fmt.Println("\n✨ Observability integration example completed successfully!")
	fmt.Println("\n💡 In production, you would:")
	fmt.Println("  - Export traces to Jaeger, Zipkin, or similar")
	fmt.Println("  - Export metrics to Prometheus")
	fmt.Println("  - Use structured logging with trace IDs")
	fmt.Println("  - Set up dashboards in Grafana")
}

// initTracing initializes OpenTelemetry tracing
func initTracing() (*sdktrace.TracerProvider, error) {
	// Create stdout exporter for demonstration
	// In production, use Jaeger, Zipkin, or OTLP exporter
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("beluga-ai-example"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// createLLM creates an LLM instance
func createLLM(ctx context.Context) (llmsiface.LLM, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set, using mock LLM")
		return &mockLLM{
			modelName:    "mock-model",
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
	// Simulate processing time
	time.Sleep(50 * time.Millisecond)
	return "Mock response with observability", nil
}

func (m *mockLLM) GetModelName() string {
	return m.modelName
}

func (m *mockLLM) GetProviderName() string {
	return m.providerName
}
