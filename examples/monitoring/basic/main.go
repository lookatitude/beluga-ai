package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring"
)

func main() {
	fmt.Println("📊 Beluga AI - Monitoring Package Example")
	fmt.Println("==========================================")

	ctx := context.Background()

	// Step 1: Create monitoring system
	fmt.Println("\n📋 Step 1: Creating monitoring system...")
	// Use defaults - safety and ethical checks are enabled by default
	monitor, err := monitoring.NewMonitor()
	if err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}
	fmt.Println("✅ Monitor created")

	// Step 2: Use structured logging
	fmt.Println("\n📋 Step 2: Using structured logging...")
	monitor.Logger().Info(ctx, "Example application started", map[string]interface{}{
		"service": "beluga-example",
		"version": "1.0.0",
	})
	fmt.Println("✅ Log entry created")

	// Step 3: Create a trace span
	fmt.Println("\n📋 Step 3: Creating trace span...")
	ctx, span := monitor.Tracer().StartSpan(ctx, "example_operation")
	defer monitor.Tracer().FinishSpan(span)
	fmt.Println("✅ Trace span created")

	// Step 4: Perform safety check
	fmt.Println("\n📋 Step 4: Performing safety check...")
	safetyResult, err := monitor.SafetyChecker().CheckContent(ctx, "Hello, how are you?", "chat")
	if err != nil {
		log.Printf("Safety check error: %v", err)
	} else {
		fmt.Printf("✅ Safety check passed: Safe=%v, RiskScore=%.2f\n", safetyResult.Safe, safetyResult.RiskScore)
	}

	// Step 5: Record metrics
	fmt.Println("\n📋 Step 5: Recording metrics...")
	// Metrics are automatically recorded by the monitor
	time.Sleep(100 * time.Millisecond) // Simulate some work
	fmt.Println("✅ Metrics recorded")

	// Step 6: Perform health check
	fmt.Println("\n📋 Step 6: Performing health check...")
	healthChecks := monitor.HealthChecker().RunChecks(ctx)
	isHealthy := monitor.HealthChecker().IsHealthy(ctx)
	fmt.Printf("✅ Health checks: %+v\n", healthChecks)
	fmt.Printf("✅ Is healthy: %v\n", isHealthy)

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Configure OpenTelemetry endpoint for distributed tracing")
	fmt.Println("- Set up metrics collection backends")
	fmt.Println("- Customize safety and ethical validation rules")
}
