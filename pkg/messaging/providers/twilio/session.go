package twilio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	messagingiface "github.com/lookatitude/beluga-ai/pkg/messaging/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// MessagingSession manages conversation state and agent integration for messaging.
type MessagingSession struct {
	startTime           time.Time
	lastActivity        time.Time
	agentInstance       agentsiface.Agent
	memoryInstance      memoryiface.Memory
	config              *TwilioConfig
	provider            *TwilioProvider
	agentCallback       func(context.Context, string) (string, error)
	metadata            map[string]any
	conversationSID     string
	participantIdentity string
	id                  string
	participants        []*messagingiface.Participant
	channels            []string
	conversationHistory []*messagingiface.Message
	mu                  sync.RWMutex
	active              bool
}

// NewMessagingSession creates a new messaging session.
func NewMessagingSession(
	conversationSID string,
	config *TwilioConfig,
	provider *TwilioProvider,
) (*MessagingSession, error) {
	sessionID := uuid.New().String()

	return &MessagingSession{
		id:                  sessionID,
		conversationSID:     conversationSID,
		config:              config,
		provider:            provider,
		conversationHistory: make([]*messagingiface.Message, 0),
		participants:        make([]*messagingiface.Participant, 0),
		channels:            make([]string, 0),
		metadata:            make(map[string]any),
		active:              false,
	}, nil
}

// Start starts the messaging session.
func (s *MessagingSession) Start(ctx context.Context) error {
	ctx, span := s.startSpan(ctx, "MessagingSession.Start")
	defer span.End()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		span.SetStatus(codes.Ok, "already active")
		return nil
	}

	s.active = true
	s.startTime = time.Now()
	s.lastActivity = time.Now()

	// Initialize memory instance if not set
	if s.memoryInstance == nil {
		// In a full implementation, this would create a VectorStoreMemory instance
		// For now, we'll leave it as nil
	}

	span.SetStatus(codes.Ok, "session started")
	return nil
}

// Stop stops the messaging session.
func (s *MessagingSession) Stop(ctx context.Context) error {
	ctx, span := s.startSpan(ctx, "MessagingSession.Stop")
	defer span.End()

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		span.SetStatus(codes.Ok, "already stopped")
		return nil
	}

	s.active = false
	s.lastActivity = time.Now()

	// Save conversation history to memory
	if s.memoryInstance != nil {
		if err := s.saveConversationHistory(ctx); err != nil {
			span.RecordError(err)
		}
	}

	span.SetStatus(codes.Ok, "session stopped")
	return nil
}

// ProcessMessage processes an incoming message through the agent.
func (s *MessagingSession) ProcessMessage(ctx context.Context, message *messagingiface.Message) error {
	ctx, span := s.startSpan(ctx, "MessagingSession.ProcessMessage")
	defer span.End()

	s.mu.Lock()
	s.lastActivity = time.Now()
	s.conversationHistory = append(s.conversationHistory, message)
	s.mu.Unlock()

	span.SetAttributes(
		attribute.String("message_sid", message.MessageSID),
		attribute.String("channel", string(message.Channel)),
	)

	// Save to memory
	if s.memoryInstance != nil {
		inputs := map[string]any{
			"message": message.Body,
			"channel": message.Channel,
			"from":    message.From,
		}
		if err := s.memoryInstance.SaveContext(ctx, inputs, nil); err != nil {
			span.RecordError(err)
		}
	}

	// Process through agent
	var response string
	var err error

	s.mu.RLock()
	agentInstance := s.agentInstance
	agentCallback := s.agentCallback
	s.mu.RUnlock()

	if agentInstance != nil {
		// Use agent instance (full implementation would use executor)
		if agentCallback != nil {
			response, err = agentCallback(ctx, message.Body)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}
		}
	} else if agentCallback != nil {
		response, err = agentCallback(ctx, message.Body)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	// Send response message
	if response != "" {
		responseMessage := &messagingiface.Message{
			ConversationSID: s.conversationSID,
			Body:            response,
			Channel:         message.Channel, // Use same channel
			From:            message.To,      // Swap from/to
			To:              message.From,
			DateCreated:     time.Now(),
		}

		if err := s.provider.SendMessage(ctx, s.conversationSID, responseMessage); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		// Save response to history
		s.mu.Lock()
		s.conversationHistory = append(s.conversationHistory, responseMessage)
		s.mu.Unlock()

		// Save to memory
		if s.memoryInstance != nil {
			outputs := map[string]any{
				"response": response,
			}
			if err := s.memoryInstance.SaveContext(ctx, nil, outputs); err != nil {
				span.RecordError(err)
			}
		}
	}

	span.SetStatus(codes.Ok, "message processed")
	return nil
}

// GetConversationHistory returns the conversation history.
func (s *MessagingSession) GetConversationHistory(ctx context.Context) ([]*messagingiface.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Load from memory if available
	if s.memoryInstance != nil {
		variables, err := s.memoryInstance.LoadMemoryVariables(ctx, map[string]any{})
		if err == nil {
			// Extract messages from memory variables
			// In a full implementation, this would reconstruct messages from memory
		}
		_ = variables
	}

	return s.conversationHistory, nil
}

// SetAgentInstance sets the agent instance.
func (s *MessagingSession) SetAgentInstance(agent agentsiface.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agentInstance = agent
	return nil
}

// SetAgentCallback sets the agent callback function.
func (s *MessagingSession) SetAgentCallback(callback func(context.Context, string) (string, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agentCallback = callback
	return nil
}

// SetMemoryInstance sets the memory instance for conversation persistence.
func (s *MessagingSession) SetMemoryInstance(memory memoryiface.Memory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.memoryInstance = memory
	return nil
}

// LinkParticipantIdentity links this session to a participant identity for multi-channel context preservation.
func (s *MessagingSession) LinkParticipantIdentity(identity string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.participantIdentity = identity
}

// GetParticipantIdentity returns the linked participant identity.
func (s *MessagingSession) GetParticipantIdentity() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.participantIdentity
}

// saveConversationHistory saves conversation history to memory.
func (s *MessagingSession) saveConversationHistory(ctx context.Context) error {
	if s.memoryInstance == nil {
		return nil
	}

	// Convert messages to schema messages for memory storage
	messages := make([]schema.Message, 0, len(s.conversationHistory))
	for _, msg := range s.conversationHistory {
		// Determine message type based on direction
		// In a full implementation, this would use proper message type detection
		schemaMsg := schema.NewHumanMessage(msg.Body)
		messages = append(messages, schemaMsg)
	}

	// Save to memory
	inputs := map[string]any{
		"conversation_sid":     s.conversationSID,
		"messages":             messages,
		"participant_identity": s.participantIdentity,
	}

	return s.memoryInstance.SaveContext(ctx, inputs, nil)
}

// HandleLongConversation handles very long conversations exceeding memory limits (edge case).
func (s *MessagingSession) HandleLongConversation(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If conversation is too long, summarize or truncate while maintaining key context
	maxMessages := 1000 // Configurable limit

	if len(s.conversationHistory) > maxMessages {
		// Keep recent messages and summarize older ones
		recentMessages := s.conversationHistory[len(s.conversationHistory)-maxMessages:]

		// Summarize older messages (in full implementation, would use LLM for summarization)
		summary := fmt.Sprintf("Previous conversation with %d messages", len(s.conversationHistory)-maxMessages)

		// Replace history with summary + recent messages
		summaryMessage := &messagingiface.Message{
			Body:        summary,
			DateCreated: time.Now(),
		}

		s.conversationHistory = append([]*messagingiface.Message{summaryMessage}, recentMessages...)
	}

	return nil
}

// startSpan starts an OTEL span for tracing.
func (s *MessagingSession) startSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	if s.provider.metrics != nil && s.provider.metrics.Tracer() != nil {
		return s.provider.metrics.Tracer().Start(ctx, operation,
			trace.WithAttributes(
				attribute.String("session_id", s.id),
				attribute.String("conversation_sid", s.conversationSID),
			))
	}
	return ctx, trace.SpanFromContext(ctx)
}
