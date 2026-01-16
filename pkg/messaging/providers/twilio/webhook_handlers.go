package twilio

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// HandleConversationEvent handles conversation.* webhook events.
func (p *TwilioProvider) HandleConversationEvent(ctx context.Context, event *iface.WebhookEvent) error {
	ctx, span := p.startSpan(ctx, "TwilioProvider.HandleConversationEvent")
	defer span.End()

	span.SetAttributes(attribute.String("event_type", event.EventType))

	switch event.EventType {
	case "conversation.created":
		return p.handleConversationCreatedEvent(ctx, event)
	case "conversation.updated":
		return p.handleConversationUpdatedEvent(ctx, event)
	default:
		span.SetStatus(codes.Ok, "event handled")
		return nil
	}
}

// HandleMessageEvent handles message.* webhook events.
func (p *TwilioProvider) HandleMessageEvent(ctx context.Context, event *iface.WebhookEvent) error {
	ctx, span := p.startSpan(ctx, "TwilioProvider.HandleMessageEvent")
	defer span.End()

	span.SetAttributes(attribute.String("event_type", event.EventType))

	switch event.EventType {
	case "message.added":
		// Trigger message flow workflow if orchestration is enabled
		// Note: Orchestrator would come from base Config, not TwilioConfig
		// For now, skip orchestration integration - would need to get from p.config.Config.Orchestrator
		if false {
			var orchestrator interface{} // Would be from p.config.Config.Orchestrator
			orchestrationMgr, err := NewOrchestrationManager(p, orchestrator)
			if err == nil {
				return orchestrationMgr.TriggerMessageFlowWorkflow(ctx, event)
			}
		}
		return p.handleMessageAddedEvent(ctx, event)
	case "message.updated":
		return p.handleMessageUpdatedEvent(ctx, event)
	case "message.delivery.updated":
		return p.handleMessageDeliveryUpdatedEvent(ctx, event)
	default:
		span.SetStatus(codes.Ok, "event handled")
		return nil
	}
}

// HandleParticipantEvent handles participant.* webhook events.
func (p *TwilioProvider) HandleParticipantEvent(ctx context.Context, event *iface.WebhookEvent) error {
	ctx, span := p.startSpan(ctx, "TwilioProvider.HandleParticipantEvent")
	defer span.End()

	span.SetAttributes(attribute.String("event_type", event.EventType))

	switch event.EventType {
	case "participant.added":
		return p.handleParticipantAddedEvent(ctx, event)
	case "participant.removed":
		return p.handleParticipantRemovedEvent(ctx, event)
	default:
		span.SetStatus(codes.Ok, "event handled")
		return nil
	}
}

// HandleTypingEvent handles typing.* webhook events.
func (p *TwilioProvider) HandleTypingEvent(ctx context.Context, event *iface.WebhookEvent) error {
	ctx, span := p.startSpan(ctx, "TwilioProvider.HandleTypingEvent")
	defer span.End()

	span.SetAttributes(attribute.String("event_type", event.EventType))

	// Handle typing indicator
	return p.handleTypingStartedEvent(ctx, event)
}

// handleMessageUpdatedEvent handles message.updated webhook events.
func (p *TwilioProvider) handleMessageUpdatedEvent(ctx context.Context, event *iface.WebhookEvent) error {
	// Update message status
	return nil
}
