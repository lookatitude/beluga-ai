# Event Routing with Prefix Matching

Route webhook events using prefix-based matching.

```go
switch {
case strings.HasPrefix(event.EventType, "conversation."):
    return p.HandleConversationEvent(ctx, event)
case strings.HasPrefix(event.EventType, "message."):
    return p.HandleMessageEvent(ctx, event)
case strings.HasPrefix(event.EventType, "participant."):
    return p.HandleParticipantEvent(ctx, event)
case strings.HasPrefix(event.EventType, "typing."):
    return p.HandleTypingEvent(ctx, event)
default:
    return nil  // Graceful degradation for unknown events
}
```

## Benefits
- Single handler manages multiple related event subtypes
- New subtypes (e.g., `message.delivered`) work without code changes
- Unknown events silently succeed (no error)

## Event Type Conventions
- Format: `{resource}.{action}` (e.g., `conversation.created`, `message.added`)
- Group by resource prefix for routing
- Subtype handlers dispatch to specific `handle{Resource}{Action}Event` functions

## When to Use
- Webhook handlers receiving diverse event types
- Extensible APIs where new event types may be added
- NOT when exact event matching is required for security
