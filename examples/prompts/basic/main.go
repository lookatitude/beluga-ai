package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/pkg/prompts"
)

func main() {
	fmt.Println("📝 Beluga AI - Prompts Package Example")
	fmt.Println("======================================")

	ctx := context.Background()

	// Step 1: Create prompt manager
	fmt.Println("\n📋 Step 1: Creating prompt manager...")
	manager, err := prompts.NewPromptManager(
		prompts.WithConfig(prompts.DefaultConfig()),
	)
	if err != nil {
		log.Fatalf("Failed to create prompt manager: %v", err)
	}
	fmt.Println("✅ Prompt manager created")

	// Step 2: Create a string template
	fmt.Println("\n📋 Step 2: Creating string template...")
	template, err := manager.NewStringTemplate(
		"greeting",
		"Hello, {{name}}! Welcome to {{company}}. Today is {{date}}.",
	)
	if err != nil {
		log.Fatalf("Failed to create template: %v", err)
	}
	fmt.Println("✅ Template created")

	// Step 3: Format template with variables
	fmt.Println("\n📋 Step 3: Formatting template...")
	result, err := template.Format(ctx, map[string]interface{}{
		"name":    "Alice",
		"company": "Beluga AI",
		"date":    "2025-01-27",
	})
	if err != nil {
		log.Fatalf("Failed to format template: %v", err)
	}
	fmt.Printf("✅ Formatted result: %s\n", result)

	// Step 4: Create another template
	fmt.Println("\n📋 Step 4: Creating another template...")
	emailTemplate, err := manager.NewStringTemplate(
		"email",
		"Subject: {{subject}}\n\nDear {{recipient}},\n\n{{body}}\n\nBest regards,\n{{sender}}",
	)
	if err != nil {
		log.Fatalf("Failed to create email template: %v", err)
	}
	fmt.Println("✅ Email template created")

	// Step 5: Format email template
	fmt.Println("\n📋 Step 5: Formatting email template...")
	emailResult, err := emailTemplate.Format(ctx, map[string]interface{}{
		"subject":   "Welcome to Beluga AI",
		"recipient": "Alice",
		"body":      "Thank you for joining our platform!",
		"sender":    "Beluga Team",
	})
	if err != nil {
		log.Fatalf("Failed to format email template: %v", err)
	}
	fmt.Printf("✅ Email result:\n%s\n", emailResult)

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Create more complex templates with conditionals")
	fmt.Println("- Use chat message templates for LLM interactions")
	fmt.Println("- Integrate with prompt caching for performance")
}
