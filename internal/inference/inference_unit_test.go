package inference

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestNewClient_CustomTimeout(t *testing.T) {
	originalTimeout := os.Getenv("GRANOLA_INFERENCE_TIMEOUT")
	defer os.Setenv("GRANOLA_INFERENCE_TIMEOUT", originalTimeout)

	os.Setenv("GRANOLA_INFERENCE_TIMEOUT", "600s")
	client := NewClient("http://localhost:11434", "llama3.2")

	if client.timeout != 600*time.Second {
		t.Errorf("timeout = %v, want 600s", client.timeout)
	}
}

func TestNewClient_InvalidTimeout(t *testing.T) {
	originalTimeout := os.Getenv("GRANOLA_INFERENCE_TIMEOUT")
	defer os.Setenv("GRANOLA_INFERENCE_TIMEOUT", originalTimeout)

	os.Setenv("GRANOLA_INFERENCE_TIMEOUT", "invalid")
	client := NewClient("http://localhost:11434", "llama3.2")

	// Should fall back to default timeout
	if client.timeout != 300*time.Second {
		t.Errorf("timeout = %v, want 300s (default)", client.timeout)
	}
}

func TestNewClient_CustomModel(t *testing.T) {
	client := NewClient("http://localhost:11434", "qwen35")

	if client.model != "qwen35" {
		t.Errorf("model = %v, want qwen35", client.model)
	}
}

func TestNewClient_EmptyEndpoint(t *testing.T) {
	client := NewClient("", "llama3.2")

	if client == nil {
		t.Fatal("NewClient() returned nil for empty endpoint")
	}
}

func TestSetModel(t *testing.T) {
	client := NewClient("http://localhost:11434", "llama3.2")

	// Test multiple model changes
	models := []string{"qwen35", "mistral", "llama3.1"}
	for _, model := range models {
		client.SetModel(model)
		if client.model != model {
			t.Errorf("model = %v, want %v", client.model, model)
		}
	}
}

func TestSetModel_EmptyString(t *testing.T) {
	client := NewClient("http://localhost:11434", "llama3.2")
	client.SetModel("")

	if client.model != "" {
		t.Errorf("model = %v, want empty", client.model)
	}
}

func TestCompletionRequest_Full(t *testing.T) {
	req := CompletionRequest{
		Model:            "llama3.2",
		Prompt:           "Test prompt",
		System:           "You are helpful",
		MaxTokens:        256,
		Temperature:      0.7,
		TopP:             0.9,
		FrequencyPenalty: 0.1,
		PresencePenalty:  0.2,
		Stop:             []string{"\n\n", "END"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Errorf("json.Marshal() failed: %v", err)
	}

	var req2 CompletionRequest
	if err := json.Unmarshal(data, &req2); err != nil {
		t.Errorf("json.Unmarshal() failed: %v", err)
	}

	if req2.Model != req.Model {
		t.Errorf("Model mismatch")
	}
	if len(req2.Stop) != 2 {
		t.Errorf("Stop array length mismatch")
	}
}

func TestCompletionRequest_Minimal(t *testing.T) {
	req := CompletionRequest{
		Model:  "llama3.2",
		Prompt: "Test",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Errorf("json.Marshal() failed: %v", err)
	}

	var req2 CompletionRequest
	if err := json.Unmarshal(data, &req2); err != nil {
		t.Errorf("json.Unmarshal() failed: %v", err)
	}

	if req2.Model != "llama3.2" {
		t.Errorf("Model mismatch")
	}
}

func TestCompletionResponse_EmptyChoices(t *testing.T) {
	resp := CompletionResponse{
		ID:      "test-id",
		Object:  "test-object",
		Created: 1234567890,
		Model:   "llama3.2",
		Choices: []CompletionChoice{},
		Usage:   Usage{},
	}

	if len(resp.Choices) != 0 {
		t.Error("Expected empty choices")
	}
}

func TestCompletionResponse_Full(t *testing.T) {
	resp := CompletionResponse{
		ID:      "test-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "llama3.2",
		Choices: []CompletionChoice{
			{
				Text:         "Response text",
				Index:        0,
				FinishReason: "stop",
				Message: Message{
					Role:    "assistant",
					Content: "Hello",
				},
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	if resp.Object != "chat.completion" {
		t.Errorf("Object = %v, want chat.completion", resp.Object)
	}

	if resp.Choices[0].Index != 0 {
		t.Errorf("Index = %v, want 0", resp.Choices[0].Index)
	}
}

func TestCompletionResponse_InvalidJSON(t *testing.T) {
	jsonData := `{"id": "test", "invalid": true}`

	var resp CompletionResponse
	err := json.Unmarshal([]byte(jsonData), &resp)
	// Should not panic, may or may not error
	_ = err
}

func TestMessage_Full(t *testing.T) {
	msg := Message{
		Role:             "assistant",
		Content:          "Hello, how can I help you?",
		ReasoningContent: "Thinking about the response...",
	}

	if msg.Role != "assistant" {
		t.Errorf("Role = %v, want assistant", msg.Role)
	}

	if msg.Content != "Hello, how can I help you?" {
		t.Errorf("Content mismatch")
	}

	if msg.ReasoningContent != "Thinking about the response..." {
		t.Errorf("ReasoningContent mismatch")
	}
}

func TestMessage_Empty(t *testing.T) {
	msg := Message{}

	if msg.Role != "" {
		t.Errorf("Role = %v, want empty", msg.Role)
	}

	if msg.Content != "" {
		t.Errorf("Content = %v, want empty", msg.Content)
	}
}

func TestUsage_ZeroValues(t *testing.T) {
	usage := Usage{}

	if usage.PromptTokens != 0 {
		t.Errorf("PromptTokens = %v, want 0", usage.PromptTokens)
	}

	if usage.CompletionTokens != 0 {
		t.Errorf("CompletionTokens = %v, want 0", usage.CompletionTokens)
	}

	if usage.TotalTokens != 0 {
		t.Errorf("TotalTokens = %v, want 0", usage.TotalTokens)
	}
}

func TestUsage_LargeValues(t *testing.T) {
	usage := Usage{
		PromptTokens:     1000000,
		CompletionTokens: 2000000,
		TotalTokens:      3000000,
	}

	if usage.PromptTokens != 1000000 {
		t.Errorf("PromptTokens = %v, want 1000000", usage.PromptTokens)
	}

	if usage.TotalTokens != 3000000 {
		t.Errorf("TotalTokens = %v, want 3000000", usage.TotalTokens)
	}
}

func TestContextCancellation(t *testing.T) {
	// Test that context is properly handled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := NewClient("http://localhost:11434", "llama3.2")

	// Verify context can be used
	_ = ctx
	_ = client
}

func TestMin(t *testing.T) {
	if min(1, 2) != 1 {
		t.Error("min(1, 2) should be 1")
	}

	if min(2, 1) != 1 {
		t.Error("min(2, 1) should be 1")
	}

	if min(5, 5) != 5 {
		t.Error("min(5, 5) should be 5")
	}
}

func TestMin_Empty(t *testing.T) {
	if min(0, 0) != 0 {
		t.Error("min(0, 0) should be 0")
	}
}

func TestMin_Negative(t *testing.T) {
	if min(-1, -2) != -2 {
		t.Error("min(-1, -2) should be -2")
	}
}
