// Package multimodal provides test utilities and mock implementations for testing.
//
// Test Coverage Exclusions:
//
// The following code paths are intentionally excluded from 100% coverage requirements:
//
// 1. Panic Recovery Paths:
//   - Panic handlers in concurrent test runners
//   - These paths are difficult to test without causing actual panics in test code
//
// 2. Context Cancellation Edge Cases:
//   - Some context cancellation paths in streaming operations are difficult to reliably test
//   - Race conditions between context cancellation and channel operations
//
// 3. Error Paths Requiring System Conditions:
//   - Network errors that require actual network failures (NewContentBlockFromURL)
//   - File system errors that require specific OS conditions (NewContentBlockFromFile)
//   - Memory exhaustion scenarios
//
// 4. Provider-Specific Untestable Paths:
//   - Provider implementations in pkg/multimodal/providers/* require external service failures
//   - These are tested through integration tests rather than unit tests
//   - Provider registry initialization code (init() functions)
//
// 5. Test Utility Functions:
//   - Helper functions in test_utils.go that are used by tests but not directly tested
//   - These are validated through their usage in actual test cases
//
// 6. Initialization Code:
//   - Package init() functions and global variable initialization
//   - Registry registration code that executes automatically
//
// All exclusions are documented here to maintain transparency about coverage goals.
// The target is 100% coverage of testable code paths, excluding the above categories.
package multimodal

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/internal"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

// MockMultimodalModel provides a mock implementation of MultimodalModel for testing.
type MockMultimodalModel struct {
	errorToReturn    error
	capabilities     *types.ModalityCapabilities
	lastInput        *types.MultimodalInput
	lastOutput       *types.MultimodalOutput
	providerName     string
	modelName        string
	simulateDelay    time.Duration
	processCallCount int
	streamCallCount  int
	mu               sync.RWMutex
	shouldError      bool
}

// MockOption configures the behavior of MockMultimodalModel.
type MockOption func(*MockMultimodalModel)

// WithMockError configures the mock to return an error.
func WithMockError(shouldError bool, err error) MockOption {
	return func(m *MockMultimodalModel) {
		m.shouldError = shouldError
		m.errorToReturn = err
	}
}

// WithErrorCode configures the mock to return a MultimodalError with a specific error code.
func WithErrorCode(op, code string) MockOption {
	return func(m *MockMultimodalModel) {
		m.shouldError = true
		m.errorToReturn = NewMultimodalError(op, code, errors.New("mock error"))
	}
}

// WithTimeoutError configures the mock to return a timeout error.
func WithTimeoutError(op string) MockOption {
	return WithErrorCode(op, ErrCodeTimeout)
}

// WithRateLimitError configures the mock to return a rate limit error.
func WithRateLimitError(op string) MockOption {
	return WithErrorCode(op, ErrCodeRateLimit)
}

// WithNetworkError configures the mock to return a network error.
func WithNetworkError(op string) MockOption {
	return WithErrorCode(op, ErrCodeNetworkError)
}

// WithInvalidInputError configures the mock to return an invalid input error.
func WithInvalidInputError(op string) MockOption {
	return WithErrorCode(op, ErrCodeInvalidInput)
}

// WithProviderError configures the mock to return a provider error.
func WithProviderError(op string) MockOption {
	return WithErrorCode(op, ErrCodeProviderError)
}

// WithMockDelay sets the delay for operations.
func WithMockDelay(delay time.Duration) MockOption {
	return func(m *MockMultimodalModel) {
		m.simulateDelay = delay
	}
}

// NewMockMultimodalModel creates a new mock multimodal model.
func NewMockMultimodalModel(providerName, modelName string, capabilities *types.ModalityCapabilities, opts ...MockOption) *MockMultimodalModel {
	if capabilities == nil {
		capabilities = &types.ModalityCapabilities{
			Text:  true,
			Image: true,
			Audio: true,
			Video: true,
		}
	}
	m := &MockMultimodalModel{
		providerName: providerName,
		modelName:    modelName,
		capabilities: capabilities,
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Process processes a multimodal input and returns a multimodal output.
func (m *MockMultimodalModel) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
	m.mu.Lock()
	m.processCallCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	delay := m.simulateDelay
	m.lastInput = input
	m.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	if shouldError {
		return nil, errorToReturn
	}

	// Create a mock output
	output := &types.MultimodalOutput{
		ID:            "mock-output-" + time.Now().Format(time.RFC3339Nano),
		InputID:       input.ID,
		ContentBlocks: input.ContentBlocks,
		Metadata:      make(map[string]any),
		Confidence:    0.95,
		Provider:      m.providerName,
		Model:         m.modelName,
		CreatedAt:     time.Now(),
	}

	m.mu.Lock()
	m.lastOutput = output
	m.mu.Unlock()

	return output, nil
}

// ProcessStream processes a multimodal input and streams results incrementally.
func (m *MockMultimodalModel) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
	m.mu.Lock()
	m.streamCallCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	m.lastInput = input
	m.mu.Unlock()

	if shouldError {
		return nil, errorToReturn
	}

	ch := make(chan *types.MultimodalOutput, 1)
	go func() {
		defer close(ch)

		// Create a mock output
		output := &types.MultimodalOutput{
			ID:            "mock-output-stream-" + time.Now().Format(time.RFC3339Nano),
			InputID:       input.ID,
			ContentBlocks: input.ContentBlocks,
			Metadata:      make(map[string]any),
			Confidence:    0.95,
			Provider:      m.providerName,
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
func (m *MockMultimodalModel) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.capabilities, nil
}

// SupportsModality checks if this model supports a specific modality.
func (m *MockMultimodalModel) SupportsModality(ctx context.Context, modality string) (bool, error) {
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
func (m *MockMultimodalModel) CheckHealth(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldError {
		return m.errorToReturn
	}

	// Mock health check always passes unless configured to error
	return nil
}

// MockMultimodalProvider provides a mock implementation of MultimodalProvider for testing.
type MockMultimodalProvider struct {
	errorToReturn error
	capabilities  *types.ModalityCapabilities
	name          string
	shouldError   bool
}

// NewMockMultimodalProvider creates a new mock multimodal provider.
func NewMockMultimodalProvider(name string, capabilities *types.ModalityCapabilities) *MockMultimodalProvider {
	if capabilities == nil {
		capabilities = &types.ModalityCapabilities{
			Text:  true,
			Image: true,
			Audio: true,
			Video: true,
		}
	}
	return &MockMultimodalProvider{
		name:         name,
		capabilities: capabilities,
	}
}

// CreateModel creates a new mock model instance.
func (m *MockMultimodalProvider) CreateModel(ctx context.Context, config map[string]any) (iface.MultimodalModel, error) {
	if m.shouldError {
		return nil, m.errorToReturn
	}
	modelName := "mock-model"
	if modelCfg, ok := config["Model"].(string); ok {
		modelName = modelCfg
	}
	return NewMockMultimodalModel(m.name, modelName, m.capabilities), nil
}

// GetName returns the name of this provider.
func (m *MockMultimodalProvider) GetName() string {
	return m.name
}

// GetCapabilities returns the capabilities of this provider.
func (m *MockMultimodalProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	return m.capabilities, nil
}

// ValidateConfig validates the provider-specific configuration.
func (m *MockMultimodalProvider) ValidateConfig(ctx context.Context, config map[string]any) error {
	if m.shouldError {
		return m.errorToReturn
	}
	// Basic validation
	if _, ok := config["Provider"].(string); !ok {
		return errors.New("provider not specified")
	}
	return nil
}

// MockContentBlock provides a mock implementation of ContentBlock for testing.
type MockContentBlock struct {
	metadata    map[string]any
	contentType string
	url         string
	filePath    string
	mimeType    string
	data        []byte
	size        int64
}

// NewMockContentBlock creates a new mock content block.
func NewMockContentBlock(contentType string, data []byte) *MockContentBlock {
	return &MockContentBlock{
		contentType: contentType,
		data:        data,
		size:        int64(len(data)),
		metadata:    make(map[string]any),
	}
}

// GetType returns the content type.
func (m *MockContentBlock) GetType() string {
	return m.contentType
}

// GetData returns the raw content data.
func (m *MockContentBlock) GetData() []byte {
	return m.data
}

// GetURL returns the URL to the content.
func (m *MockContentBlock) GetURL() string {
	return m.url
}

// GetFilePath returns the file path to the content.
func (m *MockContentBlock) GetFilePath() string {
	return m.filePath
}

// GetMIMEType returns the MIME type of the content.
func (m *MockContentBlock) GetMIMEType() string {
	return m.mimeType
}

// GetSize returns the size of the content in bytes.
func (m *MockContentBlock) GetSize() int64 {
	return m.size
}

// GetMetadata returns additional metadata.
func (m *MockContentBlock) GetMetadata() map[string]any {
	return m.metadata
}

// TestBaseMultimodalModel is a public wrapper for internal.BaseMultimodalModel for testing.
// This allows integration tests to access base model functionality without importing internal.
type TestBaseMultimodalModel struct {
	*internal.BaseMultimodalModel
}

// NewTestBaseMultimodalModel creates a new test base multimodal model.
// This is a public wrapper around internal.NewBaseMultimodalModel for use in integration tests.
// Config can be either a Config struct or map[string]any.
// Capabilities can be either *ModalityCapabilities or *types.ModalityCapabilities.
func NewTestBaseMultimodalModel(providerName, modelName string, config, capabilities any) *TestBaseMultimodalModel {
	var configMap map[string]any
	switch v := config.(type) {
	case Config:
		configMap = map[string]any{
			"Provider": v.Provider,
			"Model":    v.Model,
			"APIKey":   v.APIKey,
		}
	case map[string]any:
		configMap = v
	default:
		configMap = make(map[string]any)
	}

	// Convert capabilities to *types.ModalityCapabilities
	var typeCapabilities *types.ModalityCapabilities
	switch v := capabilities.(type) {
	case *types.ModalityCapabilities:
		typeCapabilities = v
	case *ModalityCapabilities:
		typeCapabilities = &types.ModalityCapabilities{
			Text:                  v.Text,
			Image:                 v.Image,
			Audio:                 v.Audio,
			Video:                 v.Video,
			MaxImageSize:          v.MaxImageSize,
			MaxAudioSize:          v.MaxAudioSize,
			MaxVideoSize:          v.MaxVideoSize,
			SupportedImageFormats: v.SupportedImageFormats,
			SupportedAudioFormats: v.SupportedAudioFormats,
			SupportedVideoFormats: v.SupportedVideoFormats,
		}
	default:
		// Default capabilities if nil or unknown type
		typeCapabilities = &types.ModalityCapabilities{
			Text:  true,
			Image: true,
			Audio: true,
			Video: true,
		}
	}

	baseModel := internal.NewBaseMultimodalModel(providerName, modelName, configMap, typeCapabilities)
	return &TestBaseMultimodalModel{BaseMultimodalModel: baseModel}
}

// TestMultimodalAgentExtension is a public wrapper for internal.MultimodalAgentExtension for testing.
// This allows integration tests to access agent extension functionality without importing internal.
type TestMultimodalAgentExtension struct {
	*internal.MultimodalAgentExtension
}

// NewTestMultimodalAgentExtension creates a new test multimodal agent extension.
// This is a public wrapper around internal.NewMultimodalAgentExtension for use in integration tests.
func NewTestMultimodalAgentExtension(model *TestBaseMultimodalModel) *TestMultimodalAgentExtension {
	extension := internal.NewMultimodalAgentExtension(model.BaseMultimodalModel)
	return &TestMultimodalAgentExtension{MultimodalAgentExtension: extension}
}
