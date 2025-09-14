// docs/examples/tools/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time" // Added for shell tool timeout

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools/gofunc"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools/shell"
)

// Define a simple Go function to be used as a tool
// Updated signature to match what GoFunctionTool expects: (context.Context, map[string]any) -> (any, error)
func getCurrentWeather(ctx context.Context, args map[string]any) (any, error) {
	location, okL := args["location"].(string)
	unit, okU := args["unit"].(string)

	if !okL || !okU {
		return nil, fmt.Errorf("invalid arguments: location and unit must be strings")
	}

	// In a real scenario, this would call a weather API
	if location == "London" {
		if unit == "celsius" {
			return `{"temperature": 15, "unit": "celsius", "description": "Cloudy"}`, nil
		} else {
			return `{"temperature": 59, "unit": "fahrenheit", "description": "Cloudy"}`, nil
		}
	}
	return `{"error": "Location not found"}`, fmt.Errorf("location not found: %s", location)
}

// Define the schema for the Go function tool
var weatherSchema = `{ 
    "type": "object",
    "properties": {
        "location": {
            "type": "string",
            "description": "The city and state, e.g. San Francisco, CA"
        },
        "unit": { 
            "type": "string", 
            "enum": ["celsius", "fahrenheit"],
            "description": "The temperature unit to use. Infer this from the user's location."
        }
    },
    "required": ["location", "unit"]
}`

func main() {
	ctx := context.Background()

	// --- Go Function Tool Example --- 
	fmt.Println("--- Go Function Tool Example ---")

	// Create the Go function tool
	weatherTool, err := gofunc.NewGoFunctionTool(
		"get_current_weather",
		"Get the current weather in a given location",
		weatherSchema,
		getCurrentWeather, // Pass the function directly
	)
	if err != nil {
		log.Fatalf("Failed to create weather tool: %v", err)
	}

	// Prepare arguments as a map[string]any (as GoFunctionTool Execute expects)
	argsWeatherMap := map[string]any{"location": "London", "unit": "celsius"}

	// Execute the tool
	resultWeather, err := weatherTool.Execute(ctx, argsWeatherMap)
	if err != nil {
		log.Printf("Weather tool execution failed: %v", err)
	} else {
		fmt.Printf("Weather Tool Result: %s\n", resultWeather)
	}

	// --- Shell Tool Example --- 
	fmt.Println("\n--- Shell Tool Example ---")

	// Create a shell tool (use with caution!)
	// NewShellTool now only takes a timeout. Name, description, and schema are defaulted.
	// AllowedCommands functionality is not present in the current shell.go constructor.
	listFilesTool, err := shell.NewShellTool(5 * time.Second) // Example timeout
	if err != nil {
		log.Fatalf("Failed to create shell tool: %v", err)
	}
	// To customize name, description, or schema, you would modify the Def field after creation if needed:
	// listFilesTool.Def.Name = "list_files_custom"
	// listFilesTool.Def.Description = "Lists files in the current directory using the ls command (custom)."
	// For this example, we use the defaults.

	// Prepare arguments (command to run). ShellTool Execute can take string or map[string]any.
	argsShell := "ls -la" // Direct string input

	// Execute the shell tool
	resultShell, err := listFilesTool.Execute(ctx, argsShell)
	if err != nil {
		// ShellTool Execute now returns error as nil and puts command error in the result string.
		// So, this path might not be hit for command execution errors, only for input validation type errors.
		log.Printf("Shell tool execution error (unexpected): %v", err)
		fmt.Printf("Shell Tool Result (with error in output):\n%s\n", resultShell)
	} else {
		fmt.Printf("Shell Tool Result:\n%s\n", resultShell)
	}

	// --- Using the Base Tool Interface --- 
	fmt.Println("\n--- Base Tool Interface Example ---")
	// You can treat all tools uniformly using the tools.Tool interface
	myTools := []tools.Tool{weatherTool, listFilesTool}

	for _, tool := range myTools {
		definition := tool.Definition()
		fmt.Printf("Tool Name: %s\n", definition.Name)
		fmt.Printf("Tool Description: %s\n", definition.Description)
		// InputSchema is map[string]any, marshal to JSON for printing as string
		schemaBytes, _ := json.MarshalIndent(definition.InputSchema, "", "  ")
		fmt.Printf("Tool Schema: %s\n", string(schemaBytes))
	}
}

