package inference

import (
	"testing"
)

func TestClient(t *testing.T) {
	client := NewClient("http://localhost:11434", "llama3.2")

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.endpoint != "http://localhost:11434" {
		t.Errorf("endpoint = %v, want http://localhost:11434", client.endpoint)
	}

	if client.model != "llama3.2" {
		t.Errorf("model = %v, want llama3.2", client.model)
	}
}

func TestClient_DefaultTimeout(t *testing.T) {
	client := NewClient("http://localhost:11434", "llama3.2")

	if client.timeout == 0 {
		t.Error("timeout should not be zero")
	}
}

func TestClient_SetModel(t *testing.T) {
	client := NewClient("http://localhost:11434", "llama3.2")

	client.SetModel("qwen35")

	if client.model != "qwen35" {
		t.Errorf("model = %v, want qwen35", client.model)
	}
}

func TestCompletionRequest(t *testing.T) {
	req := CompletionRequest{
		Model:       "llama3.2",
		Prompt:      "Test prompt",
		System:      "You are a helpful assistant",
		MaxTokens:   100,
		Temperature: 0.7,
		TopP:        0.9,
	}

	if req.Model != "llama3.2" {
		t.Errorf("Model = %v, want llama3.2", req.Model)
	}

	if req.Prompt != "Test prompt" {
		t.Errorf("Prompt = %v, want Test prompt", req.Prompt)
	}

	if req.MaxTokens != 100 {
		t.Errorf("MaxTokens = %v, want 100", req.MaxTokens)
	}

	if req.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", req.Temperature)
	}
}

func TestCompletionResponse(t *testing.T) {
	resp := CompletionResponse{
		ID:      "test-id",
		Object:  "test-object",
		Created: 1234567890,
		Model:   "llama3.2",
		Choices: []CompletionChoice{
			{
				Text:         "Test response",
				Index:        0,
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	if resp.ID != "test-id" {
		t.Errorf("ID = %v, want test-id", resp.ID)
	}

	if len(resp.Choices) != 1 {
		t.Errorf("Choices length = %v, want 1", len(resp.Choices))
	}

	if resp.Choices[0].Text != "Test response" {
		t.Errorf("Choice[0].Text = %v, want Test response", resp.Choices[0].Text)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %v, want 30", resp.Usage.TotalTokens)
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: "Hello, how can I help you?",
	}

	if msg.Role != "assistant" {
		t.Errorf("Role = %v, want assistant", msg.Role)
	}

	if msg.Content != "Hello, how can I help you?" {
		t.Errorf("Content = %v, want Hello, how can I help you?", msg.Content)
	}
}

func TestEmptyResponseHandling(t *testing.T) {
	resp := CompletionResponse{
		ID:      "test-id",
		Object:  "test-object",
		Created: 1234567890,
		Model:   "llama3.2",
		Choices: []CompletionChoice{}, // Empty choices
		Usage:   Usage{},
	}

	if len(resp.Choices) != 0 {
		t.Error("Expected empty choices")
	}
}

func TestUsage(t *testing.T) {
	usage := Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}

	if usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %v, want 10", usage.PromptTokens)
	}

	if usage.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %v, want 20", usage.CompletionTokens)
	}

	if usage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %v, want 30", usage.TotalTokens)
	}
}
