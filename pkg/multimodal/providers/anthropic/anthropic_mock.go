package anthropic

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

// AdvancedMockAnthropicProvider provides a comprehensive mock implementation for testing Anthropic multimodal provider.
type AdvancedMockAnthropicProvider struct {
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

// NewAdvancedMockAnthropicProvider creates a new advanced mock with configurable behavior.
func NewAdvancedMockAnthropicProvider(modelName string) *AdvancedMockAnthropicProvider {
	mock := &AdvancedMockAnthropicProvider{
		modelName: modelName,
		capabilities: &types.ModalityCapabilities{
			Text:  true,
			Image: true,
			Audio: false, // Anthropic supports text and image, not audio/video
			Video: false,
			MaxImageSize:          20 * 1024 * 1024, // 20MB
			SupportedImageFormats: []string{"png", "jpeg", "jpg", "gif", "webp"},
		},
	}
	return mock
}

// Process processes a multimodal input and returns a multimodal output.
func (m *AdvancedMockAnthropicProvider) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
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
		ID:            "mock-anthropic-output-" + time.Now().Format(time.RFC3339Nano),
		InputID:       input.ID,
		ContentBlocks: input.ContentBlocks,
		Metadata:      map[string]any{"provider": "anthropic", "model": m.modelName},
		Confidence:    0.95,
		Provider:      "anthropic",
		Model:         m.modelName,
		CreatedAt:     time.Now(),
	}

	m.mu.Lock()
	m.lastOutput = output
	m.mu.Unlock()

	return output, nil
}

// ProcessStream processes a multimodal input and streams results incrementally.
func (m *AdvancedMockAnthropicProvider) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
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
			ID:            "mock-anthropic-stream-" + time.Now().Format(time.RFC3339Nano),
			InputID:       input.ID,
			ContentBlocks: input.ContentBlocks,
			Metadata:      map[string]any{"provider": "anthropic", "model": m.modelName, "streaming": true},
			Confidence:    0.95,
			Provider:      "anthropic",
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
func (m *AdvancedMockAnthropicProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.capabilities, nil
}

// SupportsModality checks if this model supports a specific modality.
func (m *AdvancedMockAnthropicProvider) SupportsModality(ctx context.Context, modality string) (bool, error) {
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
func (m *AdvancedMockAnthropicProvider) CheckHealth(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError {
		return m.errorToReturn
	}

	return nil
}

// WithMockError configures the mock to return an error.
func (m *AdvancedMockAnthropicProvider) WithMockError(shouldError bool, err error) *AdvancedMockAnthropicProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorToReturn = err
	return m
}

// WithMockDelay sets the delay for operations.
func (m *AdvancedMockAnthropicProvider) WithMockDelay(delay time.Duration) *AdvancedMockAnthropicProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateDelay = delay
	return m
}

// WithRateLimit simulates rate limiting behavior.
func (m *AdvancedMockAnthropicProvider) WithRateLimit(enabled bool) *AdvancedMockAnthropicProvider {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateRateLimit = enabled
	return m
}

// GetCallCount returns the number of times Process or ProcessStream was called.
func (m *AdvancedMockAnthropicProvider) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetLastInput returns the last input that was processed.
func (m *AdvancedMockAnthropicProvider) GetLastInput() *types.MultimodalInput {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastInput
}

// GetLastOutput returns the last output that was generated.
func (m *AdvancedMockAnthropicProvider) GetLastOutput() *types.MultimodalOutput {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastOutput
}

// Ensure AdvancedMockAnthropicProvider implements iface.MultimodalModel
var _ iface.MultimodalModel = (*AdvancedMockAnthropicProvider)(nil)
