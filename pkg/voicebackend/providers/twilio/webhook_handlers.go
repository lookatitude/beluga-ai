package twilio

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// HandleCallStatusEvent handles call.status webhook events.
func (b *TwilioBackend) HandleCallStatusEvent(ctx context.Context, event *WebhookEvent) error {
	ctx, span := b.startSpan(ctx, "TwilioBackend.HandleCallStatusEvent")
	defer span.End()

	span.SetAttributes(
		attribute.String("event_type", event.EventType),
		attribute.String("call_sid", event.ResourceSID),
	)

	callStatus := getStringFromEvent(event.EventData, "CallStatus")
	span.SetAttributes(attribute.String("call_status", callStatus))

	// Route to appropriate handler based on status
	switch callStatus {
	case "answered":
		// Trigger call flow workflow if orchestration is enabled
		if b.config.Orchestrator != nil {
			orchestrationMgr, err := NewOrchestrationManager(b, b.config.Orchestrator)
			if err == nil {
				return orchestrationMgr.TriggerCallFlowWorkflow(ctx, event)
			}
		}
		return b.handleCallEvent(ctx, event)
	case "completed", "failed", "busy", "no-answer", "canceled":
		return b.handleCallEvent(ctx, event)
	default:
		span.SetStatus(codes.Ok, "event handled")
		return nil
	}
}

// HandleStreamEvent handles stream.event webhook events.
func (b *TwilioBackend) HandleStreamEvent(ctx context.Context, event *WebhookEvent) error {
	ctx, span := b.startSpan(ctx, "TwilioBackend.HandleStreamEvent")
	defer span.End()

	span.SetAttributes(
		attribute.String("event_type", event.EventType),
		attribute.String("stream_sid", getStringFromEvent(event.EventData, "StreamSid")),
	)

	// Handle stream events (start, stop, etc.)
	streamSID := getStringFromEvent(event.EventData, "StreamSid")
	streamStatus := getStringFromEvent(event.EventData, "Status")

	span.SetAttributes(attribute.String("stream_status", streamStatus))

	// Update stream state if needed
	_ = streamSID
	_ = streamStatus

	span.SetStatus(codes.Ok, "stream event handled")
	return nil
}

// HandleTranscriptionCompletedEvent handles transcription.completed webhook events.
func (b *TwilioBackend) HandleTranscriptionCompletedEvent(ctx context.Context, event *WebhookEvent) error {
	ctx, span := b.startSpan(ctx, "TwilioBackend.HandleTranscriptionCompletedEvent")
	defer span.End()

	span.SetAttributes(
		attribute.String("event_type", event.EventType),
		attribute.String("transcription_sid", getStringFromEvent(event.EventData, "TranscriptionSid")),
		attribute.String("call_sid", getStringFromEvent(event.EventData, "CallSid")),
	)

	// Process transcription (T096)
	transcriptionSID := getStringFromEvent(event.EventData, "TranscriptionSid")
	_ = getStringFromEvent(event.EventData, "CallSid") // CallSID available in transcription metadata

	// Create transcription manager
	transcriptionMgr := NewTranscriptionManager(b)

	// Retrieve transcription from Twilio
	transcription, err := transcriptionMgr.RetrieveTranscription(ctx, transcriptionSID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Store transcription for RAG
	if err := transcriptionMgr.StoreTranscription(ctx, transcription); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "transcription processed and stored")
	return nil
}

// getStringFromEvent extracts a string value from event data.
func getStringFromEvent(eventData map[string]any, key string) string {
	if value, ok := eventData[key].(string); ok {
		return value
	}
	if value, ok := eventData[key]; ok {
		return fmt.Sprintf("%v", value)
	}
	return ""
}
