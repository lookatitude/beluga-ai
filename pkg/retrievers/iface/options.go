// Package iface provides option types and functions for configuring retrievers.
package iface

import "github.com/lookatitude/beluga-ai/pkg/core"

// --- Options for VectorStore/Retriever ---

// optionFunc is a helper type for creating options from functions.
type optionFunc func(*map[string]any)

func (f optionFunc) Apply(config *map[string]any) {
	f(config)
}

// WithEmbedder specifies an Embedder to use for an operation (e.g., AddDocuments).
type WithEmbedderOption struct {
	Embedder Embedder
}

func (o WithEmbedderOption) Apply(config *map[string]any) {
	(*config)["embedder"] = o.Embedder
}

// WithEmbedder creates an option to specify an embedder for operations.
func WithEmbedder(embedder Embedder) core.Option {
	return WithEmbedderOption{Embedder: embedder}
}

// WithScoreThreshold sets a minimum similarity score threshold for retrieved documents.
type WithScoreThresholdOption struct {
	Threshold float32
}

func (o WithScoreThresholdOption) Apply(config *map[string]any) {
	(*config)["score_threshold"] = o.Threshold
}

// WithScoreThreshold creates an option to set a similarity score threshold.
func WithScoreThreshold(threshold float32) core.Option {
	return WithScoreThresholdOption{Threshold: threshold}
}

// WithMetadataFilter applies a filter based on document metadata.
// The exact filter format depends on the VectorStore implementation.
type WithMetadataFilterOption struct {
	Filter map[string]any
}

func (o WithMetadataFilterOption) Apply(config *map[string]any) {
	(*config)["metadata_filter"] = o.Filter
}

// WithMetadataFilter creates an option to apply metadata filtering.
func WithMetadataFilter(filter map[string]any) core.Option {
	return WithMetadataFilterOption{Filter: filter}
}

// WithK specifies the number of documents to retrieve in similarity searches.
type WithKOption struct {
	K int
}

func (o WithKOption) Apply(config *map[string]any) {
	(*config)["k"] = o.K
}

// WithK creates an option to specify the number of documents to retrieve.
func WithK(k int) core.Option {
	return WithKOption{K: k}
}
