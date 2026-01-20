package gemma

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockGemmaProvider provides a comprehensive mock implementation for testing Gemma multimodal provider.
type AdvancedMockGemmaProvider struct {
	mock.Mock
	mu                sync.RWMutex
	callCount         int
	shouldError       bool
	errorToReturn     error
	modelName         string
	capabilities      *types.ModalityCapabilities
	simulateDelay     time.Duration
	simulateRateLimit bool
	rateLimitCount    int
	lastInput         *types.MultimodalInput
	lastOutput        *types.MultimodalOutput
}

// NewAdvancedMockGemmaProvider creates a new advanced mock with configurable behavior.
func NewAdvancedMockGemmaProvider(modelName string) *AdvancedMockGemmaProvider {
	mock := &AdvancedMockGemmaProvider{
		modelName: modelName,
		capabilities: &types.ModalityCapabilities{
			Text:                  true,
			Image:                 true,
			Audio:                 true,
			Video:                 true,
			MaxImageSize:          20 * 1024 * 1024,  // 20MB
			MaxAudioSize:          25 * 1024 * 1024,  // 25MB
			MaxVideoSize:          100 * 1024 * 1024, // 100MB
			SupportedImageFormats: []string{"png", "jpeg", "jpg", "gif", "webp"},
			SupportedAudioFormats: []string{"mp3", "wav", "m4a", "ogg"},
			SupportedVideoFormats: []string{"mp4", "webm", "mov"},
		},
	}
	return mock
}

// Process processes a multimodal input and returns a multimodal output.
func (m *AdvancedMockGemmaProvider) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	simulateRateLimit := m.simulateRateLimit
	rateLimitCount := m.rateLimitCount
	m.lastInput = input
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	m.mu.Lock()
	if simulateRateLimit && rateLimitCount > 5 {
		m.mu.Unlock()
		return nil, multimodal.NewMultimodalError("Process", multimodal.ErrCodeRateLimit, errors.New("rate limit exceeded"))
	}
	m.rateLimitCount++
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, multimodal.NewMultimodalError("Process", multimodal.ErrCodeProviderError, errors.New("mock error"))
	}

	output := &types.MultimodalOutput{
		ID:            "mock-gemma-output-" + time.Now().Format(time.RFC3339Nano),
		InputID:       input.ID,
		ContentBlocks: input.ContentBlocks,
		Metadata:      map[string]any{"provider": "gemma", "model": m.modelName},
		Confidence:    0.95,
		Provider:      "gemma",
		Model:         m.modelName,
		CreatedAt:     time.Now(),
	}

	m.mu.Lock()
	m.lastOutput = output
	m.mu.Unlock()

	return output, nil
}

// ProcessStream processes a multimodal input and streams results incrementally.
func (m *AdvancedMockGemmaProvider) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.lastInput = input
	m.mu.Unlock()

	if shouldError {
		if errorToReturn != nil {
			return nil, errorToReturn
		}
		return nil, multimodal.NewMultimodalError("ProcessStream", multimodal.ErrCodeProviderError, errors.New("mock error"))
	}

	ch := make(chan *types.MultimodalOutput, 1)
	go func() {
		defer close(ch)

		output := &types.MultimodalOutput{
			ID:            "mock-gemma-stream-" + time.Now().Format(time.RFC3339Nano),
			InputID:       input.ID,
			ContentBlocks: input.ContentBlocks,
			Metadata:      map[string]any{"provider": "gemma", "model": m.modelName, "streaming": true},
			Confidence:    0.95,
			Provider:      "gemma",
			Model:         m.modelName,
			CreatedAt:     time.Now(),
		}

		select {
		case ch <- output:
		case <-ctx.Done():
			return
		}
	}()

	return ch, nil
}

// GetCapabilities returns the capabilities of this model.
func (m *AdvancedMockGemmaProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.capabilities, nil
}

// SupportsModality checks if this model supports a specific modality.
func (m *AdvancedMockGemmaProvider) SupportsModality(ctx context.Context, modality string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch modality {
	case "text":
		return m.capabilities.Text, nil
	case "image":
		return m.capabilities.Image, nil
	case "audio":
		return m.capabilities.Audio, nil
	case "video":
		return m.capabilities.Video, nil
	default:
		return false, nil
	}
}

// CheckHealth performs a health check and returns an error if the model is unhealthy.
func (m *AdvancedMockGemmaProvider) CheckHealth(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError {
		return m.errorToReturn
	}

	return nil
}

// WithMockError configures the mock to return an error.
func (m *AdvancedMockGemmaProvider) WithMockError(shouldError bool, err error) *AdvancedMockGemmaProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorToReturn = err
	return m
}

// WithMockDelay sets the delay for operations.
func (m *AdvancedMockGemmaProvider) WithMockDelay(delay time.Duration) *AdvancedMockGemmaProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateDelay = delay
	return m
}

// WithRateLimit simulates rate limiting behavior.
func (m *AdvancedMockGemmaProvider) WithRateLimit(enabled bool) *AdvancedMockGemmaProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateRateLimit = enabled
	return m
}

// GetCallCount returns the number of times Process or ProcessStream was called.
func (m *AdvancedMockGemmaProvider) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetLastInput returns the last input that was processed.
func (m *AdvancedMockGemmaProvider) GetLastInput() *types.MultimodalInput {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastInput
}

// GetLastOutput returns the last output that was generated.
func (m *AdvancedMockGemmaProvider) GetLastOutput() *types.MultimodalOutput {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastOutput
}

// Ensure AdvancedMockGemmaProvider implements iface.MultimodalModel
var _ iface.MultimodalModel = (*AdvancedMockGemmaProvider)(nil)
