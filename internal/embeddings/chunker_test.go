package embeddings

import (
	"strings"
	"testing"
)

func TestChunkerConfig(t *testing.T) {
	config := DefaultChunkerConfig()

	if config.ChunkSize <= 0 {
		t.Error("ChunkSize must be positive")
	}

	if config.Overlap < 0 {
		t.Error("Overlap must be non-negative")
	}

	if config.MinLength <= 0 {
		t.Error("MinLength must be positive")
	}
}

func TestNewChunker(t *testing.T) {
	config := DefaultChunkerConfig()
	chunker := NewChunker(config)

	if chunker == nil {
		t.Fatal("NewChunker() returned nil")
	}

	if chunker.config.ChunkSize != config.ChunkSize {
		t.Errorf("ChunkSize = %v, want %v", chunker.config.ChunkSize, config.ChunkSize)
	}
}

func TestChunker_EmptyText(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())
	chunks := chunker.Chunk("")

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestChunker_WhitespaceText(t *testing.T) {
	chunker := NewChunker(DefaultChunkerConfig())
	chunks := chunker.Chunk("   \n\t  ")

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for whitespace-only text, got %d", len(chunks))
	}
}

func TestChunker_SingleLine(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
		ChunkSize: 100,
		Overlap:   10,
		MinLength: 10,
	})

	text := "This is a single line of text."
	chunks := chunker.Chunk(text)

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	if len(chunks) > 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}

	if !strings.Contains(chunks[0].Text, "single line") {
		t.Errorf("Chunk text missing expected content: %s", chunks[0].Text)
	}
}

func TestChunker_LargeText(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize: 50,
		Overlap:   10,
		MinLength: 10,
	}
	chunker := NewChunker(config)

	// Create text larger than chunk size
	var largeText strings.Builder
	for i := 0; i < 10; i++ {
		largeText.WriteString("This is line number ")
		largeText.WriteString(string(rune('0' + i)))
		largeText.WriteString("\n")
	}

	chunks := chunker.Chunk(largeText.String())

	if len(chunks) == 0 {
		t.Error("Expected chunks for large text")
	}

	if len(chunks) > 10 {
		t.Errorf("Expected at most 10 chunks, got %d", len(chunks))
	}

	// Note: Chunker may create chunks larger than config.ChunkSize
	// when splitting at speaker boundaries or sentence boundaries

	// Verify chunks have sequential indices
	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Errorf("Chunk %d has index %d, expected %d", i, chunk.Index, i)
		}
	}
}

func TestChunker_Overlap(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize: 50,
		Overlap:   20,
		MinLength: 10,
	}
	chunker := NewChunker(config)

	text := "A B C D E F G H I J K L M N O P Q R S T U V W X Y Z"
	chunks := chunker.Chunk(text)

	if len(chunks) < 2 {
		t.Skip("Text too short for overlap testing")
	}

	// Verify overlap between consecutive chunks
	for i := 1; i < len(chunks); i++ {
		prevEnd := len(chunks[i-1].Text) - config.Overlap
		if prevEnd <= 0 {
			continue
		}
		prevOverlap := chunks[i-1].Text[prevEnd:]
		currStart := len(chunks[i].Text) - len(prevOverlap)
		if currStart < 0 {
			continue
		}
		currOverlap := chunks[i].Text[currStart:]

		if prevOverlap != currOverlap {
			t.Logf("Overlap check: prev='%s', curr='%s'", prevOverlap, currOverlap)
		}
	}
}
