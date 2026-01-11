package mock

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// BedrockRuntimeClient is an interface for AWS Bedrock Runtime operations.
// This allows us to mock Bedrock API calls in tests.
type BedrockRuntimeClient interface {
	InvokeModel(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
	InvokeModelWithResponseStream(ctx context.Context, params *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error)
}

// MockBedrockClient is a mock implementation of BedrockRuntimeClient for testing.
type MockBedrockClient struct {
	mu              sync.RWMutex
	responses       map[string]*MockBedrockResponse
	defaultResp     *MockBedrockResponse
	requestCount    map[string]int
	invokeErrors    map[string]error
	streamResponses map[string]*MockBedrockStreamResponse
	defaultStreamResp *MockBedrockStreamResponse
}

// MockBedrockResponse represents a mock Bedrock API response.
type MockBedrockResponse struct {
	Body        []byte
	ContentType string
	Error       error
}

// MockBedrockStreamResponse represents a mock Bedrock streaming API response.
type MockBedrockStreamResponse struct {
	Chunks []types.ResponseStream
	Error  error
}

// NewMockBedrockClient creates a new mock Bedrock client.
func NewMockBedrockClient() *MockBedrockClient {
	return &MockBedrockClient{
		responses:        make(map[string]*MockBedrockResponse),
		requestCount:     make(map[string]int),
		invokeErrors:     make(map[string]error),
		streamResponses:  make(map[string]*MockBedrockStreamResponse),
	}
}

// SetResponse sets a mock response for a specific model ID.
func (m *MockBedrockClient) SetResponse(modelID string, resp *MockBedrockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[modelID] = resp
}

// SetDefaultResponse sets the default response for unmatched model IDs.
func (m *MockBedrockClient) SetDefaultResponse(resp *MockBedrockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultResp = resp
}

// SetError sets an error to return for a specific model ID.
func (m *MockBedrockClient) SetError(modelID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invokeErrors[modelID] = err
}

// GetRequestCount returns the number of times a model ID was invoked.
func (m *MockBedrockClient) GetRequestCount(modelID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestCount[modelID]
}

// InvokeModel implements BedrockRuntimeClient interface.
func (m *MockBedrockClient) InvokeModel(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
	modelID := ""
	if params.ModelId != nil {
		modelID = *params.ModelId
	}

	m.mu.Lock()
	m.requestCount[modelID]++
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for explicit error
	if err, ok := m.invokeErrors[modelID]; ok {
		return nil, err
	}

	// Find matching response
	var resp *MockBedrockResponse
	if r, ok := m.responses[modelID]; ok {
		resp = r
	} else {
		resp = m.defaultResp
	}

	if resp == nil {
		// Default error response
		return nil, &types.ValidationException{
			Message: aws.String("Model not found"),
		}
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	contentType := resp.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	output := &bedrockruntime.InvokeModelOutput{
		Body:        resp.Body,
		ContentType: &contentType,
	}

	return output, nil
}

// SetInvokeModelWithResponseStream sets a mock response for InvokeModelWithResponseStream.
func (m *MockBedrockClient) SetInvokeModelWithResponseStream(modelID string, resp *MockBedrockStreamResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamResponses[modelID] = resp
}

// SetDefaultStreamResponse sets the default stream response.
func (m *MockBedrockClient) SetDefaultStreamResponse(resp *MockBedrockStreamResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultStreamResp = resp
}

// InvokeModelWithResponseStream implements BedrockRuntimeClient interface.
// NOTE: This mock returns a basic output. The GetStream() method is populated by the SDK internally.
// For comprehensive streaming tests, use the actual Bedrock SDK client or implement a more complete mock.
func (m *MockBedrockClient) InvokeModelWithResponseStream(ctx context.Context, params *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error) {
	modelID := ""
	if params.ModelId != nil {
		modelID = *params.ModelId
	}

	m.mu.Lock()
	m.requestCount[modelID]++
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for explicit error
	if err, ok := m.invokeErrors[modelID]; ok {
		return nil, err
	}

	// Find matching stream response
	var streamResp *MockBedrockStreamResponse
	if r, ok := m.streamResponses[modelID]; ok {
		streamResp = r
	} else {
		streamResp = m.defaultStreamResp
	}

	if streamResp != nil && streamResp.Error != nil {
		return nil, streamResp.Error
	}

	// Return basic output - GetStream() is populated by SDK internally
	// For testing, this will need to be handled differently or use real SDK
	return &bedrockruntime.InvokeModelWithResponseStreamOutput{
		ContentType: aws.String("application/json"),
	}, nil
}
