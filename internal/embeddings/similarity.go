package embeddings

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

// SimilarityEngine handles similarity calculations
type SimilarityEngine struct {
	provider Provider
}

// NewSimilarityEngine creates a new similarity engine
func NewSimilarityEngine(provider Provider) *SimilarityEngine {
	return &SimilarityEngine{
		provider: provider,
	}
}

// CosineSimilarity calculates cosine similarity between two vectors
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// EmbedToBlob converts embedding slice to BLOB format
func EmbedToBlob(embedding []float32) []byte {
	buf := new(bytes.Buffer)
	for _, f := range embedding {
		binary.Write(buf, binary.LittleEndian, f)
	}
	return buf.Bytes()
}

// BlobToEmbedding converts BLOB to embedding slice
func BlobToEmbedding(blob []byte, dimensions int) []float32 {
	if len(blob) == 0 {
		return nil
	}

	embedding := make([]float32, 0, dimensions)
	reader := bytes.NewReader(blob)

	for i := 0; i < dimensions; i++ {
		var f float32
		binary.Read(reader, binary.LittleEndian, &f)
		embedding = append(embedding, f)
	}

	return embedding
}

// SearchQuery represents a search query with filters
type SearchQuery struct {
	Text      string
	MinScore  float64
	Limit     int
	Workspace string
	Since     string
	Before    string
}

// SearchResult represents a search result
type SearchResult struct {
	MeetingID    string
	MeetingTitle string
	Score        float64
	ChunkIndex   int
	ChunkText    string
	StartTime    string
	EndTime      string
	Speakers     []string
	Provider     string
	Model        string
}

// Search performs a semantic search
func (e *SimilarityEngine) Search(ctx context.Context, query SearchQuery, existingChunks []ChunkData) ([]SearchResult, error) {
	if query.Text == "" {
		return nil, fmt.Errorf("query text is required")
	}

	if query.Limit <= 0 {
		query.Limit = 10
	}

	if query.MinScore < 0 {
		query.MinScore = 0
	}

	// Generate query embedding
	queryEmbedding, err := e.provider.GenerateEmbedding(ctx, query.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Calculate similarity for each chunk
	type scoredChunk struct {
		data  ChunkData
		score float64
	}

	var scoredChunks []scoredChunk
	for _, chunk := range existingChunks {
		embedding := BlobToEmbedding(chunk.Embedding, e.provider.Dimensions())
		score := CosineSimilarity(queryEmbedding, embedding)

		if score >= query.MinScore {
			scoredChunks = append(scoredChunks, scoredChunk{
				data:  chunk,
				score: score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].score > scoredChunks[j].score
	})

	// Limit results
	if len(scoredChunks) > query.Limit {
		scoredChunks = scoredChunks[:query.Limit]
	}

	// Convert to search results
	results := make([]SearchResult, len(scoredChunks))
	for i, sc := range scoredChunks {
		results[i] = SearchResult{
			MeetingID:    sc.data.MeetingID,
			MeetingTitle: sc.data.MeetingTitle,
			Score:        sc.score,
			ChunkIndex:   sc.data.ChunkIndex,
			ChunkText:    sc.data.ChunkText,
			StartTime:    sc.data.StartTime,
			EndTime:      sc.data.EndTime,
			Speakers:     sc.data.Speakers,
			Provider:     sc.data.Provider,
			Model:        sc.data.Model,
		}
	}

	return results, nil
}

// ChunkData represents a chunk from the database
type ChunkData struct {
	ID           string
	MeetingID    string
	MeetingTitle string
	ChunkIndex   int
	ChunkText    string
	Embedding    []byte
	Dimensions   int
	Provider     string
	Model        string
	StartTime    string
	EndTime      string
	Speakers     []string
}
