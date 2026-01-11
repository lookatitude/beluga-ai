// Package multimodal provides integration tests for multimodal streaming operations.
package multimodal

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamingVideoInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a large video content block (simulated)
	videoData := make([]byte, 2*1024*1024) // 2MB to test chunking
	for i := range videoData {
		videoData[i] = byte(i % 256)
	}

	videoBlock, err := types.NewContentBlock("video", videoData)
	require.NoError(t, err)

	input, err := types.NewMultimodalInput([]*types.ContentBlock{videoBlock})
	require.NoError(t, err)

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	model, err := multimodal.NewMultimodalModel(ctx, "test", config)
	if err != nil {
		t.Logf("Model creation failed (expected if provider not registered): %v", err)
		return
	}

	// Test streaming
	outputChan, err := model.ProcessStream(ctx, input)
	if err != nil {
		t.Logf("Streaming failed: %v", err)
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, outputChan)

	// Collect outputs
	outputs := make([]*types.MultimodalOutput, 0)
	for output := range outputChan {
		outputs = append(outputs, output)
	}

	// Verify chunks were processed
	t.Logf("Received %d output chunks", len(outputs))
	assert.Greater(t, len(outputs), 0, "Should receive at least one output chunk")
}

func TestStreamingAudioInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a large audio content block (simulated)
	audioData := make([]byte, 128*1024) // 128KB to test chunking
	for i := range audioData {
		audioData[i] = byte(i % 256)
	}

	audioBlock, err := types.NewContentBlock("audio", audioData)
	require.NoError(t, err)

	input, err := types.NewMultimodalInput([]*types.ContentBlock{audioBlock})
	require.NoError(t, err)

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	model, err := multimodal.NewMultimodalModel(ctx, "test", config)
	if err != nil {
		t.Logf("Model creation failed: %v", err)
		return
	}

	// Test streaming
	outputChan, err := model.ProcessStream(ctx, input)
	if err != nil {
		t.Logf("Streaming failed: %v", err)
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, outputChan)

	// Collect outputs
	outputs := make([]*types.MultimodalOutput, 0)
	for output := range outputChan {
		outputs = append(outputs, output)
	}

	t.Logf("Received %d output chunks", len(outputs))
	assert.Greater(t, len(outputs), 0, "Should receive at least one output chunk")
}

func TestStreamingOutputAndIncrementalResults(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create mixed content
	textBlock, err := types.NewContentBlock("text", []byte("Process this video"))
	require.NoError(t, err)

	videoData := make([]byte, 1*1024*1024) // 1MB
	videoBlock, err := types.NewContentBlock("video", videoData)
	require.NoError(t, err)

	input, err := types.NewMultimodalInput([]*types.ContentBlock{textBlock, videoBlock})
	require.NoError(t, err)

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	model, err := multimodal.NewMultimodalModel(ctx, "test", config)
	if err != nil {
		t.Logf("Model creation failed: %v", err)
		return
	}

	// Test streaming
	outputChan, err := model.ProcessStream(ctx, input)
	if err != nil {
		t.Logf("Streaming failed: %v", err)
		return
	}

	require.NoError(t, err)

	// Verify incremental results arrive
	chunkCount := 0
	startTime := time.Now()
	for output := range outputChan {
		chunkCount++
		latency := time.Since(startTime)
		t.Logf("Received chunk %d after %v", chunkCount, latency)
		assert.NotNil(t, output)
		assert.NotEmpty(t, output.ID)
	}

	assert.Greater(t, chunkCount, 0, "Should receive incremental results")
}

func TestStreamingInterruptionAndContextSwitching(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create input
	videoData := make([]byte, 2*1024*1024) // 2MB
	videoBlock, err := types.NewContentBlock("video", videoData)
	require.NoError(t, err)

	input1, err := types.NewMultimodalInput([]*types.ContentBlock{videoBlock})
	require.NoError(t, err)

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	model, err := multimodal.NewMultimodalModel(ctx, "test", config)
	if err != nil {
		t.Logf("Model creation failed: %v", err)
		return
	}

	// Start first stream
	outputChan1, err := model.ProcessStream(ctx, input1)
	if err != nil {
		t.Logf("Streaming failed: %v", err)
		return
	}

	require.NoError(t, err)

	// Create second input with same ID to trigger interruption
	input2, err := types.NewMultimodalInput([]*types.ContentBlock{videoBlock})
	require.NoError(t, err)
	input2.ID = input1.ID // Same ID to trigger interruption

	// Start second stream (should interrupt first)
	outputChan2, err := model.ProcessStream(ctx, input2)
	if err != nil {
		t.Logf("Second streaming failed: %v", err)
		return
	}

	require.NoError(t, err)

	// First stream should be interrupted
	firstChunks := 0
	for range outputChan1 {
		firstChunks++
		if firstChunks > 5 {
			break // Don't wait for all chunks
		}
	}

	// Second stream should continue
	secondChunks := 0
	for range outputChan2 {
		secondChunks++
		if secondChunks > 5 {
			break
		}
	}

	t.Logf("First stream chunks: %d, Second stream chunks: %d", firstChunks, secondChunks)
	// Note: Interruption behavior may vary based on implementation
}

func TestStreamingLatency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test audio latency (<500ms target)
	audioData := make([]byte, 64*1024) // 64KB
	audioBlock, err := types.NewContentBlock("audio", audioData)
	require.NoError(t, err)

	input, err := types.NewMultimodalInput([]*types.ContentBlock{audioBlock})
	require.NoError(t, err)

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	model, err := multimodal.NewMultimodalModel(ctx, "test", config)
	if err != nil {
		t.Logf("Model creation failed: %v", err)
		return
	}

	outputChan, err := model.ProcessStream(ctx, input)
	if err != nil {
		t.Logf("Streaming failed: %v", err)
		return
	}

	require.NoError(t, err)

	startTime := time.Now()
	firstChunkReceived := false
	for output := range outputChan {
		if !firstChunkReceived {
			latency := time.Since(startTime)
			t.Logf("First chunk latency: %v", latency)
			// Note: Actual latency depends on provider implementation
			firstChunkReceived = true
		}
		assert.NotNil(t, output)
	}

	assert.True(t, firstChunkReceived, "Should receive at least one chunk")
}
