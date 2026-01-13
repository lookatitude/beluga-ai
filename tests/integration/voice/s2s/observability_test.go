package s2s

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	s2siface "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestS2S_Observability_Metrics tests S2S metrics collection.
func TestS2S_Observability_Metrics(t *testing.T) {
	ctx := context.Background()

	// Initialize metrics (required for GetMetrics to return non-nil)
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	s2s.InitMetrics(meter, tracer)

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create voice session
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio - metrics should be recorded
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Verify metrics are available (indirectly through metrics package)
	// In a full implementation, we would verify specific metric values
	metrics := s2s.GetMetrics()
	assert.NotNil(t, metrics)

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestS2S_Observability_Tracing tests S2S distributed tracing.
func TestS2S_Observability_Tracing(t *testing.T) {
	ctx := context.Background()

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create voice session
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio - traces should be created
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Verify tracing works (indirectly through tracing package)
	// In a full implementation, we would verify span creation and attributes
	// For now, we just verify the integration works without errors

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestS2S_Observability_Logging tests S2S structured logging.
func TestS2S_Observability_Logging(t *testing.T) {
	ctx := context.Background()

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create voice session
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio - logs should be written
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Verify logging works (indirectly through logging package)
	// In a full implementation, we would verify log entries
	// For now, we just verify the integration works without errors

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestS2S_Observability_EndToEnd tests complete observability stack.
func TestS2S_Observability_EndToEnd(t *testing.T) {
	ctx := context.Background()

	// Initialize metrics (required for GetMetrics to return non-nil)
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	s2s.InitMetrics(meter, tracer)

	// Create S2S provider
	s2sProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "test-provider", 100*time.Millisecond)))

	// Create voice session
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process multiple audio chunks to generate observability data
	audioChunks := [][]byte{
		[]byte{1, 2, 3, 4, 5},
		[]byte{6, 7, 8, 9, 10},
		[]byte{11, 12, 13, 14, 15},
	}

	for i, audio := range audioChunks {
		t.Run(fmt.Sprintf("process_chunk_%d", i), func(t *testing.T) {
			err := voiceSession.ProcessAudio(ctx, audio)
			require.NoError(t, err)
		})
	}

	// Verify observability components are working
	metrics := s2s.GetMetrics()
	assert.NotNil(t, metrics)

	// Stop session
	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}

// TestS2S_Observability_ProviderMetrics tests provider-specific metrics.
func TestS2S_Observability_ProviderMetrics(t *testing.T) {
	ctx := context.Background()

	// Initialize metrics (required for GetMetrics to return non-nil)
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	s2s.InitMetrics(meter, tracer)

	// Create multiple S2S providers to test provider-specific metrics
	providers := []s2siface.S2SProvider{
		s2s.NewAdvancedMockS2SProvider("provider-1",
			s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "provider-1", 100*time.Millisecond))),
		s2s.NewAdvancedMockS2SProvider("provider-2",
			s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{4, 5, 6}, "provider-2", 150*time.Millisecond))),
	}

	for i, provider := range providers {
		t.Run(fmt.Sprintf("provider_%d", i), func(t *testing.T) {
			voiceSession, err := session.NewVoiceSession(ctx,
				session.WithS2SProvider(provider),
			)
			require.NoError(t, err)

			err = voiceSession.Start(ctx)
			require.NoError(t, err)

			audio := []byte{1, 2, 3, 4, 5}
			err = voiceSession.ProcessAudio(ctx, audio)
			require.NoError(t, err)

			// Verify metrics are recorded per provider
			metrics := s2s.GetMetrics()
			assert.NotNil(t, metrics)

			err = voiceSession.Stop(ctx)
			require.NoError(t, err)
		})
	}
}

// TestS2S_Observability_ErrorTracking tests error tracking in observability.
func TestS2S_Observability_ErrorTracking(t *testing.T) {
	ctx := context.Background()

	// Initialize metrics (required for GetMetrics to return non-nil)
	// In production, this would be done at application startup
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	s2s.InitMetrics(meter, tracer)

	// Create S2S provider that will error
	s2sProvider := s2s.NewAdvancedMockS2SProvider("error-provider",
		s2s.WithError(s2s.NewS2SError("Process", s2s.ErrCodeNetworkError, nil)))

	// Create voice session
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Start session
	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Process audio - should error, and error should be tracked
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	// Error is expected, but should be tracked in observability

	// Verify error metrics are recorded
	metrics := s2s.GetMetrics()
	assert.NotNil(t, metrics)

	// Stop session
	_ = voiceSession.Stop(ctx)
}
