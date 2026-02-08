// Package postgres provides a PostgreSQL-backed implementation of
// memory.MessageStore. Messages are stored in a table with columns for role,
// content (JSON), metadata (JSON), and created_at timestamp. Search uses
// case-insensitive ILIKE queries on the content column.
//
// This implementation uses github.com/jackc/pgx/v5 as the PostgreSQL driver.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// DBTX is the minimal interface satisfied by both pgx.Conn, pgxpool.Pool,
// and pgxmock for testing.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Config holds configuration for the PostgreSQL MessageStore.
type Config struct {
	// DB is the database connection to use. Required.
	DB DBTX
	// Table is the name of the messages table. Defaults to "messages".
	Table string
}

// MessageStore is a PostgreSQL-backed implementation of memory.MessageStore.
type MessageStore struct {
	db    DBTX
	table string
}

// New creates a new PostgreSQL MessageStore with the given config.
// The caller is responsible for creating the table schema before use.
// Use EnsureTable to auto-create it.
func New(cfg Config) (*MessageStore, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("postgres: db is required")
	}
	table := cfg.Table
	if table == "" {
		table = "messages"
	}
	return &MessageStore{
		db:    cfg.DB,
		table: table,
	}, nil
}

// EnsureTable creates the messages table if it does not exist.
func (s *MessageStore) EnsureTable(ctx context.Context) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		role TEXT NOT NULL,
		content JSONB NOT NULL,
		metadata JSONB,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`, s.table)
	_, err := s.db.Exec(ctx, query)
	return err
}

// storedPart is the JSON representation of a schema.ContentPart.
type storedPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// storedContent is the JSON representation of a message's content.
type storedContent struct {
	Parts      []storedPart      `json:"parts"`
	ToolCalls  []schema.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	ModelID    string            `json:"model_id,omitempty"`
}

// Append adds a message to the store.
func (s *MessageStore) Append(ctx context.Context, msg schema.Message) error {
	sc := storedContent{}
	for _, p := range msg.GetContent() {
		sp := storedPart{Type: string(p.PartType())}
		if tp, ok := p.(schema.TextPart); ok {
			sp.Text = tp.Text
		}
		sc.Parts = append(sc.Parts, sp)
	}
	if ai, ok := msg.(*schema.AIMessage); ok {
		sc.ToolCalls = ai.ToolCalls
		sc.ModelID = ai.ModelID
	}
	if tm, ok := msg.(*schema.ToolMessage); ok {
		sc.ToolCallID = tm.ToolCallID
	}

	contentJSON, err := json.Marshal(sc)
	if err != nil {
		return fmt.Errorf("postgres: marshal content: %w", err)
	}

	metadataJSON, err := json.Marshal(msg.GetMetadata())
	if err != nil {
		return fmt.Errorf("postgres: marshal metadata: %w", err)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (role, content, metadata) VALUES ($1, $2, $3)",
		s.table,
	)
	_, err = s.db.Exec(ctx, query, string(msg.GetRole()), contentJSON, metadataJSON)
	if err != nil {
		return fmt.Errorf("postgres: append: %w", err)
	}
	return nil
}

// Search finds messages whose text content contains the query as a
// case-insensitive substring, returning at most k results.
func (s *MessageStore) Search(ctx context.Context, query string, k int) ([]schema.Message, error) {
	sqlQuery := fmt.Sprintf(
		"SELECT role, content, metadata FROM %s WHERE content::text ILIKE $1 ORDER BY created_at ASC LIMIT $2",
		s.table,
	)
	pattern := "%" + query + "%"

	rows, err := s.db.Query(ctx, sqlQuery, pattern, k)
	if err != nil {
		return nil, fmt.Errorf("postgres: search: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// All returns all stored messages in chronological order.
func (s *MessageStore) All(ctx context.Context) ([]schema.Message, error) {
	query := fmt.Sprintf(
		"SELECT role, content, metadata FROM %s ORDER BY created_at ASC",
		s.table,
	)
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: all: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// Clear removes all messages from the store.
func (s *MessageStore) Clear(ctx context.Context) error {
	query := fmt.Sprintf("DELETE FROM %s", s.table)
	_, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("postgres: clear: %w", err)
	}
	return nil
}

func scanMessages(rows pgx.Rows) ([]schema.Message, error) {
	var msgs []schema.Message
	for rows.Next() {
		var role string
		var contentJSON, metadataJSON []byte
		if err := rows.Scan(&role, &contentJSON, &metadataJSON); err != nil {
			return nil, fmt.Errorf("postgres: scan: %w", err)
		}

		var sc storedContent
		if err := json.Unmarshal(contentJSON, &sc); err != nil {
			return nil, fmt.Errorf("postgres: unmarshal content: %w", err)
		}

		var metadata map[string]any
		if len(metadataJSON) > 0 && string(metadataJSON) != "null" {
			if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
				return nil, fmt.Errorf("postgres: unmarshal metadata: %w", err)
			}
		}

		parts := make([]schema.ContentPart, 0, len(sc.Parts))
		for _, sp := range sc.Parts {
			switch schema.ContentType(sp.Type) {
			case schema.ContentText:
				parts = append(parts, schema.TextPart{Text: sp.Text})
			default:
				parts = append(parts, schema.TextPart{Text: sp.Text})
			}
		}

		var msg schema.Message
		switch schema.Role(role) {
		case schema.RoleSystem:
			msg = &schema.SystemMessage{Parts: parts, Metadata: metadata}
		case schema.RoleHuman:
			msg = &schema.HumanMessage{Parts: parts, Metadata: metadata}
		case schema.RoleAI:
			msg = &schema.AIMessage{Parts: parts, ToolCalls: sc.ToolCalls, ModelID: sc.ModelID, Metadata: metadata}
		case schema.RoleTool:
			msg = &schema.ToolMessage{ToolCallID: sc.ToolCallID, Parts: parts, Metadata: metadata}
		default:
			msg = &schema.HumanMessage{Parts: parts, Metadata: metadata}
		}
		msgs = append(msgs, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: rows: %w", err)
	}
	return msgs, nil
}

// Verify interface compliance.
var _ memory.MessageStore = (*MessageStore)(nil)
