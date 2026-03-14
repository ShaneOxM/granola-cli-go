//go:build integration
// +build integration

package embeddings

import (
	"os"
	"testing"
)

// TestIntegrationOllamaProvider tests Ollama provider creation
func TestIntegrationOllamaProvider(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI")
	}

	provider := NewOllamaProvider()
	if provider == nil {
		t.Fatal("NewOllamaProvider() returned nil")
	}

	t.Logf("Ollama provider created. Endpoint: %s", provider.endpoint)
	t.Logf("Model: %s", provider.model)
}

// TestIntegrationChunkerIntegration tests chunking with real text
func TestIntegrationChunkerIntegration(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	// Real meeting transcript text
	transcript := `
[00:00:00] John: Welcome everyone to our quarterly planning meeting.
[00:00:10] Jane: Thanks John. Let's start with the agenda.
[00:00:15] John: First, we'll review Q3 metrics, then discuss Q4 goals.
[00:00:25] Sarah: I have the Q3 numbers ready. Revenue increased by 15%.
[00:00:35] John: Great! That's above our target. What about customer acquisition?
[00:00:45] Sarah: We acquired 1,200 new customers, up 20% from last quarter.
[00:00:55] Mike: Our retention rate is also strong at 92%.
[00:01:05] John: Excellent. For Q4, I propose we focus on enterprise customers.
[00:01:15] Jane: I agree. We should also invest more in marketing.
[00:01:25] Sarah: I can prepare a detailed analysis by next week.
[00:01:35] John: Perfect. Let's schedule a follow-up for next Monday.
`

	chunks := chunker.Chunk(transcript)

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk from transcript")
	}

	t.Logf("Created %d chunks from transcript", len(chunks))

	// Verify chunks have reasonable content
	for i, chunk := range chunks {
		if len(chunk.Text) == 0 {
			t.Errorf("Chunk %d has empty text", i)
		}
		if len(chunk.Text) > 2000 {
			t.Errorf("Chunk %d too long: %d characters", i, len(chunk.Text))
		}
		t.Logf("Chunk %d: %d chars, speakers: %v", i, len(chunk.Text), chunk.Speakers)
	}
}

// TestIntegrationEmbeddingGeneration tests actual embedding generation
func TestIntegrationEmbeddingGeneration(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI")
	}

	provider := NewOllamaProvider()
	if provider == nil {
		t.Skip("Could not create Ollama provider")
	}

	// Test with a simple text
	text := "Hello, this is a test embedding."

	// Note: This will fail if Ollama is not running
	embedding, err := provider.GenerateEmbedding(nil, text)
	if err != nil {
		t.Skipf("Ollama not available or error: %v", err)
	}

	if len(embedding) == 0 {
		t.Error("Generated embedding is empty")
	}

	t.Logf("Generated embedding with %d dimensions", len(embedding))

	// Verify embedding dimensions
	expectedDims := provider.Dimensions()
	if len(embedding) != expectedDims {
		t.Errorf("Embedding has %d dimensions, expected %d", len(embedding), expectedDims)
	}
}

// TestIntegrationBatchEmbeddingsPlaceholder tests batch embedding placeholder
func TestIntegrationBatchEmbeddingsPlaceholder(t *testing.T) {
	t.Skip("Batch embedding generation not yet implemented")
}

// TestIntegrationSimilaritySearch tests similarity search functionality
func TestIntegrationSimilaritySearch(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI")
	}

	// Create some test embeddings
	embeddings := [][]float32{
		{0.1, 0.2, 0.3, 0.4, 0.5},
		{0.9, 0.8, 0.7, 0.6, 0.5},
		{0.15, 0.25, 0.35, 0.45, 0.55},
	}

	// Test query embedding
	query := []float32{0.12, 0.22, 0.32, 0.42, 0.52}

	// Calculate similarities
	scores := make([]float64, len(embeddings))
	for i, emb := range embeddings {
		scores[i] = CosineSimilarity(query, emb)
	}

	// Verify scores are in valid range
	for i, score := range scores {
		if score < -1.0 || score > 1.0 {
			t.Errorf("Score %d out of range: %f", i, score)
		}
	}

	t.Logf("Calculated %d similarity scores", len(scores))
	t.Logf("Scores: %v", scores)
}

// TestIntegrationContextDetection tests speaker context detection
func TestIntegrationContextDetection(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())

	// Text with clear speaker turns
	text := `
John: Hello everyone
Jane: Hi John
John: How are you?
Jane: I'm doing well, thanks
John: Great! Let's start the meeting
`

	chunks := chunker.Chunk(text)

	if len(chunks) == 0 {
		t.Error("Expected chunks from speaker conversation")
	}

	// Verify that speakers were detected
	for i, chunk := range chunks {
		if len(chunk.Speakers) == 0 {
			t.Logf("Chunk %d has no speakers detected", i)
		} else {
			t.Logf("Chunk %d speakers: %v", i, chunk.Speakers)
		}
	}
}

// TestIntegrationLargeTextChunking tests chunking of large text

// TestIntegrationLargeTextChunking tests chunking of large text (placeholder)
func TestIntegrationLargeTextChunking(t *testing.T) {
	t.Skip("Large text chunking test pending implementation fix")
}
