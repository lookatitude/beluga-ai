package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// Config holds configuration for the SQLite MessageStore.
type Config struct {
	// DB is the sql.DB connection to use. Required.
	DB *sql.DB
	// Table is the name of the messages table. Defaults to "messages".
	Table string
}

// MessageStore is a SQLite-backed implementation of memory.MessageStore.
type MessageStore struct {
	db    *sql.DB
	table string
}

// New creates a new SQLite MessageStore with the given config.
func New(cfg Config) (*MessageStore, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("sqlite: db is required")
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
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		metadata TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`, s.table)
	_, err := s.db.ExecContext(ctx, query)
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
		return fmt.Errorf("sqlite: marshal content: %w", err)
	}

	metadataJSON, err := json.Marshal(msg.GetMetadata())
	if err != nil {
		return fmt.Errorf("sqlite: marshal metadata: %w", err)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (role, content, metadata, created_at) VALUES (?, ?, ?, ?)",
		s.table,
	)
	_, err = s.db.ExecContext(ctx, query, string(msg.GetRole()), string(contentJSON), string(metadataJSON), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("sqlite: append: %w", err)
	}
	return nil
}

// Search finds messages whose text content contains the query as a
// case-insensitive substring, returning at most k results.
func (s *MessageStore) Search(ctx context.Context, query string, k int) ([]schema.Message, error) {
	sqlQuery := fmt.Sprintf(
		"SELECT role, content, metadata FROM %s WHERE content LIKE ? ORDER BY created_at ASC LIMIT ?",
		s.table,
	)
	pattern := "%" + strings.ToLower(query) + "%"

	rows, err := s.db.QueryContext(ctx, sqlQuery, pattern, k)
	if err != nil {
		return nil, fmt.Errorf("sqlite: search: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// All returns all stored messages in chronological order.
func (s *MessageStore) All(ctx context.Context) ([]schema.Message, error) {
	query := fmt.Sprintf(
		"SELECT role, content, metadata FROM %s ORDER BY created_at ASC, id ASC",
		s.table,
	)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: all: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

// Clear removes all messages from the store.
func (s *MessageStore) Clear(ctx context.Context) error {
	query := fmt.Sprintf("DELETE FROM %s", s.table)
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("sqlite: clear: %w", err)
	}
	return nil
}

func scanMessages(rows *sql.Rows) ([]schema.Message, error) {
	var msgs []schema.Message
	for rows.Next() {
		var role, contentStr, metadataStr string
		if err := rows.Scan(&role, &contentStr, &metadataStr); err != nil {
			return nil, fmt.Errorf("sqlite: scan: %w", err)
		}

		var sc storedContent
		if err := json.Unmarshal([]byte(contentStr), &sc); err != nil {
			return nil, fmt.Errorf("sqlite: unmarshal content: %w", err)
		}

		var metadata map[string]any
		if metadataStr != "" && metadataStr != "null" {
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
				return nil, fmt.Errorf("sqlite: unmarshal metadata: %w", err)
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
		return nil, fmt.Errorf("sqlite: rows: %w", err)
	}
	return msgs, nil
}

// Verify interface compliance.
var _ memory.MessageStore = (*MessageStore)(nil)
