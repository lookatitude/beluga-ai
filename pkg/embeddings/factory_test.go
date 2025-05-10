package embeddings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/embeddings" // Import the package being tested
	// "github.com/lookatitude/beluga-ai/pkg/embeddings/iface" // This was unused
	"github.com/lookatitude/beluga-ai/pkg/embeddings/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a temporary config file for testing
func createTempConfigFile(t *testing.T, content string) (string, string, func()) {
	t.Helper()
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(content), 0600)
	require.NoError(t, err)
	return tempDir, "config", func() { os.Remove(configFile) }
}

func TestNewEmbedderProvider(t *testing.T) {
	t.Run("ValidMockProvider", func(t *testing.T) {
		configContent := `
embeddings:
  provider: "mock"
  mock:
    dimension: 128
    seed: 42
    randomize_nil: false
`
		tempDir, configName, cleanup := createTempConfigFile(t, configContent)
		defer cleanup()

		vp, err := config.NewViperProvider(configName, []string{tempDir}, "")
		require.NoError(t, err)

		embedder, err := embeddings.NewEmbedderProvider(vp)
		require.NoError(t, err)
		require.NotNil(t, embedder)

		_, ok := embedder.(*embeddings.MockEmbedder)
		assert.True(t, ok, "Expected a MockEmbedder instance")

		mockEmb := embedder.(*embeddings.MockEmbedder)
		dim, _ := mockEmb.GetDimension(nil)
		assert.Equal(t, 128, dim)
	})

	t.Run("ValidOpenAIProvider", func(t *testing.T) {
		configContent := `
embeddings:
  provider: "openai"
  openai:
    api_key: "test-openai-key"
    model: "text-embedding-ada-002"
    timeout_seconds: 60
`
		tempDir, configName, cleanup := createTempConfigFile(t, configContent)
		defer cleanup()

		vp, err := config.NewViperProvider(configName, []string{tempDir}, "")
		require.NoError(t, err)

		embedder, err := embeddings.NewEmbedderProvider(vp)
		require.NoError(t, err)
		require.NotNil(t, embedder)

		_, ok := embedder.(*openai.OpenAIEmbedder)
		assert.True(t, ok, "Expected an OpenAIEmbedder instance")

		openaiEmb := embedder.(*openai.OpenAIEmbedder)
		dim, _ := openaiEmb.GetDimension(nil)
		assert.Equal(t, openai.Ada002Dimension, dim)
	})

	t.Run("UnknownProvider", func(t *testing.T) {
		configContent := `
embeddings:
  provider: "unknown-provider"
`
		tempDir, configName, cleanup := createTempConfigFile(t, configContent)
		defer cleanup()

		vp, err := config.NewViperProvider(configName, []string{tempDir}, "")
		require.NoError(t, err)

		_, err = embeddings.NewEmbedderProvider(vp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown or unregistered embedding provider specified: unknown-provider")
	})

	t.Run("MissingProviderKey", func(t *testing.T) {
		configContent := `
embeddings:
  # provider field is missing
  openai:
    api_key: "test-openai-key"
`
		tempDir, configName, cleanup := createTempConfigFile(t, configContent)
		defer cleanup()

		vp, err := config.NewViperProvider(configName, []string{tempDir}, "")
		require.NoError(t, err)

		_, err = embeddings.NewEmbedderProvider(vp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "embedding provider key 'embeddings.provider' not found in configuration")
	})

	t.Run("EmptyProviderValue", func(t *testing.T) {
		configContent := `
embeddings:
  provider: ""
`
		tempDir, configName, cleanup := createTempConfigFile(t, configContent)
		defer cleanup()

		vp, err := config.NewViperProvider(configName, []string{tempDir}, "")
		require.NoError(t, err)

		_, err = embeddings.NewEmbedderProvider(vp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "embedding provider not specified in configuration under 'embeddings.provider'")
	})

	t.Run("MissingSpecificProviderConfigOpenAI", func(t *testing.T) {
		configContent := `
embeddings:
  provider: "openai"
  # openai specific config block is missing
`
		tempDir, configName, cleanup := createTempConfigFile(t, configContent)
		defer cleanup()

		vp, err := config.NewViperProvider(configName, []string{tempDir}, "")
		require.NoError(t, err)

		// Need to ensure openai provider is registered for this test to run properly
		// by importing its package for the side effect of its init() function.
		_ = openai.Ada002Dimension // This is a trick to ensure openai package is imported

		_, err = embeddings.NewEmbedderProvider(vp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "OpenAI API key is required and was not found in config")
	})

	t.Run("MissingSpecificProviderConfigMock", func(t *testing.T) {
		configContent := `
embeddings:
  provider: "mock"
  # mock specific config block is missing
`
		tempDir, configName, cleanup := createTempConfigFile(t, configContent)
		defer cleanup()

		vp, err := config.NewViperProvider(configName, []string{tempDir}, "")
		require.NoError(t, err)

		embedder, err := embeddings.NewEmbedderProvider(vp)
		require.NoError(t, err)
		require.NotNil(t, embedder)
		_, ok := embedder.(*embeddings.MockEmbedder)
		assert.True(t, ok)
		mockEmb := embedder.(*embeddings.MockEmbedder)
		dim, _ := mockEmb.GetDimension(nil)
		assert.Equal(t, 128, dim) // Default dimension set in mock_embedder.go init or NewMockEmbedder
	})

	// This test is removed as UnmarshalKey is not used directly by the factory anymore.
	// The factory relies on GetString for the provider and then provider-specific unmarshalling.
	/*
	t.Run("InvalidEmbeddingsFactoryConfigStructure", func(t *testing.T) {
		configContent := `
embeddings: "not_a_map"
`
		tempDir, configName, cleanup := createTempConfigFile(t, configContent)
		defer cleanup()

		vp, err := config.NewViperProvider(configName, []string{tempDir}, "")
		require.NoError(t, err)

		_, err = embeddings.NewEmbedderProvider(vp)
		require.Error(t, err)
		// The error will now be about "embedding provider key 'embeddings.provider' not found"
		// because GetString("embeddings.provider") on a string value will likely fail or return empty.
		assert.Contains(t, err.Error(), "embedding provider key 'embeddings.provider' not found in configuration")
	})
	*/
}

