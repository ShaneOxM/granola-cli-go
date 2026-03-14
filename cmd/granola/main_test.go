package main

import (
	"fmt"
	"testing"

	"github.com/ShaneOxM/granola-cli-go/internal/embeddings"
)

func TestIsAICommand_Summarize(t *testing.T) {
	if !isAICommand("summarize") {
		t.Error("summarize should be recognized as AI command")
	}
}

func TestIsAICommand_ExtractActions(t *testing.T) {
	if !isAICommand("actions") {
		t.Error("actions should be recognized as AI command")
	}
}

func TestIsAICommand_KeyTakeaways(t *testing.T) {
	if !isAICommand("key-takeaways") {
		t.Error("key-takeaways should be recognized as AI command")
	}
}

func TestIsAICommand_Unknown(t *testing.T) {
	if isAICommand("unknown-command") {
		t.Error("unknown-command should not be recognized as AI command")
	}
}

func TestIsAICommand_Empty(t *testing.T) {
	if isAICommand("") {
		t.Error("empty string should not be recognized as AI command")
	}
}

func TestSplitForEmbedding(t *testing.T) {
	text := "alpha beta gamma delta epsilon zeta eta theta iota kappa lambda"
	parts := splitForEmbedding(text, 20)
	if len(parts) < 2 {
		t.Fatalf("expected multiple parts, got %d", len(parts))
	}
	for _, p := range parts {
		if len(p) > 20 {
			t.Fatalf("part exceeds max chars: %d", len(p))
		}
	}
}

func TestIsEmbeddingTooLong(t *testing.T) {
	if !isEmbeddingTooLong(fmt.Errorf("API error 500: input length exceeds the context length")) {
		t.Fatal("expected context-length detection")
	}
	if isEmbeddingTooLong(fmt.Errorf("other error")) {
		t.Fatal("did not expect non-context error to match")
	}
}

func TestLexicalFallbackResults(t *testing.T) {
	chunks := []embeddings.ChunkData{
		{MeetingID: "m1", MeetingTitle: "SCS Follow-up", ChunkText: "matching engine and waitlist roadmap"},
		{MeetingID: "m2", MeetingTitle: "Other", ChunkText: "nothing relevant here"},
	}
	results := lexicalFallbackResults("matching waitlist", chunks, 5)
	if len(results) == 0 || results[0].MeetingID != "m1" {
		t.Fatalf("unexpected fallback results: %+v", results)
	}
}

func TestBestSplitPoint(t *testing.T) {
	text := "one two three. four five six. seven eight nine"
	split := bestSplitPoint(text, 20)
	if split <= 0 || split > 20 {
		t.Fatalf("unexpected split point %d", split)
	}
}
