# Schema Package Mock Infrastructure

This directory contains auto-generated mocks for all schema package interfaces following constitutional patterns.

## Mock Generation

Mocks are generated using mockery with the configuration in `.mockery.yaml`:

```bash
# Generate all mocks
cd pkg/schema
go generate ./...

# Generate specific interface mocks
mockery --name=Message --dir=./iface --output=./internal/mock
```

## Available Mocks

- `MockMessage` - Mock implementation of Message interface
- `MockChatHistory` - Mock implementation of ChatHistory interface  
- `MockSchemaValidator` - Mock implementation of validation interfaces

## Usage in Tests

```go
import "github.com/lookatitude/beluga-ai/pkg/schema/internal/mock"

func TestExample(t *testing.T) {
    mockMsg := &mock.MockMessage{}
    mockMsg.On("GetContent").Return("test content")
    mockMsg.On("GetType").Return(iface.RoleHuman)
    
    // Use mock in test
    result := functionUnderTest(mockMsg)
    
    mockMsg.AssertExpectations(t)
}
```

## Constitutional Compliance

All mocks follow the Beluga AI Framework constitutional requirements:
- Interface-focused design (ISP)
- Comprehensive test coverage
- Performance benchmarking support
- Structured error handling
