package session

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/session/internal"
	"github.com/stretchr/testify/assert"
)

func TestPreemptiveGeneration_HandleInterim(t *testing.T) {
	interimReceived := false
	interimHandler := func(transcript string) {
		interimReceived = true
	}

	pg := internal.NewPreemptiveGeneration(true, internal.ResponseStrategyAlwaysUse)
	pg.SetInterimHandler(interimHandler)

	ctx := context.Background()
	pg.HandleInterim(ctx, "test transcript")

	assert.True(t, interimReceived)
}

func TestPreemptiveGeneration_HandleFinal(t *testing.T) {
	finalReceived := false
	finalText := ""
	finalHandler := func(transcript string) {
		finalReceived = true
		finalText = transcript
	}

	pg := internal.NewPreemptiveGeneration(true, internal.ResponseStrategyAlwaysUse)
	pg.SetFinalHandler(finalHandler)

	ctx := context.Background()
	pg.HandleFinal(ctx, "final transcript")

	assert.True(t, finalReceived)
	assert.Equal(t, "final transcript", finalText)
}

func TestPreemptiveGeneration_ResponseStrategy(t *testing.T) {
	pg := internal.NewPreemptiveGeneration(true, internal.ResponseStrategyUseIfSimilar)
	assert.Equal(t, internal.ResponseStrategyUseIfSimilar, pg.GetResponseStrategy())

	pg = internal.NewPreemptiveGeneration(true, internal.ResponseStrategyAlwaysUse)
	assert.Equal(t, internal.ResponseStrategyAlwaysUse, pg.GetResponseStrategy())
}
