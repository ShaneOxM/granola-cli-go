package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	endpoint   string
	model      string
	timeout    time.Duration
	httpClient *http.Client
}

// EmbeddingRequest represents an Ollama embedding request
type EmbeddingRequest struct {
	Model   string  `json:"model"`
	Prompt  string  `json:"prompt"`
	Options Options `json:"options,omitempty"`
}

// Options represents model options
type Options struct {
	NumKeep int `json:"num_keep,omitempty"`
}

// EmbeddingResponse represents an Ollama embedding response
type EmbeddingResponse struct {
	Embedding       []float32 `json:"embedding"`
	TotalDuration   int64     `json:"total_duration"`
	LoadDuration    int64     `json:"load_duration"`
	PromptEvalCount int       `json:"prompt_eval_count"`
}

// NewOllamaProvider creates a new Ollama embedding provider.
//
// Environment variables:
//   - OLLAMA_EMBED_ENDPOINT: Ollama API endpoint (default: http://localhost:11434)
//   - OLLAMA_EMBED_MODEL: Model name (default: nomic-embed-text)
//   - GRANOLA_EMBEDDING_TIMEOUT: Request timeout (default: 300s)
//
// The provider uses nomic-embed-text model which produces 768-dimensional vectors.
func NewOllamaProvider() *OllamaProvider {
	endpoint := os.Getenv("OLLAMA_EMBED_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	model := os.Getenv("OLLAMA_EMBED_MODEL")
	if model == "" {
		model = "nomic-embed-text"
	}

	timeout := 300 * time.Second
	if timeoutStr := os.Getenv("GRANOLA_EMBEDDING_TIMEOUT"); timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	return &OllamaProvider{
		endpoint: endpoint,
		model:    model,
		timeout:  timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Name returns the provider name
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Model returns the model name
func (p *OllamaProvider) Model() string {
	return p.model
}

// Dimensions returns the embedding vector dimensions
func (p *OllamaProvider) Dimensions() int {
	return 768 // nomic-embed-text uses 768 dimensions
}

// MaxChars returns the maximum text length supported
func (p *OllamaProvider) MaxChars() int {
	if v := os.Getenv("OLLAMA_EMBED_MAX_CHARS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 2000
}

// IsAvailable checks if Ollama is accessible
func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
	url := p.endpoint + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GenerateEmbedding creates a single embedding vector
func (p *OllamaProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if len(text) > p.MaxChars() {
		text = text[:p.MaxChars()]
	}

	reqBody := EmbeddingRequest{
		Model:  p.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.endpoint + "/api/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embeddingResp.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return embeddingResp.Embedding, nil
}

// GenerateBatchEmbeddings creates multiple embeddings efficiently
func (p *OllamaProvider) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	batch := make([][]float32, len(texts))

	for i, text := range texts {
		embedding, err := p.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		batch[i] = embedding
	}

	return batch, nil
}
