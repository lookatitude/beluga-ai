// Package mongodb provides a MongoDB-backed implementation of memory.MessageStore.
// Messages are stored as BSON documents in a MongoDB collection with a sequence
// field for chronological ordering.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/memory/stores/mongodb"
//
//	store, err := mongodb.New(mongodb.Config{
//	    Collection: client.Database("beluga").Collection("messages"),
//	})
package mongodb

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Collection defines the subset of mongo.Collection methods used by this store.
// This interface enables testing with mock implementations.
type Collection interface {
	InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error)
	Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error)
	DeleteMany(ctx context.Context, filter any, opts ...options.Lister[options.DeleteManyOptions]) (*mongo.DeleteResult, error)
}

// Config holds configuration for the MongoDB MessageStore.
type Config struct {
	// Collection is the MongoDB collection to use. Required.
	Collection Collection
}

// MessageStore is a MongoDB-backed implementation of memory.MessageStore.
// Messages are stored as BSON documents with a monotonically increasing
// sequence number for chronological ordering.
type MessageStore struct {
	coll Collection
	seq  atomic.Int64
}

// New creates a new MongoDB MessageStore with the given config.
func New(cfg Config) (*MessageStore, error) {
	if cfg.Collection == nil {
		return nil, fmt.Errorf("mongodb: collection is required")
	}
	return &MessageStore{
		coll: cfg.Collection,
	}, nil
}

// messageDoc is the BSON representation of a schema.Message.
type messageDoc struct {
	Seq        int64         `bson:"seq"`
	Role       string        `bson:"role"`
	Parts      []partDoc     `bson:"parts"`
	Metadata   bson.M        `bson:"metadata,omitempty"`
	ToolCalls  []toolCallDoc `bson:"tool_calls,omitempty"`
	ToolCallID string        `bson:"tool_call_id,omitempty"`
	ModelID    string        `bson:"model_id,omitempty"`
	Timestamp  time.Time     `bson:"timestamp"`
}

// partDoc is the BSON representation of a schema.ContentPart.
type partDoc struct {
	Type string `bson:"type"`
	Text string `bson:"text,omitempty"`
}

// toolCallDoc is the BSON representation of a schema.ToolCall.
type toolCallDoc struct {
	ID        string `bson:"id"`
	Name      string `bson:"name"`
	Arguments string `bson:"arguments"`
}

// Append adds a message to the store.
func (s *MessageStore) Append(ctx context.Context, msg schema.Message) error {
	doc := s.marshalMessage(msg)
	_, err := s.coll.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("mongodb: insert: %w", err)
	}
	return nil
}

// Search finds messages whose text content contains the query as a
// case-insensitive substring, returning at most k results.
func (s *MessageStore) Search(ctx context.Context, query string, k int) ([]schema.Message, error) {
	all, err := s.allDocs(ctx)
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(query)
	var results []schema.Message
	for _, doc := range all {
		msg := unmarshalDoc(doc)
		if q == "" || containsText(msg, q) {
			results = append(results, msg)
			if len(results) >= k {
				break
			}
		}
	}
	return results, nil
}

// All returns all stored messages in chronological order.
func (s *MessageStore) All(ctx context.Context) ([]schema.Message, error) {
	docs, err := s.allDocs(ctx)
	if err != nil {
		return nil, err
	}

	msgs := make([]schema.Message, 0, len(docs))
	for _, doc := range docs {
		msgs = append(msgs, unmarshalDoc(doc))
	}
	return msgs, nil
}

// Clear removes all messages from the store.
func (s *MessageStore) Clear(ctx context.Context) error {
	_, err := s.coll.DeleteMany(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("mongodb: clear: %w", err)
	}
	return nil
}

// allDocs returns all documents sorted by sequence number.
func (s *MessageStore) allDocs(ctx context.Context) ([]messageDoc, error) {
	findOpts := options.Find().SetSort(bson.D{{Key: "seq", Value: 1}})
	cursor, err := s.coll.Find(ctx, bson.D{}, findOpts)
	if err != nil {
		return nil, fmt.Errorf("mongodb: find: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []messageDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongodb: decode: %w", err)
	}
	return docs, nil
}

func (s *MessageStore) marshalMessage(msg schema.Message) messageDoc {
	doc := messageDoc{
		Seq:       s.seq.Add(1),
		Role:      string(msg.GetRole()),
		Metadata:  toBsonM(msg.GetMetadata()),
		Timestamp: time.Now(),
	}
	for _, p := range msg.GetContent() {
		pd := partDoc{Type: string(p.PartType())}
		if tp, ok := p.(schema.TextPart); ok {
			pd.Text = tp.Text
		}
		doc.Parts = append(doc.Parts, pd)
	}
	if ai, ok := msg.(*schema.AIMessage); ok {
		doc.ModelID = ai.ModelID
		for _, tc := range ai.ToolCalls {
			doc.ToolCalls = append(doc.ToolCalls, toolCallDoc{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			})
		}
	}
	if tm, ok := msg.(*schema.ToolMessage); ok {
		doc.ToolCallID = tm.ToolCallID
	}
	return doc
}

func unmarshalDoc(doc messageDoc) schema.Message {
	parts := make([]schema.ContentPart, 0, len(doc.Parts))
	for _, pd := range doc.Parts {
		parts = append(parts, schema.TextPart{Text: pd.Text})
	}

	meta := fromBsonM(doc.Metadata)

	switch schema.Role(doc.Role) {
	case schema.RoleSystem:
		return &schema.SystemMessage{Parts: parts, Metadata: meta}
	case schema.RoleHuman:
		return &schema.HumanMessage{Parts: parts, Metadata: meta}
	case schema.RoleAI:
		var tcs []schema.ToolCall
		for _, td := range doc.ToolCalls {
			tcs = append(tcs, schema.ToolCall{ID: td.ID, Name: td.Name, Arguments: td.Arguments})
		}
		return &schema.AIMessage{Parts: parts, ToolCalls: tcs, ModelID: doc.ModelID, Metadata: meta}
	case schema.RoleTool:
		return &schema.ToolMessage{ToolCallID: doc.ToolCallID, Parts: parts, Metadata: meta}
	default:
		return &schema.HumanMessage{Parts: parts, Metadata: meta}
	}
}

// containsText checks if any text part in the message contains the query.
func containsText(msg schema.Message, lowerQuery string) bool {
	for _, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			if strings.Contains(strings.ToLower(tp.Text), lowerQuery) {
				return true
			}
		}
	}
	return false
}

func toBsonM(m map[string]any) bson.M {
	if m == nil {
		return nil
	}
	result := make(bson.M, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func fromBsonM(m bson.M) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// Verify interface compliance.
var _ memory.MessageStore = (*MessageStore)(nil)
