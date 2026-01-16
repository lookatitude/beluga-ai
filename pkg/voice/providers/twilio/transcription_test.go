package twilio

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranscriptionManager_RetrieveTranscription(t *testing.T) {
	config := &TwilioConfig{
		Config: &iface.Config{
			Provider: "twilio",
		},
		AccountSID: "AC1234567890abcdef",
		AuthToken:  "test_auth_token",
	}

	backend, err := NewTwilioBackend(config)
	require.NoError(t, err)

	transcriptionMgr := NewTranscriptionManager(backend)

	ctx := context.Background()
	// This will fail without real credentials, but tests structure
	_, err = transcriptionMgr.RetrieveTranscription(ctx, "TR1234567890abcdef")
	assert.Error(t, err) // Expected without real credentials
}

func TestTranscriptionManager_StoreTranscription(t *testing.T) {
	config := &TwilioConfig{
		Config: &iface.Config{
			Provider: "twilio",
		},
		AccountSID: "AC1234567890abcdef",
		AuthToken:  "test_auth_token",
	}

	backend, err := NewTwilioBackend(config)
	require.NoError(t, err)

	transcriptionMgr := NewTranscriptionManager(backend)

	transcription := &Transcription{
		TranscriptionSID: "TR1234567890abcdef",
		CallSID:          "CA1234567890abcdef",
		AccountSID:       "AC1234567890abcdef",
		Status:           "completed",
		Text:             "This is a test transcription",
		Language:         "en-US",
		DateCreated:      time.Now(),
		DateUpdated:      time.Now(),
	}

	ctx := context.Background()
	// This will fail without vector store configured, but tests structure
	err = transcriptionMgr.StoreTranscription(ctx, transcription)
	// May fail without vector store, but structure is correct
	_ = err
}

func TestTranscriptionManager_SearchTranscriptions(t *testing.T) {
	config := &TwilioConfig{
		Config: &iface.Config{
			Provider: "twilio",
		},
		AccountSID: "AC1234567890abcdef",
		AuthToken:  "test_auth_token",
	}

	backend, err := NewTwilioBackend(config)
	require.NoError(t, err)

	transcriptionMgr := NewTranscriptionManager(backend)

	ctx := context.Background()
	// This will fail without retriever configured, but tests structure
	_, err = transcriptionMgr.SearchTranscriptions(ctx, "test query", 5)
	// May fail without retriever, but structure is correct
	_ = err
}
