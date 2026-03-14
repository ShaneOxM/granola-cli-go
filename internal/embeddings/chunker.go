// Package embeddings provides text embedding generation and similarity search.
// It supports multiple providers (Ollama) and chunk-based semantic search.
package embeddings

import (
	"fmt"
	"strings"
)

// Chunk represents a text segment with metadata
type Chunk struct {
	ID        string
	Index     int
	Text      string
	StartTime string
	EndTime   string
	Speakers  []string
}

// ChunkerConfig defines chunking parameters
type ChunkerConfig struct {
	ChunkSize int // Maximum characters per chunk
	Overlap   int // Overlap between chunks (characters)
	MinLength int // Minimum chunk length before splitting
}

// DefaultChunkerConfig returns sensible defaults
func DefaultChunkerConfig() ChunkerConfig {
	return ChunkerConfig{
		ChunkSize: 2000, // Smaller chunks to stay under Ollama's limit
		Overlap:   200,
		MinLength: 100,
	}
}

// Chunker splits text into overlapping chunks
type Chunker struct {
	config ChunkerConfig
}

// NewChunker creates a new chunker with the given config
func NewChunker(config ChunkerConfig) *Chunker {
	if config.ChunkSize <= 0 {
		config.ChunkSize = 3000
	}
	if config.Overlap < 0 {
		config.Overlap = 0
	}
	if config.Overlap >= config.ChunkSize {
		config.Overlap = config.ChunkSize / 10
	}
	if config.MinLength <= 0 {
		config.MinLength = 100
	}

	return &Chunker{config: config}
}

// Chunk splits text into chunks with overlap
func (c *Chunker) Chunk(text string) []Chunk {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Detect speaker turns
	chunks := c.chunkBySpeakerTurns(text)
	if len(chunks) > 0 {
		return chunks
	}

	// Fallback to character-based chunking
	return c.chunkBySize(text)
}

// chunkBySpeakerTurns splits transcript by speaker turns into chunks
func (c *Chunker) chunkBySpeakerTurns(text string) []Chunk {
	lines := strings.Split(text, "\n")
	var chunks []Chunk
	var currentChunk strings.Builder
	currentChunkSize := 0
	var currentStart, currentEnd string
	speakerSet := make(map[string]bool)
	chunkCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this line starts a new speaker turn
		newTurn := strings.HasPrefix(line, "[") && strings.Contains(line, "]:")

		// Start new chunk if needed
		if newTurn && currentChunkSize > 0 && currentChunkSize+len(line)+10 > c.config.ChunkSize {
			if currentChunk.Len() >= c.config.MinLength {
				chunks = append(chunks, Chunk{
					ID:        fmt.Sprintf("chunk_%d", chunkCount),
					Index:     chunkCount,
					Text:      strings.TrimSpace(currentChunk.String()),
					StartTime: currentStart,
					EndTime:   currentEnd,
					Speakers:  c.setToList(speakerSet),
				})
				chunkCount++
			}
			currentChunk.Reset()
			currentChunkSize = 0
			speakerSet = make(map[string]bool)
		}

		// Add line to chunk
		if currentChunkSize == 0 {
			currentStart = line
		}
		currentEnd = line
		currentChunk.WriteString(line + "\n")
		currentChunkSize += len(line) + 1

		// Extract speaker name
		if newTurn {
			parts := strings.SplitN(line, "]:", 2)
			if len(parts) == 2 {
				speakerPart := strings.TrimSpace(parts[0])
				speakerPart = strings.TrimPrefix(speakerPart, "[")
				speakerSet[speakerPart] = true
			}
		}
	}

	// Add final chunk
	if currentChunk.Len() >= c.config.MinLength {
		chunks = append(chunks, Chunk{
			ID:        fmt.Sprintf("chunk_%d", chunkCount),
			Index:     chunkCount,
			Text:      strings.TrimSpace(currentChunk.String()),
			StartTime: currentStart,
			EndTime:   currentEnd,
			Speakers:  c.setToList(speakerSet),
		})
	}

	return chunks
}

// chunkBySize does character-based chunking with overlap
func (c *Chunker) chunkBySize(text string) []Chunk {
	var chunks []Chunk
	totalLen := len(text)

	if totalLen <= c.config.ChunkSize {
		chunks = append(chunks, Chunk{
			ID:    "chunk_0",
			Index: 0,
			Text:  text,
		})
		return chunks
	}

	for start := 0; start < totalLen; {
		end := start + c.config.ChunkSize
		if end > totalLen {
			end = totalLen
		}

		// Try to break at sentence boundary
		if end < totalLen {
			sentenceEnd := c.findSentenceBoundary(text[start:end])
			if sentenceEnd > 0 && sentenceEnd < c.config.ChunkSize {
				end = start + sentenceEnd
			}
		}

		chunks = append(chunks, Chunk{
			ID:    fmt.Sprintf("chunk_%d", len(chunks)),
			Index: len(chunks),
			Text:  strings.TrimSpace(text[start:end]),
		})

		// Move to next chunk with overlap
		start = end - c.config.Overlap
		if start <= 0 {
			start = end
		}
	}

	return chunks
}

// findSentenceBoundary finds the best sentence break point
func (c *Chunker) findSentenceBoundary(text string) int {
	re := strings.NewReplacer(".", ". ", "!", "! ", "?", "? ")
	modified := re.Replace(text)

	parts := strings.Split(modified, ". ")
	if len(parts) > 1 {
		pos := 0
		for i, p := range parts[1:] {
			if len(p) > 0 && p[0] != ' ' {
				continue
			}
			if len(p) > 1 && (p[1] >= 'A' && p[1] <= 'Z') {
				pos = 0
				for j := 0; j <= i; j++ {
					pos += len(parts[j]) + 2
				}
				return pos
			}
		}
	}
	return c.config.ChunkSize
}

// setToList converts a string set to a sorted slice
func (c *Chunker) setToList(speakerSet map[string]bool) []string {
	speakers := make([]string, 0, len(speakerSet))
	for speaker := range speakerSet {
		speakers = append(speakers, speaker)
	}
	return speakers
}
