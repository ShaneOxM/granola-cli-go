// Package embeddings provides text embedding generation and similarity search.
// It supports multiple providers (Ollama) and chunk-based semantic search.
package embeddings

import (
	"context"
)

// Provider defines the interface for embedding generation
type Provider interface {
	// Name returns the provider name (e.g., "ollama")
	Name() string

	// Model returns the model name (e.g., "nomic-embed-text")
	Model() string

	// Dimensions returns the embedding vector dimensions
	Dimensions() int

	// MaxChars returns the maximum text length supported
	MaxChars() int

	// IsAvailable checks if the provider is accessible
	IsAvailable(ctx context.Context) bool

	// GenerateEmbedding creates a single embedding vector
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateBatchEmbeddings creates multiple embeddings efficiently
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}
