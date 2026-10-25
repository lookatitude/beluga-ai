package memory

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/memory"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

const (
	// MetaKeySignature is the metadata key used to store the HMAC-SHA256
	// signature on memory entries.
	MetaKeySignature = "_beluga_sig"

	// MetaKeySignatureVersion is the metadata key holding the signature
	// scheme version. Version 2 signs a canonical encoding of the role, text,
	// tool calls and tool-call ID. Entries without a version were signed by
	// the legacy text-only scheme.
	MetaKeySignatureVersion = "_beluga_sig_v"

	// signatureVersion is the current signature scheme version.
	signatureVersion = "2"
)

// SignedMemoryMiddleware is a memory.Middleware that signs entries on Save with
// HMAC-SHA256 and verifies signatures on Load and Search, ensuring memory
// integrity.
//
// Both the input and the output message of every Save are signed. The MAC
// covers the message role, its text, its tool calls (ID, name, arguments)
// and, for tool messages, the tool-call ID, so a stored message cannot be
// relabelled (for example from "tool" to "system") or have its tool calls
// altered without failing verification.
//
// Search results are documents. A document is returned only when it carries
// a valid signature produced by SignDocument; unsigned or tampered documents
// are dropped and OnSignatureInvalid fires. Use WithUnverifiedSearch to opt
// out when the document tier is trusted separately.
type SignedMemoryMiddleware struct {
	key   []byte
	hooks Hooks
	// acceptLegacy accepts entries signed by the pre-v2 text-only scheme.
	acceptLegacy bool
	// unverifiedSearch passes Search results through without verification.
	unverifiedSearch bool
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

// WithLegacySignatures makes Load also accept entries signed by the legacy
// (pre-v2) scheme, which covered only the message text. It exists to migrate
// existing stores: legacy entries do not bind the role or tool calls, so a
// relabelled legacy entry still verifies. Re-save history and drop this
// option as soon as possible.
func WithLegacySignatures() SignedOption {
	return func(s *SignedMemoryMiddleware) {
		s.acceptLegacy = true
	}
}

// WithUnverifiedSearch makes Search return documents from the wrapped Memory
// without signature verification. Use it only when the document tier is
// protected by other means.
func WithUnverifiedSearch() SignedOption {
	return func(s *SignedMemoryMiddleware) {
		s.unverifiedSearch = true
	}
}

// Wrap returns a memory.Middleware that wraps a Memory with signing and
// verification.
func (s *SignedMemoryMiddleware) Wrap(next memory.Memory) memory.Memory {
	return s.newSigned(next)
}

func (s *SignedMemoryMiddleware) newSigned(next memory.Memory) *signedMemory {
	return &signedMemory{
		next:             next,
		key:              s.key,
		hooks:            s.hooks,
		acceptLegacy:     s.acceptLegacy,
		unverifiedSearch: s.unverifiedSearch,
	}
}

// signedMemory wraps a Memory to add HMAC signing on Save and verification
// on Load and Search.
type signedMemory struct {
	next             memory.Memory
	key              []byte
	hooks            Hooks
	acceptLegacy     bool
	unverifiedSearch bool
}

// Compile-time check.
var _ memory.Memory = (*signedMemory)(nil)

// Save signs both the input and the output message, stores the signatures
// in their metadata, then delegates to the wrapped Memory.
func (m *signedMemory) Save(ctx context.Context, input, output schema.Message) error {
	signedIn, err := m.sign(input)
	if err != nil {
		return err
	}
	signedOut, err := m.sign(output)
	if err != nil {
		return err
	}
	return m.next.Save(ctx, signedIn, signedOut)
}

// sign attaches a v2 signature to msg. A nil message is passed through.
func (m *signedMemory) sign(msg schema.Message) (schema.Message, error) {
	if msg == nil {
		return nil, nil
	}
	payload, err := canonicalMessage(msg)
	if err != nil {
		return nil, err
	}
	return withSignature(msg, computeHMAC(m.key, payload))
}

// Load retrieves messages from the wrapped Memory and verifies signatures.
// Messages with missing or invalid signatures are filtered out and the
// OnSignatureInvalid hook is called for each.
func (m *signedMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	msgs, err := m.next.Load(ctx, query)
	if err != nil {
		return nil, err
	}

	verified := make([]schema.Message, 0, len(msgs))
	for _, msg := range msgs {
		if reason := m.verifyMessage(msg); reason != "" {
			m.invalid(ctx, reason)
			continue
		}
		verified = append(verified, msg)
	}

	return verified, nil
}

// verifyMessage returns "" when msg carries a valid signature, otherwise the
// reason it was rejected.
func (m *signedMemory) verifyMessage(msg schema.Message) string {
	if msg == nil {
		return "missing signature"
	}
	meta := msg.GetMetadata()
	sig, ok := meta[MetaKeySignature].(string)
	if !ok {
		return "missing signature"
	}
	version, _ := meta[MetaKeySignatureVersion].(string)
	switch {
	case version == signatureVersion:
		payload, err := canonicalMessage(msg)
		if err != nil || !verifyHMAC(m.key, payload, sig) {
			return "signature mismatch"
		}
	case version == "" && m.acceptLegacy:
		if !verifyHMAC(m.key, extractMessageText(msg), sig) {
			return "signature mismatch"
		}
	case version == "":
		return "legacy signature rejected"
	default:
		return "unknown signature version"
	}
	return ""
}

func (m *signedMemory) invalid(ctx context.Context, reason string) {
	if m.hooks.OnSignatureInvalid != nil {
		m.hooks.OnSignatureInvalid(ctx, reason)
	}
}

// Search delegates to the wrapped Memory and returns only documents carrying
// a valid SignDocument signature, unless WithUnverifiedSearch was set.
func (m *signedMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	docs, err := m.next.Search(ctx, query, k)
	if err != nil || m.unverifiedSearch {
		return docs, err
	}
	verified := make([]schema.Document, 0, len(docs))
	for _, doc := range docs {
		sig, ok := doc.Metadata[MetaKeySignature].(string)
		if !ok {
			m.invalid(ctx, "missing signature")
			continue
		}
		if !verifyHMAC(m.key, canonicalDocument(doc), sig) {
			m.invalid(ctx, "signature mismatch")
			continue
		}
		verified = append(verified, doc)
	}
	return verified, nil
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

// signedToolCall is the canonical form of a tool call inside a MAC payload.
type signedToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// signedPayload is the canonical, versioned MAC input for a message.
type signedPayload struct {
	Version    string           `json:"v"`
	Role       string           `json:"role"`
	Text       string           `json:"text"`
	ToolCalls  []signedToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

// canonicalMessage returns the v2 MAC input for msg.
func canonicalMessage(msg schema.Message) (string, error) {
	p := signedPayload{
		Version: signatureVersion,
		Role:    string(msg.GetRole()),
		Text:    extractMessageText(msg),
	}
	switch mm := msg.(type) {
	case *schema.AIMessage:
		for _, tc := range mm.ToolCalls {
			p.ToolCalls = append(p.ToolCalls, signedToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
		}
	case *schema.ToolMessage:
		p.ToolCallID = mm.ToolCallID
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", core.NewError("guard/memory.canonicalMessage", core.ErrInvalidInput, "encode signature payload", err)
	}
	return string(b), nil
}

// canonicalDocument returns the MAC input for a document: its ID, content
// and "role" metadata (when present), in a versioned JSON encoding.
func canonicalDocument(doc schema.Document) string {
	role, _ := doc.Metadata["role"].(string)
	b, _ := json.Marshal(struct {
		Version string `json:"v"`
		Kind    string `json:"kind"`
		ID      string `json:"id"`
		Role    string `json:"role,omitempty"`
		Content string `json:"content"`
	}{Version: signatureVersion, Kind: "document", ID: doc.ID, Role: role, Content: doc.Content})
	return string(b)
}

// withSignature returns a new message with the signature added to metadata.
// It creates a shallow copy of the message with updated metadata. An error is
// returned for unknown message implementations so the caller can fail fast
// rather than silently producing an unsigned round-trip that will later be
// dropped by Load.
func withSignature(msg schema.Message, sig string) (schema.Message, error) {
	meta := msg.GetMetadata()
	newMeta := make(map[string]any, len(meta)+1)
	for k, v := range meta {
		newMeta[k] = v
	}
	newMeta[MetaKeySignature] = sig
	newMeta[MetaKeySignatureVersion] = signatureVersion

	switch m := msg.(type) {
	case *schema.HumanMessage:
		return &schema.HumanMessage{Parts: m.Parts, Metadata: newMeta}, nil
	case *schema.AIMessage:
		return &schema.AIMessage{
			Parts:     m.Parts,
			ToolCalls: m.ToolCalls,
			Usage:     m.Usage,
			ModelID:   m.ModelID,
			Metadata:  newMeta,
		}, nil
	case *schema.SystemMessage:
		return &schema.SystemMessage{Parts: m.Parts, Metadata: newMeta}, nil
	case *schema.ToolMessage:
		return &schema.ToolMessage{
			ToolCallID: m.ToolCallID,
			Parts:      m.Parts,
			Metadata:   newMeta,
		}, nil
	default:
		return nil, core.NewError(
			"guard/memory.withSignature",
			core.ErrInvalidInput,
			fmt.Sprintf("unsupported schema.Message implementation %T; cannot attach signature", msg),
			nil,
		)
	}
}

// MessageMiddleware returns this as a memory.Middleware function suitable for
// use with memory.ApplyMiddleware.
func (s *SignedMemoryMiddleware) MessageMiddleware() memory.Middleware {
	return func(next memory.Memory) memory.Memory {
		return s.newSigned(next)
	}
}

// SignDocument returns a copy of doc whose metadata carries a signature over
// its ID, content and "role" metadata. Documents written to the archival tier
// must be signed this way to be returned by a signed Memory's Search.
func (s *SignedMemoryMiddleware) SignDocument(doc schema.Document) schema.Document {
	meta := make(map[string]any, len(doc.Metadata)+2)
	for k, v := range doc.Metadata {
		meta[k] = v
	}
	doc.Metadata = meta
	meta[MetaKeySignature] = computeHMAC(s.key, canonicalDocument(doc))
	meta[MetaKeySignatureVersion] = signatureVersion
	return doc
}

// SignMessage returns a copy of msg carrying a current-version signature, as
// Save would store it. It is useful for seeding stores and for tests.
func (s *SignedMemoryMiddleware) SignMessage(msg schema.Message) (schema.Message, error) {
	return s.newSigned(nil).sign(msg)
}

// Sign computes and returns the raw HMAC-SHA256 hex digest of content. It is
// a low-level helper: message signatures produced by Save use a canonical
// payload that also binds the role and tool calls (see SignMessage).
func (s *SignedMemoryMiddleware) Sign(content string) string {
	return computeHMAC(s.key, content)
}

// Verify checks whether signature is the raw HMAC-SHA256 hex digest of
// content (the counterpart of Sign).
func (s *SignedMemoryMiddleware) Verify(content, signature string) bool {
	return verifyHMAC(s.key, content, signature)
}
