package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// ToolGenerator uses an LLM to generate tool implementations from natural language
// descriptions. It produces DynamicTool instances with generated code and schemas.
type ToolGenerator struct {
	model          llm.ChatModel
	validator      *ASTValidator
	allowedImports []string
	maxRetries     int
}

// GeneratorOption configures a ToolGenerator.
type GeneratorOption func(*generatorOptions)

type generatorOptions struct {
	allowedImports []string
	maxRetries     int
}

// WithAllowedImports sets the list of Go import paths allowed in generated code.
// If not set, the default safe set from ASTValidator is used.
func WithAllowedImports(imports []string) GeneratorOption {
	return func(o *generatorOptions) {
		o.allowedImports = imports
	}
}

// WithMaxRetries sets the maximum number of generation retries on validation failure.
// Defaults to 3.
func WithMaxRetries(n int) GeneratorOption {
	return func(o *generatorOptions) {
		if n > 0 {
			o.maxRetries = n
		}
	}
}

// NewToolGenerator creates a new ToolGenerator that uses the given ChatModel for
// code generation. Options can configure allowed imports and retry behavior.
func NewToolGenerator(model llm.ChatModel, opts ...GeneratorOption) *ToolGenerator {
	o := generatorOptions{
		maxRetries: 3,
	}
	for _, opt := range opts {
		opt(&o)
	}

	return &ToolGenerator{
		model:          model,
		validator:      NewASTValidator(o.allowedImports),
		allowedImports: o.allowedImports,
		maxRetries:     o.maxRetries,
	}
}

// GenerateRequest describes what tool to generate.
type GenerateRequest struct {
	// Name is the desired tool name.
	Name string
	// Description describes what the tool should do.
	Description string
	// InputFields describes the expected input parameters as field name to description.
	InputFields map[string]string
	// Examples provides optional few-shot examples of input/output pairs.
	Examples []Example
}

// Example is an input/output pair for few-shot prompting.
type Example struct {
	// Input is a sample input as a JSON object string.
	Input string
	// Output is the expected output string.
	Output string
}

// generatedTool is the structured output expected from the LLM.
type generatedTool struct {
	Code        string         `json:"code"`
	InputSchema map[string]any `json:"input_schema"`
}

// Generate creates a new DynamicTool from the given request using the LLM.
// It retries up to MaxRetries times if the generated code fails AST validation.
// The returned tool uses a NoopExecutor by default; callers should replace the
// executor with a production implementation before registering.
func (g *ToolGenerator) Generate(ctx context.Context, req GenerateRequest, executor CodeExecutor) (*DynamicTool, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("tool generator: name is required")
	}
	if req.Description == "" {
		return nil, fmt.Errorf("tool generator: description is required")
	}

	prompt := g.buildPrompt(req)

	var lastErr error
	for attempt := 0; attempt <= g.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		msgs := []schema.Message{
			schema.NewSystemMessage(systemPrompt(g.allowedImports)),
			schema.NewHumanMessage(prompt),
		}

		if lastErr != nil {
			msgs = append(msgs, schema.NewHumanMessage(
				fmt.Sprintf("The previous attempt failed validation: %v\nPlease fix the code and try again.", lastErr),
			))
		}

		resp, err := g.model.Generate(ctx, msgs)
		if err != nil {
			return nil, fmt.Errorf("tool generator: llm error: %w", err)
		}

		text := resp.Text()
		gen, err := parseGeneratedTool(text)
		if err != nil {
			lastErr = fmt.Errorf("failed to parse LLM output: %w", err)
			continue
		}

		// Validate the generated code.
		if err := g.validator.Validate(gen.Code); err != nil {
			lastErr = err
			continue
		}

		return NewDynamicTool(
			req.Name,
			req.Description,
			gen.InputSchema,
			gen.Code,
			executor,
		), nil
	}

	return nil, fmt.Errorf("tool generator: failed after %d retries: %w", g.maxRetries, lastErr)
}

// buildPrompt constructs the user prompt for tool generation.
func (g *ToolGenerator) buildPrompt(req GenerateRequest) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Create a Go tool named %q.\n", req.Name))
	b.WriteString(fmt.Sprintf("Description: %s\n\n", req.Description))

	if len(req.InputFields) > 0 {
		b.WriteString("Input fields:\n")
		for name, desc := range req.InputFields {
			b.WriteString(fmt.Sprintf("  - %s: %s\n", name, desc))
		}
		b.WriteString("\n")
	}

	if len(req.Examples) > 0 {
		b.WriteString("Examples:\n")
		for i, ex := range req.Examples {
			b.WriteString(fmt.Sprintf("  Example %d:\n    Input: %s\n    Output: %s\n", i+1, ex.Input, ex.Output))
		}
	}

	return b.String()
}

// systemPrompt returns the system message for tool generation.
func systemPrompt(allowedImports []string) string {
	imports := "encoding/json, fmt, math, strings, strconv, sort, unicode"
	if len(allowedImports) > 0 {
		imports = strings.Join(allowedImports, ", ")
	}

	return fmt.Sprintf(`You are a tool code generator. Generate Go source code for tools.

Rules:
1. Only use these imports: %s
2. Do not use goroutines (no go statements)
3. Do not use the unsafe package
4. The code must be a complete, valid Go file with a package declaration
5. The main function should be named "run" and accept a JSON string input parameter and return (string, error)

Respond with ONLY a JSON object with these fields:
- "code": the complete Go source code as a string
- "input_schema": a JSON Schema object describing the input parameters

Do not include any text before or after the JSON object.`, imports)
}

// parseGeneratedTool extracts the generated tool from the LLM response text.
func parseGeneratedTool(text string) (*generatedTool, error) {
	// Try to find JSON in the response.
	text = strings.TrimSpace(text)

	// Strip markdown code fences if present.
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		var jsonLines []string
		inBlock := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock {
				jsonLines = append(jsonLines, line)
			}
		}
		text = strings.Join(jsonLines, "\n")
	}

	var gen generatedTool
	if err := json.Unmarshal([]byte(text), &gen); err != nil {
		return nil, fmt.Errorf("invalid JSON in response: %w", err)
	}

	if gen.Code == "" {
		return nil, fmt.Errorf("generated code is empty")
	}
	if gen.InputSchema == nil {
		gen.InputSchema = map[string]any{"type": "object"}
	}

	return &gen, nil
}
