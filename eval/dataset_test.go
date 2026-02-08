package eval_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataset_Creation(t *testing.T) {
	ds := eval.Dataset{
		Name: "test-dataset",
		Samples: []eval.EvalSample{
			{Input: "q1", Output: "a1"},
			{Input: "q2", Output: "a2"},
		},
	}

	assert.Equal(t, "test-dataset", ds.Name)
	assert.Len(t, ds.Samples, 2)
}

func TestDataset_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "dataset.json")

	original := eval.Dataset{
		Name: "my-eval-dataset",
		Samples: []eval.EvalSample{
			{
				Input:          "What is Go?",
				Output:         "Go is a programming language.",
				ExpectedOutput: "Go is a statically typed language.",
				RetrievedDocs: []schema.Document{
					{ID: "doc1", Content: "Go documentation"},
				},
				Metadata: map[string]any{
					"latency_ms": 100.0,
					"model":      "gpt-4",
				},
			},
			{
				Input:          "What is Python?",
				Output:         "Python is a programming language.",
				ExpectedOutput: "Python is dynamically typed.",
				Metadata: map[string]any{
					"latency_ms": 120.0,
				},
			},
		},
	}

	// Save
	err := original.Save(path)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(path)
	require.NoError(t, err)

	// Load
	loaded, err := eval.LoadDataset(path)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify content
	assert.Equal(t, original.Name, loaded.Name)
	assert.Len(t, loaded.Samples, 2)
	assert.Equal(t, original.Samples[0].Input, loaded.Samples[0].Input)
	assert.Equal(t, original.Samples[0].Output, loaded.Samples[0].Output)
	assert.Equal(t, original.Samples[0].ExpectedOutput, loaded.Samples[0].ExpectedOutput)
	assert.Equal(t, 100.0, loaded.Samples[0].Metadata["latency_ms"])
	assert.Equal(t, "gpt-4", loaded.Samples[0].Metadata["model"])
}

func TestDataset_LoadNonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.json")

	ds, err := eval.LoadDataset(path)

	require.Error(t, err)
	require.Nil(t, ds)
	assert.True(t, os.IsNotExist(err))
}

func TestDataset_LoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	err := os.WriteFile(path, []byte("not valid json"), 0o644)
	require.NoError(t, err)

	ds, err := eval.LoadDataset(path)

	require.Error(t, err)
	require.Nil(t, ds)
}

func TestDataset_SaveToInvalidPath(t *testing.T) {
	ds := eval.Dataset{
		Name: "test",
		Samples: []eval.EvalSample{
			{Input: "q", Output: "a"},
		},
	}

	// Try to save to a directory that doesn't exist
	err := ds.Save("/nonexistent/path/dataset.json")

	require.Error(t, err)
}

func TestDataset_JSONRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "roundtrip.json")

	original := eval.Dataset{
		Name: "roundtrip-test",
		Samples: []eval.EvalSample{
			{
				Input:          "test input",
				Output:         "test output",
				ExpectedOutput: "expected output",
				RetrievedDocs: []schema.Document{
					{
						ID:       "doc1",
						Content:  "document content",
						Metadata: map[string]any{"source": "test"},
						Score:    0.95,
					},
				},
				Metadata: map[string]any{
					"latency_ms":    150.5,
					"input_tokens":  100,
					"output_tokens": 50,
					"model":         "gpt-4-turbo",
					"nested": map[string]any{
						"key": "value",
					},
				},
			},
		},
	}

	// Save
	err := original.Save(path)
	require.NoError(t, err)

	// Load
	loaded, err := eval.LoadDataset(path)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Name, loaded.Name)
	assert.Len(t, loaded.Samples, 1)

	sample := loaded.Samples[0]
	assert.Equal(t, "test input", sample.Input)
	assert.Equal(t, "test output", sample.Output)
	assert.Equal(t, "expected output", sample.ExpectedOutput)
	assert.Len(t, sample.RetrievedDocs, 1)
	assert.Equal(t, "doc1", sample.RetrievedDocs[0].ID)
	assert.Equal(t, "document content", sample.RetrievedDocs[0].Content)
	assert.Equal(t, 150.5, sample.Metadata["latency_ms"])
	assert.Equal(t, float64(100), sample.Metadata["input_tokens"]) // JSON numbers are float64
	assert.Equal(t, "gpt-4-turbo", sample.Metadata["model"])
}

func TestDataset_EmptyDataset(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.json")

	original := eval.Dataset{
		Name:    "empty-dataset",
		Samples: []eval.EvalSample{},
	}

	// Save
	err := original.Save(path)
	require.NoError(t, err)

	// Load
	loaded, err := eval.LoadDataset(path)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, "empty-dataset", loaded.Name)
	assert.Empty(t, loaded.Samples)
}

func TestDataset_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "format.json")

	ds := eval.Dataset{
		Name: "format-test",
		Samples: []eval.EvalSample{
			{Input: "q1", Output: "a1"},
		},
	}

	// Save
	err := ds.Save(path)
	require.NoError(t, err)

	// Read raw file
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// Verify it's valid JSON
	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Verify indentation (should be pretty-printed)
	assert.Contains(t, string(data), "\n  ")
}

func TestDataset_WithNilMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nil-metadata.json")

	original := eval.Dataset{
		Name: "nil-metadata-test",
		Samples: []eval.EvalSample{
			{
				Input:    "question",
				Output:   "answer",
				Metadata: nil, // Nil metadata
			},
		},
	}

	// Save
	err := original.Save(path)
	require.NoError(t, err)

	// Load
	loaded, err := eval.LoadDataset(path)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, "question", loaded.Samples[0].Input)
	// Nil metadata should be preserved or become empty map
	// (JSON unmarshaling behavior)
}

func TestDataset_WithEmptyRetrievedDocs(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty-docs.json")

	original := eval.Dataset{
		Name: "empty-docs-test",
		Samples: []eval.EvalSample{
			{
				Input:         "question",
				Output:        "answer",
				RetrievedDocs: []schema.Document{}, // Empty slice
			},
		},
	}

	// Save
	err := original.Save(path)
	require.NoError(t, err)

	// Load
	loaded, err := eval.LoadDataset(path)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Empty(t, loaded.Samples[0].RetrievedDocs)
}

func TestDataset_LargeSampleCount(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "large.json")

	// Create dataset with 1000 samples
	samples := make([]eval.EvalSample, 1000)
	for i := range samples {
		samples[i] = eval.EvalSample{
			Input:  "question",
			Output: "answer",
		}
	}

	original := eval.Dataset{
		Name:    "large-dataset",
		Samples: samples,
	}

	// Save
	err := original.Save(path)
	require.NoError(t, err)

	// Load
	loaded, err := eval.LoadDataset(path)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Len(t, loaded.Samples, 1000)
}
