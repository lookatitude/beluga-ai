package memory

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

const (
	// MetaKeySignature is the metadata key used to store the HMAC-SHA256
	// signature on memory entries.
	MetaKeySignature = "_beluga_sig"
)

// SignedMemoryMiddleware is a memory.Middleware that signs entries on Save with
// HMAC-SHA256 and verifies signatures on Load, ensuring memory integrity.
type SignedMemoryMiddleware struct {
	key   []byte
	hooks Hooks
}

// NewSignedMemoryMiddleware creates a middleware that signs and verifies memory
// entries using the provided HMAC key. The key must not be empty.
func NewSignedMemoryMiddleware(key []byte, opts ...SignedOption) (*SignedMemoryMiddleware, error) {
	if len(key) == 0 {
		return nil, core.NewError("guard/memory.NewSignedMemoryMiddleware", core.ErrInvalidInput, "HMAC key must not be empty", nil)
	}
	s := &SignedMemoryMiddleware{
		key: append([]byte(nil), key...), // defensive copy
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// SignedOption configures a SignedMemoryMiddleware.
type SignedOption func(*SignedMemoryMiddleware)

// WithSignedHooks sets hooks for the signing middleware.
func WithSignedHooks(h Hooks) SignedOption {
	return func(s *SignedMemoryMiddleware) {
		s.hooks = h
	}
}

// Wrap returns a memory.Middleware that wraps a Memory with signing and
// verification.
func (s *SignedMemoryMiddleware) Wrap(next memory.Memory) memory.Memory {
	return &signedMemory{next: next, key: s.key, hooks: s.hooks}
}

// signedMemory wraps a Memory to add HMAC signing on Save and verification
// on Load.
type signedMemory struct {
	next  memory.Memory
	key   []byte
	hooks Hooks
}

// Compile-time check.
var _ memory.Memory = (*signedMemory)(nil)

// Save signs the output message content and stores the signature in metadata,
// then delegates to the wrapped Memory.
func (m *signedMemory) Save(ctx context.Context, input, output schema.Message) error {
	// Sign the output content.
	content := extractMessageText(output)
	sig := computeHMAC(m.key, content)

	// Attach signature to output metadata.
	output = withSignature(output, sig)

	return m.next.Save(ctx, input, output)
}

// Load retrieves messages from the wrapped Memory and verifies signatures.
// Messages with invalid signatures are filtered out and the OnSignatureInvalid
// hook is called for each.
func (m *signedMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	msgs, err := m.next.Load(ctx, query)
	if err != nil {
		return nil, err
	}

	verified := make([]schema.Message, 0, len(msgs))
	for _, msg := range msgs {
		meta := msg.GetMetadata()
		sig, ok := meta[MetaKeySignature].(string)
		if !ok {
			// Unsigned message — skip it.
			if m.hooks.OnSignatureInvalid != nil {
				m.hooks.OnSignatureInvalid(ctx, "missing signature")
			}
			continue
		}

		content := extractMessageText(msg)
		if !verifyHMAC(m.key, content, sig) {
			if m.hooks.OnSignatureInvalid != nil {
				m.hooks.OnSignatureInvalid(ctx, "signature mismatch")
			}
			continue
		}

		verified = append(verified, msg)
	}

	return verified, nil
}

// Search delegates directly to the wrapped Memory.
func (m *signedMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return m.next.Search(ctx, query, k)
}

// Clear delegates directly to the wrapped Memory.
func (m *signedMemory) Clear(ctx context.Context) error {
	return m.next.Clear(ctx)
}

// computeHMAC produces an HMAC-SHA256 hex digest for the given content.
func computeHMAC(key []byte, content string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(content))
	return hex.EncodeToString(mac.Sum(nil))
}

// verifyHMAC checks that the hex-encoded signature matches the HMAC of content.
func verifyHMAC(key []byte, content, signature string) bool {
	expected := computeHMAC(key, content)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// extractMessageText extracts text from a message for signing purposes.
func extractMessageText(msg schema.Message) string {
	parts := msg.GetContent()
	var text string
	for _, p := range parts {
		if tp, ok := p.(schema.TextPart); ok {
			text += tp.Text
		}
	}
	return text
}

// withSignature returns a new message with the signature added to metadata.
// It creates a shallow copy of the message with updated metadata.
func withSignature(msg schema.Message, sig string) schema.Message {
	meta := msg.GetMetadata()
	newMeta := make(map[string]any, len(meta)+1)
	for k, v := range meta {
		newMeta[k] = v
	}
	newMeta[MetaKeySignature] = sig

	switch m := msg.(type) {
	case *schema.HumanMessage:
		return &schema.HumanMessage{Parts: m.Parts, Metadata: newMeta}
	case *schema.AIMessage:
		return &schema.AIMessage{
			Parts:     m.Parts,
			ToolCalls: m.ToolCalls,
			Usage:     m.Usage,
			ModelID:   m.ModelID,
			Metadata:  newMeta,
		}
	case *schema.SystemMessage:
		return &schema.SystemMessage{Parts: m.Parts, Metadata: newMeta}
	case *schema.ToolMessage:
		return &schema.ToolMessage{
			ToolCallID: m.ToolCallID,
			Parts:      m.Parts,
			Metadata:   newMeta,
		}
	default:
		// For unknown message types, return as-is. This is a defensive path
		// that should not be reached with standard schema types.
		return msg
	}
}

// MessageMiddleware returns this as a memory.Middleware function suitable for
// use with memory.ApplyMiddleware.
func (s *SignedMemoryMiddleware) MessageMiddleware() memory.Middleware {
	return func(next memory.Memory) memory.Memory {
		return &signedMemory{next: next, key: s.key, hooks: s.hooks}
	}
}

// Sign computes and returns the HMAC-SHA256 signature for the given content.
// This is exported for testing and external verification use cases.
func (s *SignedMemoryMiddleware) Sign(content string) string {
	return computeHMAC(s.key, content)
}

// Verify checks whether the given signature is valid for the content.
// This is exported for testing and external verification use cases.
func (s *SignedMemoryMiddleware) Verify(content, signature string) bool {
	return verifyHMAC(s.key, content, signature)
}
