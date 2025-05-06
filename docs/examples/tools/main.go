// docs/examples/tools/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/tools"
	"github.com/lookatitude/beluga-ai/tools/gofunc"
	"github.com/lookatitude/beluga-ai/tools/shell"
)

// Define a simple Go function to be used as a tool
func getCurrentWeather(location string, unit string) (string, error) {
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
	 weatherTool, err := gofunc.NewGoFuncTool(
	 	 "get_current_weather",
	 	 "Get the current weather in a given location",
	 	 weatherSchema,
	 	 getCurrentWeather, // Pass the function directly
	 )
	 if err != nil {
	 	 log.Fatalf("Failed to create weather tool: %v", err)
	 }

	 // Prepare arguments as a JSON string (as an agent might)
	 argsWeather := `{"location": "London", "unit": "celsius"}`

	 // Execute the tool
	 resultWeather, err := weatherTool.Execute(ctx, argsWeather)
	 if err != nil {
	 	 log.Printf("Weather tool execution failed: %v", err)
	 } else {
	 	 fmt.Printf("Weather Tool Result: %s\n", resultWeather)
	 }

	 // --- Shell Tool Example --- 
	 fmt.Println("\n--- Shell Tool Example ---")

	 // Create a shell tool (use with caution!)
	 // Schema is optional for shell tool, but recommended if arguments are expected
	 shellToolSchema := `{ 
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The shell command to execute."
			}
		},
		"required": ["command"]
	}`
	 listFilesTool, err := shell.NewShellTool(
	 	 "list_files",
	 	 "Lists files in the current directory using the ls command.",
	 	 shellToolSchema, // Provide schema if needed
	 	 shell.WithAllowedCommands([]string{"ls"}), // IMPORTANT: Restrict allowed commands
	 )
	 if err != nil {
	 	 log.Fatalf("Failed to create shell tool: %v", err)
	 }

	 // Prepare arguments (command to run)
	 argsShellMap := map[string]string{"command": "ls -la"}
	 argsShellJSON, _ := json.Marshal(argsShellMap)

	 // Execute the shell tool
	 resultShell, err := listFilesTool.Execute(ctx, string(argsShellJSON))
	 if err != nil {
	 	 log.Printf("Shell tool execution failed: %v", err)
	 } else {
	 	 fmt.Printf("Shell Tool Result:\n%s\n", resultShell)
	 }

	 // --- Using the Base Tool Interface --- 
	 fmt.Println("\n--- Base Tool Interface Example ---")
	 // You can treat all tools uniformly using the tools.Tool interface
	 myTools := []tools.Tool{weatherTool, listFilesTool}

	 for _, tool := range myTools {
	 	 fmt.Printf("Tool Name: %s\n", tool.Name())
	 	 fmt.Printf("Tool Description: %s\n", tool.Description())
	 	 fmt.Printf("Tool Schema: %s\n", tool.Schema())
	 }
}

