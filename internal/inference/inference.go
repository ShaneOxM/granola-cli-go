// Package inference provides AI inference client for meeting transcript analysis.

package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client handles requests to remote inference servers
type Client struct {
	endpoint   string
	model      string
	apiKey     string
	timeout    time.Duration
	httpClient *http.Client
}

// CompletionRequest represents an OpenAI-compatible completion request
type CompletionRequest struct {
	Model            string   `json:"model"`
	Prompt           string   `json:"prompt"`
	System           string   `json:"system,omitempty"`
	MaxTokens        int      `json:"max_tokens,omitempty"`
	Temperature      float32  `json:"temperature,omitempty"`
	TopP             float32  `json:"top_p,omitempty"`
	FrequencyPenalty float32  `json:"frequency_penalty,omitempty"`
	PresencePenalty  float32  `json:"presence_penalty,omitempty"`
	Stop             []string `json:"stop,omitempty"`
}

// Message represents a message in the response
type Message struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// CompletionChoice represents a completion response choice
type CompletionChoice struct {
	Text         string  `json:"text,omitempty"`
	Message      Message `json:"message"`
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"`
}

// CompletionResponse represents an OpenAI-compatible completion response
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   Usage              `json:"usage"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewClient creates a new inference client
func NewClient(endpoint, model string) *Client {
	timeout := 300 * time.Second // 5 minute default timeout

	if timeoutStr := os.Getenv("GRANOLA_INFERENCE_TIMEOUT"); timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	return &Client{
		endpoint: endpoint,
		model:    model,
		apiKey:   os.Getenv("GRANOLA_INFERENCE_API_KEY"),
		timeout:  timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SetModel updates the model to use
func (c *Client) SetModel(model string) {
	c.model = model
}

// Summarize generates a summary of the given text
func (c *Client) Summarize(text string, maxTokens int) (string, error) {
	systemPrompt := "You are a helpful assistant that summarizes meeting transcripts. Provide concise, actionable summaries focusing on key decisions, action items, and important discussion points. Use bullet points for clarity."

	prompt := fmt.Sprintf("Summarize the following meeting transcript. Focus on key decisions, action items, and important points:\n\n%s\n\n---\n\nSummary:", text)

	req := CompletionRequest{
		Model:       c.model,
		Prompt:      prompt,
		System:      systemPrompt,
		MaxTokens:   maxTokens,
		Temperature: 0.3,
		TopP:        0.9,
	}

	return c.complete(context.Background(), req)
}

// ExtractActions extracts action items from text
func (c *Client) ExtractActions(text string, maxTokens int) (string, error) {
	systemPrompt := "You are an assistant that extracts action items from meeting transcripts. Identify all actionable tasks, who is responsible, and deadlines if mentioned. Format as a clear bulleted list."

	prompt := fmt.Sprintf("Extract all action items from this meeting transcript. List tasks, owners, and deadlines:\n\n%s\n\n---\n\nAction Items:", text)

	req := CompletionRequest{
		Model:       c.model,
		Prompt:      prompt,
		System:      systemPrompt,
		MaxTokens:   maxTokens,
		Temperature: 0.2,
		TopP:        0.9,
	}

	return c.complete(context.Background(), req)
}

// KeyTakeaways extracts key takeaways from text
func (c *Client) KeyTakeaways(text string, maxTokens int) (string, error) {
	systemPrompt := "You are an assistant that extracts key takeaways from meetings. Focus on decisions made, important insights, and critical information."

	prompt := fmt.Sprintf("Extract the key takeaways from this meeting:\n\n%s\n\n---\n\nKey Takeaways:", text)

	req := CompletionRequest{
		Model:       c.model,
		Prompt:      prompt,
		System:      systemPrompt,
		MaxTokens:   maxTokens,
		Temperature: 0.3,
		TopP:        0.9,
	}

	return c.complete(context.Background(), req)
}

// SentimentAnalysis analyzes the sentiment of text
func (c *Client) SentimentAnalysis(text string, maxTokens int) (string, error) {
	systemPrompt := "You are an assistant that analyzes the sentiment and tone of meeting discussions. Note overall mood, engagement level, and any notable emotional dynamics."

	prompt := fmt.Sprintf("Analyze the sentiment and tone of this meeting:\n\n%s\n\n---\n\nSentiment Analysis:", text)

	req := CompletionRequest{
		Model:       c.model,
		Prompt:      prompt,
		System:      systemPrompt,
		MaxTokens:   maxTokens,
		Temperature: 0.3,
		TopP:        0.9,
	}

	return c.complete(context.Background(), req)
}

// GenerateQuestions generates discussion questions from text
func (c *Client) GenerateQuestions(text string, maxTokens int) (string, error) {
	systemPrompt := "You are an assistant that generates thoughtful discussion questions from meeting transcripts."

	prompt := fmt.Sprintf("Generate discussion questions from this meeting:\n\n%s\n\n---\n\nDiscussion Questions:", text)

	req := CompletionRequest{
		Model:       c.model,
		Prompt:      prompt,
		System:      systemPrompt,
		MaxTokens:   maxTokens,
		Temperature: 0.5,
		TopP:        0.9,
	}

	return c.complete(context.Background(), req)
}

// complete sends a completion request to the inference server
func (c *Client) complete(ctx context.Context, req CompletionRequest) (string, error) {
	// Add model if not set
	if req.Model == "" {
		req.Model = c.model
	}

	// Use messages format (OpenAI compatible)
	messages := []map[string]string{
		{"role": "system", "content": req.System},
		{"role": "user", "content": req.Prompt},
	}

	requestBody := map[string]interface{}{
		"model":       req.Model,
		"messages":    messages,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"top_p":       req.TopP,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	url := c.endpoint + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Log debug info without exposing potentially sensitive response body
	if os.Getenv("GRANOLA_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "Debug: Received %d byte response from %s\n", len(body), c.endpoint)
	}

	// Parse response
	var completionResp CompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(completionResp.Choices) == 0 {
		return "", fmt.Errorf("no completion choices in response")
	}

	choice := completionResp.Choices[0]

	// Try message.content first, then reasoning_content, then text
	if choice.Message.Content != "" {
		return choice.Message.Content, nil
	}

	if choice.Message.ReasoningContent != "" {
		return choice.Message.ReasoningContent, nil
	}

	if choice.Text != "" {
		return choice.Text, nil
	}

	// Log warning for empty response
	if os.Getenv("GRANOLA_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "Debug: Empty response from model\n")
	}

	return "", fmt.Errorf("model returned empty response")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
