// Package internal provides agent integration for multimodal capabilities.
package internal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// MultimodalAgentExtension provides multimodal capabilities for agents.
type MultimodalAgentExtension struct {
	model *BaseMultimodalModel
}

// NewMultimodalAgentExtension creates a new multimodal agent extension.
func NewMultimodalAgentExtension(model *BaseMultimodalModel) *MultimodalAgentExtension {
	return &MultimodalAgentExtension{
		model: model,
	}
}

// HandleMultimodalMessage processes multimodal messages (ImageMessage, VideoMessage, VoiceDocument).
func (m *MultimodalAgentExtension) HandleMultimodalMessage(ctx context.Context, msg schema.Message) (*types.MultimodalInput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.HandleMultimodalMessage",
		trace.WithAttributes(
			attribute.String("message_type", fmt.Sprintf("%T", msg)),
		))
	defer span.End()

	contentBlocks := make([]*types.ContentBlock, 0)

	// Handle ImageMessage
	if imgMsg, ok := schema.AsImageMessage(msg); ok {
		var imageBlock *types.ContentBlock
		var err error

		if imgMsg.ImageURL != "" {
			imageBlock, err = types.NewContentBlockFromURL(ctx, "image", imgMsg.ImageURL)
		} else if len(imgMsg.ImageData) > 0 {
			imageBlock, err = types.NewContentBlock("image", imgMsg.ImageData)
		} else {
			err = fmt.Errorf("ImageMessage has no image URL or data")
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("HandleMultimodalMessage: %w", err)
		}

		contentBlocks = append(contentBlocks, imageBlock)

		// Add text content if present
		if imgMsg.BaseMessage.Content != "" {
			textBlock, err := types.NewContentBlock("text", []byte(imgMsg.BaseMessage.Content))
			if err != nil {
				return nil, fmt.Errorf("HandleMultimodalMessage: %w", err)
			}
			contentBlocks = append(contentBlocks, textBlock)
		}
	}

	// Handle VideoMessage
	if vidMsg, ok := schema.AsVideoMessage(msg); ok {
		var videoBlock *types.ContentBlock
		var err error

		if vidMsg.VideoURL != "" {
			videoBlock, err = types.NewContentBlockFromURL(ctx, "video", vidMsg.VideoURL)
		} else if len(vidMsg.VideoData) > 0 {
			videoBlock, err = types.NewContentBlock("video", vidMsg.VideoData)
		} else {
			err = fmt.Errorf("VideoMessage has no video URL or data")
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("HandleMultimodalMessage: %w", err)
		}

		contentBlocks = append(contentBlocks, videoBlock)

		// Add text content if present
		if vidMsg.BaseMessage.Content != "" {
			textBlock, err := types.NewContentBlock("text", []byte(vidMsg.BaseMessage.Content))
			if err != nil {
				return nil, fmt.Errorf("HandleMultimodalMessage: %w", err)
			}
			contentBlocks = append(contentBlocks, textBlock)
		}
	}

	// Handle VoiceDocument (check if message is a VoiceDocument type)
	if voiceDoc, ok := msg.(*schema.VoiceDocument); ok {
		var audioBlock *types.ContentBlock
		var err error

		if voiceDoc.AudioURL != "" {
			audioBlock, err = types.NewContentBlockFromURL(ctx, "audio", voiceDoc.AudioURL)
		} else if len(voiceDoc.AudioData) > 0 {
			audioBlock, err = types.NewContentBlock("audio", voiceDoc.AudioData)
		} else {
			err = fmt.Errorf("VoiceDocument has no audio URL or data")
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("HandleMultimodalMessage: %w", err)
		}

		contentBlocks = append(contentBlocks, audioBlock)

		// Add transcript or text content if present
		textContent := voiceDoc.GetContent()
		if textContent != "" {
			textBlock, err := types.NewContentBlock("text", []byte(textContent))
			if err != nil {
				return nil, fmt.Errorf("HandleMultimodalMessage: %w", err)
			}
			contentBlocks = append(contentBlocks, textBlock)
		}
	}

	// Handle Document with voice metadata (fallback)
	if doc, ok := msg.(*schema.Document); ok && schema.IsVoiceDocument(*doc) {
		// Extract audio data from metadata
		audioURL := doc.Metadata["audio_url"]
		var audioData []byte
		// Note: audio_data would need to be base64 encoded in metadata for this to work
		// For now, we'll just use the URL
		audioFormat := doc.Metadata["audio_format"]
		if audioFormat == "" {
			audioFormat = "mp3" // Default format
		}

		var audioBlock *types.ContentBlock
		var err error

		if audioURL != "" {
			audioBlock, err = types.NewContentBlockFromURL(ctx, "audio", audioURL)
		} else if len(audioData) > 0 {
			audioBlock, err = types.NewContentBlock("audio", audioData)
		} else {
			err = fmt.Errorf("VoiceDocument has no audio URL or data")
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("HandleMultimodalMessage: %w", err)
		}

		contentBlocks = append(contentBlocks, audioBlock)

		// Add transcript or text content if present
		textContent := doc.GetContent()
		if textContent != "" {
			textBlock, err := types.NewContentBlock("text", []byte(textContent))
			if err != nil {
				return nil, fmt.Errorf("HandleMultimodalMessage: %w", err)
			}
			contentBlocks = append(contentBlocks, textBlock)
		}
	}

	// Handle regular text messages
	if len(contentBlocks) == 0 {
		textBlock, err := types.NewContentBlock("text", []byte(msg.GetContent()))
		if err != nil {
			return nil, fmt.Errorf("HandleMultimodalMessage: %w", err)
		}
		contentBlocks = append(contentBlocks, textBlock)
	}

	if len(contentBlocks) == 0 {
		return nil, fmt.Errorf("HandleMultimodalMessage: no content blocks created from message")
	}

	input, err := types.NewMultimodalInput(contentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("HandleMultimodalMessage: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Multimodal message handled successfully",
		"content_blocks_count", len(contentBlocks))

	return input, nil
}

// ProcessForAgent processes multimodal input or output for agent use and returns schema messages.
func (m *MultimodalAgentExtension) ProcessForAgent(ctx context.Context, inputOrOutput interface{}) ([]schema.Message, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.ProcessForAgent")
	defer span.End()
	
	var output *types.MultimodalOutput
	
	// Handle both MultimodalInput and MultimodalOutput
	switch v := inputOrOutput.(type) {
	case *types.MultimodalInput:
		span.SetAttributes(attribute.String("input_id", v.ID))
		
		// Process the multimodal input
		var err error
		output, err = m.model.Process(ctx, v)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("ProcessForAgent: %w", err)
		}
	case *types.MultimodalOutput:
		output = v
		span.SetAttributes(attribute.String("output_id", output.ID))
	default:
		err := fmt.Errorf("ProcessForAgent: expected *types.MultimodalInput or *types.MultimodalOutput, got %T", inputOrOutput)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Convert output to schema messages
	messages := make([]schema.Message, 0, len(output.ContentBlocks))
	for _, block := range output.ContentBlocks {
		switch block.Type {
		case "text":
			msg := schema.NewAIMessage(string(block.Data))
			messages = append(messages, msg)
		case "image":
			// Create ImageMessage from output
			imgMsg := &schema.ImageMessage{
				ImageData:   block.Data,
				ImageFormat: block.Format,
			}
			if block.URL != "" {
				imgMsg.ImageURL = block.URL
			}
			messages = append(messages, imgMsg)
		case "video":
			// Create VideoMessage from output
			vidMsg := &schema.VideoMessage{
				VideoData:   block.Data,
				VideoFormat: block.Format,
			}
			if block.URL != "" {
				vidMsg.VideoURL = block.URL
			}
			messages = append(messages, vidMsg)
		case "audio":
			// Create VoiceDocument from output
			voiceDoc := &schema.VoiceDocument{
				AudioData:   block.Data,
				AudioFormat: block.Format,
			}
			if block.URL != "" {
				voiceDoc.AudioURL = block.URL
			}
			// Convert to Document for agent use
			doc := schema.NewDocument(string(block.Data), map[string]string{
				"audio_url":    voiceDoc.AudioURL,
				"audio_format": voiceDoc.AudioFormat,
			})
			messages = append(messages, doc)
		}
	}

	span.SetStatus(codes.Ok, "")
	return messages, nil
}

// CreateMultimodalTool creates a tool that can handle multimodal inputs/outputs.
func (m *MultimodalAgentExtension) CreateMultimodalTool(name, description string, processFunc func(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error)) tools.Tool {
	return &multimodalTool{
		name:        name,
		description: description,
		extension:   m,
		processFunc: processFunc,
	}
}

// multimodalTool is a tool that processes multimodal inputs.
type multimodalTool struct {
	tools.BaseTool
	name        string
	description string
	extension   *MultimodalAgentExtension
	processFunc func(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error)
}

// Name returns the tool name.
func (t *multimodalTool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *multimodalTool) Description() string {
	return t.description
}

// Execute executes the multimodal tool.
func (t *multimodalTool) Execute(ctx context.Context, input any) (any, error) {
	// Convert input to MultimodalInput
	var multimodalInput *types.MultimodalInput

	switch v := input.(type) {
	case *types.MultimodalInput:
		multimodalInput = v
	case map[string]any:
		// Try to extract multimodal input from map
		// This is a simplified conversion - in production, would need more robust handling
		contentBlocks := make([]*types.ContentBlock, 0)
		if text, ok := v["text"].(string); ok {
			block, err := types.NewContentBlock("text", []byte(text))
			if err == nil {
				contentBlocks = append(contentBlocks, block)
			}
		}
		if len(contentBlocks) == 0 {
			return nil, fmt.Errorf("invalid input format for multimodal tool")
		}
		var err error
		multimodalInput, err = types.NewMultimodalInput(contentBlocks)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported input type for multimodal tool: %T", input)
	}

	// Process using the provided function or default processing
	var output *types.MultimodalOutput
	var err error

	if t.processFunc != nil {
		output, err = t.processFunc(ctx, multimodalInput)
	} else {
		output, err = t.extension.model.Process(ctx, multimodalInput)
	}

	if err != nil {
		return nil, err
	}

	// Convert output to a format suitable for agent use
	return map[string]any{
		"output_id":       output.ID,
		"content_blocks":  output.ContentBlocks,
		"confidence":      output.Confidence,
		"provider":        output.Provider,
		"model":           output.Model,
	}, nil
}

// PreserveMultimodalContext preserves multimodal context throughout agent workflows.
func (m *MultimodalAgentExtension) PreserveMultimodalContext(ctx context.Context, previousContext map[string]any, newInput *types.MultimodalInput) map[string]any {
	return m.model.PreserveContext(ctx, previousContext, newInput)
}

// ConvertAgentMessagesToMultimodalInput converts agent messages to multimodal input.
func (m *MultimodalAgentExtension) ConvertAgentMessagesToMultimodalInput(ctx context.Context, messages []schema.Message) (*types.MultimodalInput, error) {
	contentBlocks := make([]*types.ContentBlock, 0)

	for _, msg := range messages {
		input, err := m.HandleMultimodalMessage(ctx, msg)
		if err != nil {
			return nil, fmt.Errorf("ConvertAgentMessagesToMultimodalInput: %w", err)
		}
		contentBlocks = append(contentBlocks, input.ContentBlocks...)
	}

	if len(contentBlocks) == 0 {
		return nil, fmt.Errorf("ConvertAgentMessagesToMultimodalInput: no content blocks created from messages")
	}

	return types.NewMultimodalInput(contentBlocks)
}

// EnableVoiceReActLoop enables voice-enabled ReAct loops by processing audio inputs in the planning phase.
func (m *MultimodalAgentExtension) EnableVoiceReActLoop(ctx context.Context, agent iface.Agent, inputs map[string]any, intermediateSteps []iface.IntermediateStep) (iface.AgentAction, iface.AgentFinish, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.EnableVoiceReActLoop",
		trace.WithAttributes(
			attribute.String("agent_name", agent.GetConfig().Name),
			attribute.Int("intermediate_steps_count", len(intermediateSteps)),
		))
	defer span.End()

	// Check if inputs contain voice/audio data
	hasVoiceInput := false
	for _, v := range inputs {
		if msg, ok := v.(schema.Message); ok {
			if schema.HasMultimodalContent(msg) || schema.IsVoiceDocument(*msg.(*schema.Document)) {
				hasVoiceInput = true
				break
			}
		}
	}

	// If voice input detected, process it first
	if hasVoiceInput {
		messages := make([]schema.Message, 0)
		for _, v := range inputs {
			if msg, ok := v.(schema.Message); ok {
				messages = append(messages, msg)
			}
		}

		multimodalInput, err := m.ConvertAgentMessagesToMultimodalInput(ctx, messages)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("EnableVoiceReActLoop: %w", err)
		}

		// Process multimodal input
		output, err := m.model.Process(ctx, multimodalInput)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("EnableVoiceReActLoop: %w", err)
		}

		// Convert output to text for agent planning
		textOutput := ""
		for _, block := range output.ContentBlocks {
			if block.Type == "text" {
				textOutput += string(block.Data) + "\n"
			}
		}

		// Update inputs with processed text
		inputs["processed_voice_input"] = textOutput
	}

	// Call agent's Plan method
	action, finish, err := agent.Plan(ctx, intermediateSteps, inputs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return iface.AgentAction{}, iface.AgentFinish{}, err
	}

	span.SetStatus(codes.Ok, "")
	return action, finish, nil
}

// HandleOrchestrationGraphInput processes image inputs in orchestration graphs.
func (m *MultimodalAgentExtension) HandleOrchestrationGraphInput(ctx context.Context, input any) (any, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.HandleOrchestrationGraphInput")
	defer span.End()

	// Check if input contains multimodal data
	switch v := input.(type) {
	case map[string]any:
		// Check for image/video/audio data in map
		for key, val := range v {
			if msg, ok := val.(schema.Message); ok {
				if schema.HasMultimodalContent(msg) {
					// Process multimodal message
					multimodalInput, err := m.HandleMultimodalMessage(ctx, msg)
					if err != nil {
						span.RecordError(err)
						span.SetStatus(codes.Error, err.Error())
						return nil, fmt.Errorf("HandleOrchestrationGraphInput: %w", err)
					}

					output, err := m.model.Process(ctx, multimodalInput)
					if err != nil {
						span.RecordError(err)
						span.SetStatus(codes.Error, err.Error())
						return nil, fmt.Errorf("HandleOrchestrationGraphInput: %w", err)
					}

					// Replace input value with processed output
					v[key] = output
				}
			}
		}
		return v, nil
	case []schema.Message:
		// Process all messages
		multimodalInput, err := m.ConvertAgentMessagesToMultimodalInput(ctx, v)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("HandleOrchestrationGraphInput: %w", err)
		}

		output, err := m.model.Process(ctx, multimodalInput)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("HandleOrchestrationGraphInput: %w", err)
		}

		// Convert output back to messages
		return m.ProcessForAgent(ctx, output) // output is *types.MultimodalOutput
	case schema.Message:
		// Single message
		multimodalInput, err := m.HandleMultimodalMessage(ctx, v)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("HandleOrchestrationGraphInput: %w", err)
		}

		output, err := m.model.Process(ctx, multimodalInput)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("HandleOrchestrationGraphInput: %w", err)
		}

		// Convert output to messages
		messages, err := m.ProcessForAgent(ctx, output)
		if err != nil {
			return nil, err
		}
		if len(messages) > 0 {
			return messages[0], nil
		}
		return nil, fmt.Errorf("HandleOrchestrationGraphInput: no messages generated from output")
	default:
		// Return input as-is if not multimodal
		return input, nil
	}
}

// PreserveMultimodalDataInAgentCommunication preserves multimodal data when agents communicate.
func (m *MultimodalAgentExtension) PreserveMultimodalDataInAgentCommunication(ctx context.Context, fromAgent iface.Agent, toAgent iface.Agent, data any) (any, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.PreserveMultimodalDataInAgentCommunication",
		trace.WithAttributes(
			attribute.String("from_agent", fromAgent.GetConfig().Name),
			attribute.String("to_agent", toAgent.GetConfig().Name),
		))
	defer span.End()

	// Check if data contains multimodal content
	switch v := data.(type) {
	case *types.MultimodalInput:
		// Already multimodal input, preserve as-is
		return v, nil
	case *types.MultimodalOutput:
		// Convert output to input for next agent
		input, err := types.NewMultimodalInput(v.ContentBlocks)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("PreserveMultimodalDataInAgentCommunication: %w", err)
		}
		return input, nil
	case []schema.Message:
		// Convert messages to multimodal input
		input, err := m.ConvertAgentMessagesToMultimodalInput(ctx, v)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("PreserveMultimodalDataInAgentCommunication: %w", err)
		}
		return input, nil
	case schema.Message:
		// Single message
		input, err := m.HandleMultimodalMessage(ctx, v)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("PreserveMultimodalDataInAgentCommunication: %w", err)
		}
		return input, nil
	default:
		// Return data as-is if not multimodal
		return data, nil
	}
}
