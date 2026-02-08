package openaicompat

import (
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

// ConvertTools converts a slice of Beluga ToolDefinitions to OpenAI tool parameters.
func ConvertTools(tools []schema.ToolDefinition) []openai.ChatCompletionToolParam {
	if len(tools) == 0 {
		return nil
	}
	out := make([]openai.ChatCompletionToolParam, len(tools))
	for i, t := range tools {
		fn := shared.FunctionDefinitionParam{
			Name:       t.Name,
			Parameters: shared.FunctionParameters(t.InputSchema),
		}
		if t.Description != "" {
			fn.Description = openai.String(t.Description)
		}
		out[i] = openai.ChatCompletionToolParam{
			Function: fn,
		}
	}
	return out
}
