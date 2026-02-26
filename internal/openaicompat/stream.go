package openaicompat

import (
	"iter"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/ssestream"
)

// StreamToSeq converts an openai-go SSE stream into a Beluga iter.Seq2 of StreamChunks.
// It handles text deltas, tool call accumulation by index, finish reasons, and usage.
func StreamToSeq(stream *ssestream.Stream[openai.ChatCompletionChunk], modelID string) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {
		defer stream.Close()
		for stream.Next() {
			sc := convertChunk(stream.Current(), modelID)
			if !yield(sc, nil) {
				return
			}
		}
		if err := stream.Err(); err != nil {
			yield(schema.StreamChunk{}, err)
		}
	}
}

// convertChunk converts an OpenAI stream chunk to a Beluga StreamChunk.
func convertChunk(chunk openai.ChatCompletionChunk, modelID string) schema.StreamChunk {
	sc := schema.StreamChunk{ModelID: modelID}
	if len(chunk.Choices) > 0 {
		delta := chunk.Choices[0].Delta
		sc.Delta = delta.Content
		sc.FinishReason = chunk.Choices[0].FinishReason
		if len(delta.ToolCalls) > 0 {
			sc.ToolCalls = make([]schema.ToolCall, len(delta.ToolCalls))
			for i, tc := range delta.ToolCalls {
				sc.ToolCalls[i] = schema.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				}
			}
		}
	}
	if chunk.Usage.TotalTokens > 0 {
		sc.Usage = &schema.Usage{
			InputTokens:  int(chunk.Usage.PromptTokens),
			OutputTokens: int(chunk.Usage.CompletionTokens),
			TotalTokens:  int(chunk.Usage.TotalTokens),
			CachedTokens: int(chunk.Usage.PromptTokensDetails.CachedTokens),
		}
	}
	return sc
}
