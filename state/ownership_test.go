package state

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOwnershipManager_ClaimAndRelease(t *testing.T) {
	om := NewOwnershipManager()

	// Claim a key.
	require.NoError(t, om.Claim("k1", "agent-a"))
	assert.Equal(t, "agent-a", om.Owner("k1"))

	// Same owner re-claiming is idempotent.
	require.NoError(t, om.Claim("k1", "agent-a"))

	// Different owner cannot claim.
	err := om.Claim("k1", "agent-b")
	assert.True(t, errors.Is(err, ErrAlreadyClaimed))

	// Release by owner.
	require.NoError(t, om.Release("k1", "agent-a"))
	assert.Equal(t, "", om.Owner("k1"))

	// Now agent-b can claim.
	require.NoError(t, om.Claim("k1", "agent-b"))
	assert.Equal(t, "agent-b", om.Owner("k1"))
}

func TestOwnershipManager_ReleaseByNonOwner(t *testing.T) {
	om := NewOwnershipManager()

	require.NoError(t, om.Claim("k", "owner"))
	err := om.Release("k", "stranger")
	assert.True(t, errors.Is(err, ErrOwnershipDenied))

	// Still owned by original.
	assert.Equal(t, "owner", om.Owner("k"))
}

func TestOwnershipManager_ReleaseUnclaimed(t *testing.T) {
	om := NewOwnershipManager()
	require.NoError(t, om.Release("missing", "anyone"))
}

func TestOwnershipManager_CheckWrite(t *testing.T) {
	om := NewOwnershipManager()

	// Unclaimed key — anyone can write.
	require.NoError(t, om.CheckWrite("k", "anyone"))

	// Claim the key.
	require.NoError(t, om.Claim("k", "owner"))

	// Owner can write.
	require.NoError(t, om.CheckWrite("k", "owner"))

	// Non-owner cannot write.
	err := om.CheckWrite("k", "stranger")
	assert.True(t, errors.Is(err, ErrOwnershipDenied))
}

func TestWithOwnerID_Context(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, "", OwnerIDFromContext(ctx))

	ctx = WithOwnerID(ctx, "agent-1")
	assert.Equal(t, "agent-1", OwnerIDFromContext(ctx))
}

func TestWithOwnership_Middleware(t *testing.T) {
	om := NewOwnershipManager()
	require.NoError(t, om.Claim("protected", "agent-a"))

	inner := newMockStore()
	store := ApplyMiddleware(inner, WithOwnership(om))

	ctx := context.Background()

	// No owner in context — unclaimed keys work.
	require.NoError(t, store.Set(ctx, "open", "val"))

	// No owner in context — claimed keys work (no enforcement without owner ID).
	require.NoError(t, store.Set(ctx, "protected", "val"))

	// Owner in context — owner can write.
	ctxA := WithOwnerID(ctx, "agent-a")
	require.NoError(t, store.Set(ctxA, "protected", "val-a"))

	// Non-owner in context — cannot write.
	ctxB := WithOwnerID(ctx, "agent-b")
	err := store.Set(ctxB, "protected", "val-b")
	assert.True(t, errors.Is(err, ErrOwnershipDenied))

	// Get always works regardless of ownership.
	_, err = store.Get(ctxB, "protected")
	require.NoError(t, err)

	// Delete by non-owner is denied.
	err = store.Delete(ctxB, "protected")
	assert.True(t, errors.Is(err, ErrOwnershipDenied))

	// Delete by owner works.
	require.NoError(t, store.Delete(ctxA, "protected"))
}
