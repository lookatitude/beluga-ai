package phi

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

// AdvancedMockPhiProvider provides a comprehensive mock implementation for testing Phi multimodal provider.
type AdvancedMockPhiProvider struct {
	errorToReturn error
	capabilities  *types.ModalityCapabilities
	lastInput     *types.MultimodalInput
	lastOutput    *types.MultimodalOutput
	mock.Mock
	modelName         string
	callCount         int
	simulateDelay     time.Duration
	rateLimitCount    int
	mu                sync.RWMutex
	shouldError       bool
	simulateRateLimit bool
}

// NewAdvancedMockPhiProvider creates a new advanced mock with configurable behavior.
func NewAdvancedMockPhiProvider(modelName string) *AdvancedMockPhiProvider {
	mock := &AdvancedMockPhiProvider{
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
func (m *AdvancedMockPhiProvider) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
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
		ID:            "mock-phi-output-" + time.Now().Format(time.RFC3339Nano),
		InputID:       input.ID,
		ContentBlocks: input.ContentBlocks,
		Metadata:      map[string]any{"provider": "phi", "model": m.modelName},
		Confidence:    0.95,
		Provider:      "phi",
		Model:         m.modelName,
		CreatedAt:     time.Now(),
	}

	m.mu.Lock()
	m.lastOutput = output
	m.mu.Unlock()

	return output, nil
}

// ProcessStream processes a multimodal input and streams results incrementally.
func (m *AdvancedMockPhiProvider) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
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
			ID:            "mock-phi-stream-" + time.Now().Format(time.RFC3339Nano),
			InputID:       input.ID,
			ContentBlocks: input.ContentBlocks,
			Metadata:      map[string]any{"provider": "phi", "model": m.modelName, "streaming": true},
			Confidence:    0.95,
			Provider:      "phi",
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
func (m *AdvancedMockPhiProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.capabilities, nil
}

// SupportsModality checks if this model supports a specific modality.
func (m *AdvancedMockPhiProvider) SupportsModality(ctx context.Context, modality string) (bool, error) {
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
func (m *AdvancedMockPhiProvider) CheckHealth(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError {
		return m.errorToReturn
	}

	return nil
}

// WithMockError configures the mock to return an error.
func (m *AdvancedMockPhiProvider) WithMockError(shouldError bool, err error) *AdvancedMockPhiProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorToReturn = err
	return m
}

// WithMockDelay sets the delay for operations.
func (m *AdvancedMockPhiProvider) WithMockDelay(delay time.Duration) *AdvancedMockPhiProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateDelay = delay
	return m
}

// WithRateLimit simulates rate limiting behavior.
func (m *AdvancedMockPhiProvider) WithRateLimit(enabled bool) *AdvancedMockPhiProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateRateLimit = enabled
	return m
}

// GetCallCount returns the number of times Process or ProcessStream was called.
func (m *AdvancedMockPhiProvider) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetLastInput returns the last input that was processed.
func (m *AdvancedMockPhiProvider) GetLastInput() *types.MultimodalInput {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastInput
}

// GetLastOutput returns the last output that was generated.
func (m *AdvancedMockPhiProvider) GetLastOutput() *types.MultimodalOutput {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastOutput
}

// Ensure AdvancedMockPhiProvider implements iface.MultimodalModel.
var _ iface.MultimodalModel = (*AdvancedMockPhiProvider)(nil)
