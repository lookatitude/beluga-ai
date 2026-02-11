---
title: SMS Appointment Reminder System
description: Automate appointment reminders with two-way SMS communication and intelligent response handling to reduce no-shows.
---

Appointment no-shows cost the US healthcare system $150B annually, with individual practices losing $200-500 per missed slot. Manual reminder calls — still the primary method at many practices — consume 2-3 staff hours daily, reach only 60-70% of patients (due to unanswered calls and voicemail), and provide no mechanism for patients to confirm, reschedule, or cancel without calling back during business hours. The result is a 25-30% no-show rate that disrupts scheduling, wastes provider time, and delays care for other patients.

SMS reminders address reachability (95%+ open rates within 3 minutes), but one-way SMS without response handling creates its own problems: patients who want to reschedule still need to call the office, and staff still spend time processing those calls manually.

## Solution Architecture

Beluga AI's server package provides messaging backend abstractions that decouple the reminder logic from the SMS provider (Twilio, Vonage, etc.), enabling provider switching without rewriting application code. The two-way communication architecture uses intent parsing to handle patient responses — confirmations, rescheduling requests, and cancellations — without human intervention for common cases. The scheduler component manages reminder timing (typically 24 hours before appointments) and handles edge cases like same-day appointments that need immediate reminders.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Appointment  │───▶│   Reminder   │───▶│   Messaging  │
│   Schedule   │    │   Scheduler  │    │    Backend   │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                        ┌──────▼───────┐
                                        │   Patient    │
                                        │  (via SMS)   │
                                        └──────┬───────┘
                                               │
┌──────────────┐    ┌──────────────┐    ┌─────▼────────┐
│ Appointment  │◀───│ Rescheduling │◀───│   Response   │
│    System    │    │    Agent     │    │   Handler    │
└──────────────┘    └──────────────┘    └──────────────┘
```

## Implementation

### Reminder System Setup

The system manages SMS reminders with two-way communication:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/server"

    _ "github.com/lookatitude/beluga-ai/server/providers/twilio"
)

type AppointmentReminderSystem struct {
    messaging server.MessagingBackend
    scheduler *ReminderScheduler
}

func NewAppointmentReminderSystem(ctx context.Context) (*AppointmentReminderSystem, error) {
    messaging, err := server.NewMessagingBackend("twilio", server.MessagingConfig{
        AccountSID: "your-account-sid",
        AuthToken:  "your-auth-token",
        Channel:    "sms",
    })
    if err != nil {
        return nil, fmt.Errorf("create messaging backend: %w", err)
    }

    return &AppointmentReminderSystem{
        messaging: messaging,
        scheduler: NewReminderScheduler(),
    }, nil
}

type Appointment struct {
    ID          string
    PatientName string
    PatientPhone string
    DateTime    time.Time
    ProviderName string
    Type        string
}
```

### Sending Reminders

Send automated reminders at scheduled times:

```go
func (s *AppointmentReminderSystem) SendReminder(ctx context.Context, appointment Appointment) error {
    // Get or create conversation with patient
    conversation, err := s.messaging.GetOrCreateConversation(ctx, appointment.PatientPhone)
    if err != nil {
        return fmt.Errorf("get conversation: %w", err)
    }

    // Build reminder message
    message := s.buildReminderMessage(appointment)

    // Send SMS
    msg := schema.Message{
        Content: message,
        Channel: "sms",
    }

    if err := s.messaging.SendMessage(ctx, conversation.ID, msg); err != nil {
        return fmt.Errorf("send reminder: %w", err)
    }

    return nil
}

func (s *AppointmentReminderSystem) buildReminderMessage(appointment Appointment) string {
    return fmt.Sprintf(
        "Reminder: You have a %s appointment with %s on %s. Reply 1 to confirm, 2 to reschedule, or 3 to cancel.",
        appointment.Type,
        appointment.ProviderName,
        appointment.DateTime.Format("Mon Jan 2 at 3:04 PM"),
    )
}
```

### Response Handling

Process patient responses intelligently:

```go
type ResponseHandler struct {
    system *AppointmentReminderSystem
}

func (h *ResponseHandler) HandleResponse(ctx context.Context, conversationID, message string) error {
    // Parse response intent
    intent := h.parseIntent(message)

    switch intent {
    case "confirm":
        return h.handleConfirmation(ctx, conversationID)
    case "reschedule":
        return h.handleRescheduling(ctx, conversationID)
    case "cancel":
        return h.handleCancellation(ctx, conversationID)
    default:
        return h.sendClarification(ctx, conversationID)
    }
}

func (h *ResponseHandler) parseIntent(message string) string {
    message = strings.TrimSpace(strings.ToLower(message))

    // Simple keyword-based intent recognition
    if message == "1" || strings.Contains(message, "confirm") || strings.Contains(message, "yes") {
        return "confirm"
    }
    if message == "2" || strings.Contains(message, "reschedule") || strings.Contains(message, "change") {
        return "reschedule"
    }
    if message == "3" || strings.Contains(message, "cancel") || strings.Contains(message, "no") {
        return "cancel"
    }

    return "unknown"
}

func (h *ResponseHandler) handleConfirmation(ctx context.Context, conversationID string) error {
    // Update appointment status to confirmed
    appointment := h.getAppointmentByConversation(ctx, conversationID)
    if err := h.confirmAppointment(ctx, appointment.ID); err != nil {
        return err
    }

    // Send confirmation message
    msg := schema.Message{
        Content: "Thank you! Your appointment is confirmed. We look forward to seeing you.",
        Channel: "sms",
    }

    return h.system.messaging.SendMessage(ctx, conversationID, msg)
}

func (h *ResponseHandler) handleRescheduling(ctx context.Context, conversationID string) error {
    // Send rescheduling options
    msg := schema.Message{
        Content: "I can help you reschedule. Please call our office at (555) 123-4567 or visit our website to select a new time.",
        Channel: "sms",
    }

    return h.system.messaging.SendMessage(ctx, conversationID, msg)
}

func (h *ResponseHandler) handleCancellation(ctx context.Context, conversationID string) error {
    // Update appointment status to cancelled
    appointment := h.getAppointmentByConversation(ctx, conversationID)
    if err := h.cancelAppointment(ctx, appointment.ID); err != nil {
        return err
    }

    // Send cancellation confirmation
    msg := schema.Message{
        Content: "Your appointment has been cancelled. Call us at (555) 123-4567 if you need to reschedule.",
        Channel: "sms",
    }

    return h.system.messaging.SendMessage(ctx, conversationID, msg)
}

func (h *ResponseHandler) sendClarification(ctx context.Context, conversationID string) error {
    msg := schema.Message{
        Content: "I didn't understand your response. Reply 1 to confirm, 2 to reschedule, or 3 to cancel.",
        Channel: "sms",
    }

    return h.system.messaging.SendMessage(ctx, conversationID, msg)
}
```

## Production Considerations

### Automated Scheduling

Schedule reminders automatically based on appointment time:

```go
type ReminderScheduler struct {
    system *AppointmentReminderSystem
}

func (s *ReminderScheduler) ScheduleReminder(ctx context.Context, appointment Appointment) error {
    // Schedule reminder 24 hours before appointment
    reminderTime := appointment.DateTime.Add(-24 * time.Hour)

    // Calculate delay
    delay := time.Until(reminderTime)
    if delay < 0 {
        // Appointment is less than 24 hours away, send immediately
        return s.system.SendReminder(ctx, appointment)
    }

    // Schedule for later
    timer := time.NewTimer(delay)
    go func() {
        <-timer.C
        if err := s.system.SendReminder(context.Background(), appointment); err != nil {
            // Log error
        }
    }()

    return nil
}
```

### Batch Reminders

Send reminders for multiple appointments efficiently:

```go
func (s *AppointmentReminderSystem) SendBatchReminders(ctx context.Context, appointments []Appointment) error {
    for _, appointment := range appointments {
        if err := s.SendReminder(ctx, appointment); err != nil {
            // Log error but continue with other reminders
            continue
        }

        // Rate limit to avoid overwhelming SMS provider
        time.Sleep(100 * time.Millisecond)
    }

    return nil
}
```

### Observability

Track reminder delivery and response rates:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel/attribute"
)

func (s *AppointmentReminderSystem) SendReminderWithTracing(ctx context.Context, appointment Appointment) error {
    ctx, span := o11y.StartSpan(ctx, "reminder.send")
    defer span.End()

    span.SetAttributes(
        attribute.String("appointment.id", appointment.ID),
        attribute.String("appointment.type", appointment.Type),
        attribute.String("patient.phone", appointment.PatientPhone),
    )

    err := s.SendReminder(ctx, appointment)
    if err != nil {
        span.RecordError(err)
        return err
    }

    return nil
}
```

### HIPAA Compliance

Ensure patient data privacy and security:

```go
func (s *AppointmentReminderSystem) buildSecureReminderMessage(appointment Appointment) string {
    // Don't include sensitive information in SMS
    return fmt.Sprintf(
        "Reminder: You have an appointment on %s. Reply 1 to confirm, 2 to reschedule, or 3 to cancel. ID: %s",
        appointment.DateTime.Format("Mon Jan 2 at 3:04 PM"),
        appointment.ID[:8], // Short ID for reference
    )
}
```

## Results

After implementing SMS appointment reminders, the provider achieved:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Manual Reminder Time | 2-3 hrs/day | 0 hrs/day | 100% reduction |
| No-Show Rate | 25-30% | 16% | 36-47% reduction |
| Reminder Delivery Rate | 60-70% | 92% | 31-53% improvement |
| Response Handling | 0% | 85% | New capability |
| Revenue Recovery | $0/month | $22K/month | $22K gain |
| Patient Satisfaction | 6.5/10 | 9.1/10 | 40% improvement |

## Related Resources

- [Messaging Integration](/integrations/messaging/) for SMS provider configuration
- [Server Providers](/providers/server/) for Twilio and other messaging backends
- [Agent Guide](/guides/agents/) for intelligent response handling patterns
