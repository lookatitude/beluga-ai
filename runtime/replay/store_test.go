package replay

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStore_Save(t *testing.T) {
	tests := []struct {
		name    string
		cp      *Checkpoint
		wantErr bool
		errCode core.ErrorCode
	}{
		{
			name: "save valid checkpoint",
			cp: &Checkpoint{
				ID:        "cp-1",
				SessionID: "sess-1",
				TurnIndex: 0,
				Timestamp: time.Now(),
			},
		},
		{
			name:    "nil checkpoint",
			cp:      nil,
			wantErr: true,
			errCode: core.ErrInvalidInput,
		},
		{
			name: "empty ID",
			cp: &Checkpoint{
				ID:        "",
				SessionID: "sess-1",
			},
			wantErr: true,
			errCode: core.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewInMemoryStore()
			err := store.Save(context.Background(), tt.cp)
			if tt.wantErr {
				require.Error(t, err)
				var coreErr *core.Error
				require.ErrorAs(t, err, &coreErr)
				assert.Equal(t, tt.errCode, coreErr.Code)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestInMemoryStore_Get(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	cp := &Checkpoint{
		ID:        "cp-1",
		SessionID: "sess-1",
		TurnIndex: 2,
		Turns: []schema.Turn{
			{Input: schema.NewHumanMessage("hello"), Timestamp: time.Now()},
		},
		Timestamp: time.Now(),
	}
	require.NoError(t, store.Save(ctx, cp))

	t.Run("existing checkpoint", func(t *testing.T) {
		got, err := store.Get(ctx, "cp-1")
		require.NoError(t, err)
		assert.Equal(t, cp.ID, got.ID)
		assert.Equal(t, cp.SessionID, got.SessionID)
		assert.Equal(t, cp.TurnIndex, got.TurnIndex)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.Get(ctx, "nonexistent")
		require.Error(t, err)
		var coreErr *core.Error
		require.ErrorAs(t, err, &coreErr)
		assert.Equal(t, core.ErrNotFound, coreErr.Code)
	})
}

func TestInMemoryStore_List(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Save checkpoints for two sessions, out of order.
	cps := []*Checkpoint{
		{ID: "cp-3", SessionID: "sess-1", TurnIndex: 2, Timestamp: time.Now()},
		{ID: "cp-1", SessionID: "sess-1", TurnIndex: 0, Timestamp: time.Now()},
		{ID: "cp-2", SessionID: "sess-1", TurnIndex: 1, Timestamp: time.Now()},
		{ID: "cp-other", SessionID: "sess-2", TurnIndex: 0, Timestamp: time.Now()},
	}
	for _, cp := range cps {
		require.NoError(t, store.Save(ctx, cp))
	}

	t.Run("returns ordered by turn index", func(t *testing.T) {
		ids, err := store.List(ctx, "sess-1")
		require.NoError(t, err)
		assert.Equal(t, []string{"cp-1", "cp-2", "cp-3"}, ids)
	})

	t.Run("different session", func(t *testing.T) {
		ids, err := store.List(ctx, "sess-2")
		require.NoError(t, err)
		assert.Equal(t, []string{"cp-other"}, ids)
	})

	t.Run("empty session", func(t *testing.T) {
		ids, err := store.List(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, ids)
	})
}

func TestInMemoryStore_Delete(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	cp := &Checkpoint{ID: "cp-1", SessionID: "sess-1", TurnIndex: 0, Timestamp: time.Now()}
	require.NoError(t, store.Save(ctx, cp))

	t.Run("delete existing", func(t *testing.T) {
		err := store.Delete(ctx, "cp-1")
		require.NoError(t, err)

		_, err = store.Get(ctx, "cp-1")
		require.Error(t, err)
	})

	t.Run("delete not found", func(t *testing.T) {
		err := store.Delete(ctx, "nonexistent")
		require.Error(t, err)
		var coreErr *core.Error
		require.ErrorAs(t, err, &coreErr)
		assert.Equal(t, core.ErrNotFound, coreErr.Code)
	})
}

func TestInMemoryStore_SaveOverwrite(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	cp1 := &Checkpoint{ID: "cp-1", SessionID: "sess-1", TurnIndex: 0, Timestamp: time.Now()}
	require.NoError(t, store.Save(ctx, cp1))

	cp2 := &Checkpoint{ID: "cp-1", SessionID: "sess-1", TurnIndex: 5, Timestamp: time.Now()}
	require.NoError(t, store.Save(ctx, cp2))

	got, err := store.Get(ctx, "cp-1")
	require.NoError(t, err)
	assert.Equal(t, 5, got.TurnIndex)
}
