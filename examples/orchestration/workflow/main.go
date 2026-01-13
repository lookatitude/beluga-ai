package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("⚙️  Beluga AI - Workflow Orchestration Example")
	fmt.Println("===============================================")

	ctx := context.Background()

	// Step 1: Define a workflow function
	// In a real scenario, this would be a Temporal workflow function
	workflowFn := func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		fmt.Println("  Executing workflow step 1: Preprocessing")
		time.Sleep(100 * time.Millisecond)

		fmt.Println("  Executing workflow step 2: Processing")
		time.Sleep(100 * time.Millisecond)

		fmt.Println("  Executing workflow step 3: Postprocessing")
		time.Sleep(100 * time.Millisecond)

		result := map[string]interface{}{
			"status": "completed",
			"input":  input,
			"output": "Workflow completed successfully",
			"steps":  3,
		}

		return result, nil
	}

	// Step 3: Create a workflow
	// Note: This is a simplified example. In production, you would use Temporal
	// For demonstration, we'll execute the workflow function directly
	fmt.Println("✅ Created workflow")

	// Step 4: Execute the workflow
	fmt.Println("\n🚀 Executing workflow...")
	input := map[string]interface{}{
		"task":    "process-data",
		"data":    "example data",
		"options": map[string]interface{}{"priority": "high"},
	}

	// In a real Temporal workflow, this would return workflow ID and run ID
	// For this example, we'll simulate execution
	fmt.Println("  Workflow execution started")
	result, err := executeWorkflow(ctx, workflowFn, input)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	// Step 5: Display the result
	fmt.Printf("\n✅ Workflow Result:\n")
	fmt.Printf("  Status: %v\n", result["status"])
	fmt.Printf("  Output: %v\n", result["output"])
	fmt.Printf("  Steps: %v\n", result["steps"])

	// Step 6: Demonstrate workflow with conditional branching
	fmt.Println("\n🔄 Demonstrating conditional workflow...")
	conditionalWorkflow := func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		condition, _ := input["condition"].(string)

		if condition == "path-a" {
			fmt.Println("  Taking path A")
			return map[string]interface{}{
				"path":   "A",
				"result": "Path A completed",
			}, nil
		} else {
			fmt.Println("  Taking path B")
			return map[string]interface{}{
				"path":   "B",
				"result": "Path B completed",
			}, nil
		}
	}

	result1, _ := executeWorkflow(ctx, conditionalWorkflow, map[string]interface{}{"condition": "path-a"})
	result2, _ := executeWorkflow(ctx, conditionalWorkflow, map[string]interface{}{"condition": "path-b"})

	fmt.Printf("  Path A result: %v\n", result1)
	fmt.Printf("  Path B result: %v\n", result2)

	fmt.Println("\n✨ Workflow orchestration example completed successfully!")
}

// executeWorkflow is a helper function to execute workflow functions
// In production, this would be handled by Temporal
func executeWorkflow(ctx context.Context, workflowFn func(context.Context, map[string]interface{}) (map[string]interface{}, error), input map[string]interface{}) (map[string]interface{}, error) {
	return workflowFn(ctx, input)
}
