package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

func TestInMemorySessionServiceCreate(t *testing.T) {
	tests := []struct {
		name    string
		agentID string
		opts    []SessionOption
		wantErr bool
		errCode core.ErrorCode
	}{
		{
			name:    "create session succeeds",
			agentID: "agent-1",
			wantErr: false,
		},
		{
			name:    "create with tenant ID",
			agentID: "agent-1",
			opts:    []SessionOption{WithSessionTenantID("tenant-1")},
			wantErr: false,
		},
		{
			name:    "context cancelled",
			agentID: "agent-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewInMemorySessionService(tt.opts...)

			ctx := context.Background()
			if tt.name == "context cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			session, err := svc.Create(ctx, tt.agentID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, session)
				assert.NotEmpty(t, session.ID)
				assert.Equal(t, tt.agentID, session.AgentID)
				if len(tt.opts) > 0 {
					assert.Equal(t, "tenant-1", session.TenantID)
				}
				assert.NotNil(t, session.State)
				assert.False(t, session.CreatedAt.IsZero())
				assert.False(t, session.UpdatedAt.IsZero())
			}
		})
	}
}

func TestInMemorySessionServiceMaxSessionsLimit(t *testing.T) {
	svc := NewInMemorySessionService(WithMaxSessions(2))

	ctx := context.Background()

	// Create first session - should succeed
	sess1, err := svc.Create(ctx, "agent-1")
	require.NoError(t, err)
	require.NotNil(t, sess1)

	// Create second session - should succeed
	sess2, err := svc.Create(ctx, "agent-1")
	require.NoError(t, err)
	require.NotNil(t, sess2)

	// Create third session - should fail with quota exceeded
	sess3, err := svc.Create(ctx, "agent-1")
	assert.Error(t, err)
	assert.Nil(t, sess3)

	// Verify error code is ErrInvalidInput (limit reached)
	typedErr, ok := err.(*core.Error)
	require.True(t, ok, "expected *core.Error")
	assert.Equal(t, core.ErrInvalidInput, typedErr.Code)

	// Verify error message does not contain session IDs
	assert.NotContains(t, typedErr.Error(), "sess1")
	assert.NotContains(t, typedErr.Error(), "sess2")
}

func TestInMemorySessionServiceGet(t *testing.T) {
	svc := NewInMemorySessionService()
	ctx := context.Background()

	// Create a session
	created, err := svc.Create(ctx, "agent-1")
	require.NoError(t, err)

	// Get existing session
	retrieved, err := svc.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.AgentID, retrieved.AgentID)

	// Get non-existent session
	notFound, err := svc.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, notFound)

	// Verify error message is sanitized (no session ID)
	typedErr, ok := err.(*core.Error)
	require.True(t, ok)
	assert.Equal(t, core.ErrNotFound, typedErr.Code)
	assert.NotContains(t, typedErr.Error(), "nonexistent")
	assert.Contains(t, typedErr.Error(), "session not found")
}

func TestInMemorySessionServiceGetExpired(t *testing.T) {
	svc := NewInMemorySessionService(WithSessionTTL(1 * time.Millisecond))
	ctx := context.Background()

	// Create a session with short TTL
	created, err := svc.Create(ctx, "agent-1")
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Try to get expired session
	retrieved, err := svc.Get(ctx, created.ID)
	assert.Error(t, err)
	assert.Nil(t, retrieved)

	// Verify error message is sanitized
	typedErr, ok := err.(*core.Error)
	require.True(t, ok)
	assert.Equal(t, core.ErrNotFound, typedErr.Code)
	assert.NotContains(t, typedErr.Error(), created.ID)
	assert.Contains(t, typedErr.Error(), "session expired")
}

func TestInMemorySessionServiceUpdate(t *testing.T) {
	svc := NewInMemorySessionService()
	ctx := context.Background()

	// Create a session
	created, err := svc.Create(ctx, "agent-1")
	require.NoError(t, err)

	// Update session state
	created.State["key"] = "value"
	err = svc.Update(ctx, created)
	require.NoError(t, err)

	// Retrieve and verify update
	retrieved, err := svc.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "value", retrieved.State["key"])

	// Update non-existent session
	fakeSession := &Session{ID: "nonexistent"}
	err = svc.Update(ctx, fakeSession)
	assert.Error(t, err)

	// Verify error message is sanitized
	typedErr, ok := err.(*core.Error)
	require.True(t, ok)
	assert.Equal(t, core.ErrNotFound, typedErr.Code)
	assert.NotContains(t, typedErr.Error(), "nonexistent")

	// Update nil session
	err = svc.Update(ctx, nil)
	assert.Error(t, err)
	typedErr, ok = err.(*core.Error)
	require.True(t, ok)
	assert.Equal(t, core.ErrInvalidInput, typedErr.Code)
}

func TestInMemorySessionServiceDelete(t *testing.T) {
	svc := NewInMemorySessionService()
	ctx := context.Background()

	// Create a session
	created, err := svc.Create(ctx, "agent-1")
	require.NoError(t, err)

	// Delete existing session
	err = svc.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify session is gone
	_, err = svc.Get(ctx, created.ID)
	assert.Error(t, err)

	// Delete non-existent session
	err = svc.Delete(ctx, "nonexistent")
	assert.Error(t, err)

	// Verify error message is sanitized
	typedErr, ok := err.(*core.Error)
	require.True(t, ok)
	assert.Equal(t, core.ErrNotFound, typedErr.Code)
	assert.NotContains(t, typedErr.Error(), "nonexistent")
	assert.Contains(t, typedErr.Error(), "session not found")
}

func TestCloneSessionDeepCopiesTurns(t *testing.T) {
	// Create a session with turns containing metadata
	session := &Session{
		ID: "sess-1",
		Turns: []schema.Turn{
			{
				Input:     schema.NewHumanMessage("hello"),
				Output:    schema.NewAIMessage("hi"),
				Timestamp: time.Now(),
				Metadata: map[string]any{
					"source": "user",
					"score":  0.95,
				},
			},
			{
				Input:     schema.NewHumanMessage("how are you"),
				Output:    schema.NewAIMessage("i'm fine"),
				Timestamp: time.Now(),
				Metadata: map[string]any{
					"confidence": 0.88,
				},
			},
		},
		State: map[string]any{
			"conversation_id": "conv-123",
		},
	}

	// Clone the session
	cloned := cloneSession(session)

	// Verify structure is correct
	assert.Equal(t, len(session.Turns), len(cloned.Turns))

	// Mutate cloned metadata
	cloned.Turns[0].Metadata["source"] = "modified"
	cloned.Turns[0].Metadata["new_key"] = "new_value"

	// Verify original is unchanged
	assert.Equal(t, "user", session.Turns[0].Metadata["source"])
	assert.NotContains(t, session.Turns[0].Metadata, "new_key")

	// Mutate cloned state
	cloned.State["conversation_id"] = "modified"
	cloned.State["new_field"] = "new_value"

	// Verify original state is unchanged
	assert.Equal(t, "conv-123", session.State["conversation_id"])
	assert.NotContains(t, session.State, "new_field")
}

func TestCloneSessionNilMetadata(t *testing.T) {
	// Create a session with turn that has nil metadata
	session := &Session{
		ID: "sess-1",
		Turns: []schema.Turn{
			{
				Input:     schema.NewHumanMessage("hello"),
				Output:    schema.NewAIMessage("hi"),
				Timestamp: time.Now(),
				Metadata:  nil,
			},
		},
	}

	cloned := cloneSession(session)

	// Verify nil metadata stays nil
	assert.Nil(t, cloned.Turns[0].Metadata)
	assert.Nil(t, session.Turns[0].Metadata)
}

func TestCloneSessionEmptyTurns(t *testing.T) {
	session := &Session{
		ID:    "sess-1",
		Turns: []schema.Turn{},
		State: map[string]any{"key": "value"},
	}

	cloned := cloneSession(session)

	// Empty turns should result in proper empty slice copy
	assert.Equal(t, 0, len(cloned.Turns))
	assert.NotNil(t, cloned.Turns)
	assert.Equal(t, "value", cloned.State["key"])
}

func TestInMemorySessionServiceContextCancellation(t *testing.T) {
	svc := NewInMemorySessionService()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// All operations should fail with context cancelled
	_, err := svc.Create(ctx, "agent-1")
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	_, err = svc.Get(ctx, "any-id")
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	err = svc.Update(ctx, &Session{ID: "any-id"})
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	err = svc.Delete(ctx, "any-id")
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestInMemorySessionServiceConcurrentAccess(t *testing.T) {
	svc := NewInMemorySessionService()
	ctx := context.Background()

	// Create multiple sessions concurrently
	sessIDs := make(chan string, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			session, err := svc.Create(ctx, "agent-1")
			require.NoError(t, err)
			sessIDs <- session.ID
		}(i)
	}

	// Collect all IDs
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := <-sessIDs
		ids[id] = true
	}

	// Verify all IDs are unique
	assert.Equal(t, 10, len(ids))

	// Verify all can be retrieved
	for id := range ids {
		_, err := svc.Get(ctx, id)
		require.NoError(t, err)
	}
}

func TestInMemorySessionServiceZeroConfigWorks(t *testing.T) {
	// Zero-config should work with defaults
	svc := NewInMemorySessionService()
	ctx := context.Background()

	session, err := svc.Create(ctx, "agent-1")
	require.NoError(t, err)
	assert.NotNil(t, session)

	retrieved, err := svc.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
}
