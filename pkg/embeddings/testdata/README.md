# Test Data and Fixtures

This directory contains test data and fixtures for the embeddings package.

## Files

### `documents.json`
Contains sample documents and queries for testing embedding functionality:
- **documents**: Array of test documents with metadata (id, title, content, category, tags)
- **queries**: Array of test queries with expected categories

Use this data for:
- Testing embedding quality and consistency
- Performance benchmarking
- Integration testing with real embedding providers
- Semantic search testing

### `configs.yaml`
Contains pre-defined configuration scenarios for different testing needs:
- **basic_config**: Standard configuration with all providers enabled
- **mock_only_config**: Configuration with only mock provider for isolated testing
- **performance_config**: High-performance settings for benchmarking
- **minimal_config**: Minimal configuration for quick unit tests
- **timeout_config**: Custom timeout settings for reliability testing
- **load_test_config**: Configuration optimized for load testing

## Usage

### Loading Documents

```go
import (
    "encoding/json"
    "os"
    "github.com/lookatitude/beluga-ai/pkg/embeddings/testdata"
)

func loadTestDocuments() ([]string, error) {
    file, err := os.Open("testdata/documents.json")
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var data struct {
        Documents []struct {
            Content string `json:"content"`
        } `json:"documents"`
    }

    if err := json.NewDecoder(file).Decode(&data); err != nil {
        return nil, err
    }

    documents := make([]string, len(data.Documents))
    for i, doc := range data.Documents {
        documents[i] = doc.Content
    }

    return documents, nil
}
```

### Loading Configurations

```go
import (
    "gopkg.in/yaml.v3"
    "os"
)

func loadTestConfig(name string) (*embeddings.Config, error) {
    data, err := os.ReadFile("testdata/configs.yaml")
    if err != nil {
        return nil, err
    }

    var configs map[string]string
    if err := yaml.Unmarshal(data, &configs); err != nil {
        return nil, err
    }

    configYAML, ok := configs[name]
    if !ok {
        return nil, fmt.Errorf("config %s not found", name)
    }

    // Parse the YAML config into embeddings.Config
    // (implementation depends on your config loading mechanism)
}
```

## Best Practices

1. **Use appropriate test data sizes**:
   - Unit tests: 1-5 documents
   - Integration tests: 10-50 documents
   - Performance tests: 100+ documents

2. **Test with diverse content**:
   - Different lengths (short sentences to long paragraphs)
   - Different topics and domains
   - Various languages (if supported)
   - Edge cases (empty strings, special characters, unicode)

3. **Configuration testing**:
   - Test with different provider combinations
   - Test timeout and retry scenarios
   - Test invalid configurations
   - Test environment variable overrides

4. **Performance baselines**:
   - Establish performance expectations for each configuration
   - Monitor for regressions in embedding speed
   - Test memory usage patterns

## Adding New Test Data

When adding new test documents or configurations:

1. Ensure documents cover diverse topics and lengths
2. Include metadata for categorization and filtering
3. Test configurations should be realistic and cover common use cases
4. Update this README with new files and their purposes
5. Consider the impact on test execution time
