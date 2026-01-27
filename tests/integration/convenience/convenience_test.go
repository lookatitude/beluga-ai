// Package convenience provides integration tests for convenience packages.
// This test suite verifies that the convenience packages work correctly
// with the underlying framework packages.
package convenience

import (
	"context"
	"testing"

	convagent "github.com/lookatitude/beluga-ai/pkg/convenience/agent"
	convcontext "github.com/lookatitude/beluga-ai/pkg/convenience/context"
	"github.com/lookatitude/beluga-ai/pkg/convenience/mock"
	convprovider "github.com/lookatitude/beluga-ai/pkg/convenience/provider"
	convrag "github.com/lookatitude/beluga-ai/pkg/convenience/rag"
	convvoice "github.com/lookatitude/beluga-ai/pkg/convenience/voiceagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockPackage tests the mock convenience package.
func TestMockPackage(t *testing.T) {
	t.Run("mock_llm_creation", func(t *testing.T) {
		llm := mock.NewLLM(mock.WithResponse("test response"))
		require.NotNil(t, llm)

		// Test invoke
		ctx := context.Background()
		result, err := llm.Invoke(ctx, "test input")
		require.NoError(t, err)
		assert.Equal(t, "test response", result)
		assert.Equal(t, 1, llm.CallCount())

		// Test reset
		llm.Reset()
		assert.Equal(t, 0, llm.CallCount())
	})

	t.Run("mock_llm_with_error", func(t *testing.T) {
		testErr := assert.AnError
		llm := mock.NewLLM(mock.WithError(testErr))

		ctx := context.Background()
		_, err := llm.Invoke(ctx, "test input")
		assert.Error(t, err)
		assert.Equal(t, testErr, err)
	})

	t.Run("mock_llm_model_name", func(t *testing.T) {
		llm := mock.NewLLM(mock.WithModelName("gpt-4"))
		assert.Equal(t, "gpt-4", llm.GetModelName())
		assert.Equal(t, "mock", llm.GetProviderName())
	})

	t.Run("mock_embedder_creation", func(t *testing.T) {
		embedder := mock.NewEmbedder(mock.WithDimension(768))
		require.NotNil(t, embedder)

		ctx := context.Background()

		// Test embed documents
		texts := []string{"text1", "text2"}
		embeddings, err := embedder.EmbedDocuments(ctx, texts)
		require.NoError(t, err)
		assert.Len(t, embeddings, 2)
		assert.Len(t, embeddings[0], 768)
		assert.Equal(t, 1, embedder.CallCount())

		// Test embed query
		embedder.Reset()
		embedding, err := embedder.EmbedQuery(ctx, "query")
		require.NoError(t, err)
		assert.Len(t, embedding, 768)
		assert.Equal(t, 1, embedder.CallCount())
	})

	t.Run("mock_stt_creation", func(t *testing.T) {
		stt := mock.NewSTT(mock.WithTranscription("hello world"))
		require.NotNil(t, stt)

		ctx := context.Background()
		text, err := stt.Transcribe(ctx, []byte("audio data"))
		require.NoError(t, err)
		assert.Equal(t, "hello world", text)
		assert.Equal(t, 1, stt.CallCount())
	})

	t.Run("mock_tts_creation", func(t *testing.T) {
		audioData := []byte("synthesized audio")
		tts := mock.NewTTS(mock.WithAudioData(audioData))
		require.NotNil(t, tts)

		ctx := context.Background()
		audio, err := tts.Synthesize(ctx, "hello")
		require.NoError(t, err)
		assert.Equal(t, audioData, audio)
		assert.Equal(t, 1, tts.CallCount())
	})

	t.Run("mock_tool_creation", func(t *testing.T) {
		tool := mock.NewTool("calculator", "Performs calculations", mock.WithResult("42"))
		require.NotNil(t, tool)

		assert.Equal(t, "calculator", tool.Name())
		assert.Equal(t, "Performs calculations", tool.Description())

		ctx := context.Background()
		result, err := tool.Execute(ctx, "2+2")
		require.NoError(t, err)
		assert.Equal(t, "42", result)
		assert.Equal(t, 1, tool.CallCount())
	})
}

// TestProviderPackage tests the provider convenience package.
func TestProviderPackage(t *testing.T) {
	t.Run("list_llm_providers", func(t *testing.T) {
		providers := convprovider.ListLLMs()
		// Should return at least some providers from the registry
		t.Logf("Available LLM providers: %v", providers)
		// Note: may be empty if no providers registered in test environment
	})

	t.Run("list_stt_providers", func(t *testing.T) {
		providers := convprovider.ListSTTs()
		t.Logf("Available STT providers: %v", providers)
	})

	t.Run("list_tts_providers", func(t *testing.T) {
		providers := convprovider.ListTTSs()
		t.Logf("Available TTS providers: %v", providers)
	})

	t.Run("get_all_providers", func(t *testing.T) {
		allProviders := convprovider.GetAllProviders()
		require.NotNil(t, allProviders)

		t.Logf("All providers by type: %+v", allProviders)

		// Verify map structure
		for providerType, infos := range allProviders {
			t.Logf("Provider type %s has %d providers", providerType, len(infos))
			for _, info := range infos {
				assert.NotEmpty(t, info.Name)
				assert.NotEmpty(t, info.Type)
				assert.Equal(t, providerType, info.Type)
			}
		}
	})

	t.Run("provider_info_struct", func(t *testing.T) {
		info := convprovider.ProviderInfo{
			Name:        "test-provider",
			Type:        "llm",
			Description: "A test provider",
		}

		assert.Equal(t, "test-provider", info.Name)
		assert.Equal(t, "llm", info.Type)
		assert.Equal(t, "A test provider", info.Description)
	})
}

// TestContextPackage tests the context convenience package.
func TestContextPackage(t *testing.T) {
	t.Run("builder_creation", func(t *testing.T) {
		builder := convcontext.NewBuilder()
		require.NotNil(t, builder)
	})

	t.Run("add_documents", func(t *testing.T) {
		builder := convcontext.NewBuilder()

		docs := []convcontext.Document{
			{ID: "1", Content: "Document 1 content", Metadata: map[string]any{"source": "test"}},
			{ID: "2", Content: "Document 2 content", Metadata: map[string]any{"source": "test"}},
		}
		scores := []float64{0.9, 0.8}

		builder.AddDocuments(docs, scores)

		retrievedDocs := builder.GetDocuments()
		assert.Len(t, retrievedDocs, 2)
		assert.Equal(t, 0.9, retrievedDocs[0].Score)
		assert.Equal(t, 0.8, retrievedDocs[1].Score)
	})

	t.Run("add_single_document", func(t *testing.T) {
		builder := convcontext.NewBuilder()

		doc := convcontext.Document{ID: "1", Content: "Single doc"}
		builder.AddDocument(doc, 0.95)

		retrievedDocs := builder.GetDocuments()
		assert.Len(t, retrievedDocs, 1)
		assert.Equal(t, 0.95, retrievedDocs[0].Score)
	})

	t.Run("add_history", func(t *testing.T) {
		builder := convcontext.NewBuilder()

		messages := []convcontext.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		}

		builder.AddHistory(messages)

		history := builder.GetHistory()
		assert.Len(t, history, 2)
		assert.Equal(t, "user", history[0].Role)
		assert.Equal(t, "assistant", history[1].Role)
	})

	t.Run("add_message", func(t *testing.T) {
		builder := convcontext.NewBuilder()

		builder.AddMessage("user", "Test message")

		history := builder.GetHistory()
		assert.Len(t, history, 1)
		assert.Equal(t, "user", history[0].Role)
		assert.Equal(t, "Test message", history[0].Content)
	})

	t.Run("with_system_prompt", func(t *testing.T) {
		builder := convcontext.NewBuilder()
		builder.WithSystemPrompt("You are a helpful assistant")

		result := builder.Build()
		assert.Contains(t, result, "You are a helpful assistant")
	})

	t.Run("with_template", func(t *testing.T) {
		builder := convcontext.NewBuilder()
		builder.WithSystemPrompt("Be helpful")
		builder.WithTemplate("System: {{system}}\nDocs: {{documents}}\nHistory: {{history}}")

		result := builder.Build()
		assert.Contains(t, result, "System: Be helpful")
	})

	t.Run("with_metadata", func(t *testing.T) {
		builder := convcontext.NewBuilder()
		builder.WithTemplate("User: {{username}}")
		builder.WithMetadata("username", "TestUser")

		result := builder.Build()
		assert.Contains(t, result, "User: TestUser")
	})

	t.Run("sort_by_score", func(t *testing.T) {
		builder := convcontext.NewBuilder()

		docs := []convcontext.Document{
			{ID: "1", Content: "Low score"},
			{ID: "2", Content: "High score"},
		}
		scores := []float64{0.5, 0.9}

		builder.AddDocuments(docs, scores).SortByScore()

		retrievedDocs := builder.GetDocuments()
		assert.Equal(t, 0.9, retrievedDocs[0].Score)
		assert.Equal(t, 0.5, retrievedDocs[1].Score)
	})

	t.Run("build_for_question", func(t *testing.T) {
		builder := convcontext.NewBuilder()
		builder.WithSystemPrompt("You are helpful")
		builder.AddDocument(convcontext.Document{ID: "1", Content: "Context info"}, 0.9)

		result := builder.BuildForQuestion("What is AI?")
		assert.Contains(t, result, "Question: What is AI?")
	})

	t.Run("build_context_struct", func(t *testing.T) {
		builder := convcontext.NewBuilder()
		builder.WithSystemPrompt("You are helpful")
		builder.AddDocument(convcontext.Document{ID: "1", Content: "Context info"}, 0.9)
		builder.AddMessage("user", "Hello")

		ctx := builder.BuildContext("What is AI?")
		require.NotNil(t, ctx)

		assert.NotEmpty(t, ctx.Content)
		assert.Len(t, ctx.Documents, 1)
		assert.Len(t, ctx.History, 1)
		assert.Greater(t, ctx.TokenCount, 0)
	})

	t.Run("max_document_length", func(t *testing.T) {
		builder := convcontext.NewBuilder()

		longContent := ""
		for i := 0; i < 20000; i++ {
			longContent += "a"
		}

		builder.WithMaxDocumentLength(100)
		builder.AddDocument(convcontext.Document{ID: "1", Content: longContent}, 0.9)

		result := builder.Build()
		// Content should be truncated
		assert.Contains(t, result, "...")
	})

	t.Run("max_history_size", func(t *testing.T) {
		builder := convcontext.NewBuilder()
		builder.WithMaxHistorySize(2)

		for i := 0; i < 5; i++ {
			builder.AddMessage("user", "Message")
		}

		result := builder.Build()
		// Should only have last 2 messages formatted
		t.Logf("Result with limited history: %s", result)
	})
}

// TestAgentPackage tests the agent convenience package.
func TestAgentPackage(t *testing.T) {
	t.Run("builder_creation", func(t *testing.T) {
		builder := convagent.NewBuilder()
		require.NotNil(t, builder)

		// Test defaults
		assert.Equal(t, "assistant", builder.GetName())
		assert.Equal(t, 10, builder.GetMaxTurns())
		assert.Equal(t, "react", builder.GetAgentType())
	})

	t.Run("builder_fluent_api", func(t *testing.T) {
		builder := convagent.NewBuilder().
			WithSystemPrompt("You are helpful").
			WithName("test-agent").
			WithMaxTurns(20).
			WithVerbose(true).
			WithAgentType("tool_calling")

		assert.Equal(t, "You are helpful", builder.GetSystemPrompt())
		assert.Equal(t, "test-agent", builder.GetName())
		assert.Equal(t, 20, builder.GetMaxTurns())
		assert.Equal(t, "tool_calling", builder.GetAgentType())
	})
}

// TestRAGPackage tests the RAG convenience package.
func TestRAGPackage(t *testing.T) {
	t.Run("builder_creation", func(t *testing.T) {
		builder := convrag.NewBuilder()
		require.NotNil(t, builder)

		// Test defaults
		assert.Equal(t, 5, builder.GetTopK())
		assert.Equal(t, 1000, builder.GetChunkSize())
		assert.Equal(t, 200, builder.GetOverlap())
	})

	t.Run("builder_fluent_api", func(t *testing.T) {
		builder := convrag.NewBuilder().
			WithDocumentSource("./docs", "md", "txt").
			WithTopK(10).
			WithChunkSize(500).
			WithOverlap(100)

		docPaths := builder.GetDocPaths()
		assert.Contains(t, docPaths, "./docs")

		extensions := builder.GetExtensions()
		assert.Contains(t, extensions, "md")
		assert.Contains(t, extensions, "txt")

		assert.Equal(t, 10, builder.GetTopK())
		assert.Equal(t, 500, builder.GetChunkSize())
		assert.Equal(t, 100, builder.GetOverlap())
	})

	t.Run("multiple_document_sources", func(t *testing.T) {
		builder := convrag.NewBuilder().
			WithDocumentSource("./docs", "md").
			WithDocumentSource("./data", "json")

		docPaths := builder.GetDocPaths()
		assert.Len(t, docPaths, 2)
		assert.Contains(t, docPaths, "./docs")
		assert.Contains(t, docPaths, "./data")
	})
}

// TestVoiceAgentPackage tests the voice agent convenience package.
func TestVoiceAgentPackage(t *testing.T) {
	t.Run("builder_creation", func(t *testing.T) {
		builder := convvoice.NewBuilder()
		require.NotNil(t, builder)

		// Test defaults
		assert.Equal(t, 50, builder.GetMemorySize())
		assert.False(t, builder.IsMemoryEnabled())
	})

	t.Run("builder_fluent_api", func(t *testing.T) {
		builder := convvoice.NewBuilder().
			WithSTT("deepgram").
			WithTTS("elevenlabs").
			WithVAD("silero").
			WithLLM("openai").
			WithMemory(true).
			WithMemorySize(100).
			WithSystemPrompt("You are a voice assistant")

		assert.Equal(t, "deepgram", builder.GetSTTProvider())
		assert.Equal(t, "elevenlabs", builder.GetTTSProvider())
		assert.Equal(t, "silero", builder.GetVADProvider())
		assert.Equal(t, "openai", builder.GetLLMProvider())
		assert.True(t, builder.IsMemoryEnabled())
		assert.Equal(t, 100, builder.GetMemorySize())
		assert.Equal(t, "You are a voice assistant", builder.GetSystemPrompt())
	})
}

// TestConveniencePackagesConcurrency tests concurrent usage of convenience packages.
func TestConveniencePackagesConcurrency(t *testing.T) {
	const numGoroutines = 10

	t.Run("concurrent_mock_llm_calls", func(t *testing.T) {
		llm := mock.NewLLM(mock.WithResponse("concurrent response"))
		ctx := context.Background()

		done := make(chan struct{}, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer func() { done <- struct{}{} }()

				_, err := llm.Invoke(ctx, "test")
				assert.NoError(t, err)
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		assert.Equal(t, numGoroutines, llm.CallCount())
	})

	t.Run("concurrent_context_building", func(t *testing.T) {
		done := make(chan struct{}, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer func() { done <- struct{}{} }()

				builder := convcontext.NewBuilder()
				builder.WithSystemPrompt("System prompt")
				builder.AddDocument(convcontext.Document{ID: "1", Content: "Content"}, 0.9)
				_ = builder.Build()
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

// BenchmarkConveniencePackages benchmarks convenience package operations.
func BenchmarkConveniencePackages(b *testing.B) {
	b.Run("MockLLM_Invoke", func(b *testing.B) {
		llm := mock.NewLLM(mock.WithResponse("test"))
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = llm.Invoke(ctx, "input")
		}
	})

	b.Run("MockEmbedder_EmbedQuery", func(b *testing.B) {
		embedder := mock.NewEmbedder(mock.WithDimension(1536))
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = embedder.EmbedQuery(ctx, "query")
		}
	})

	b.Run("ContextBuilder_Build", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			builder := convcontext.NewBuilder()
			builder.WithSystemPrompt("You are helpful")
			builder.AddDocument(convcontext.Document{ID: "1", Content: "Content"}, 0.9)
			builder.AddMessage("user", "Hello")
			_ = builder.Build()
		}
	})
}
