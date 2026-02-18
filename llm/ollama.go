package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ollamaProvider implements Provider for Ollama's native API.
// Ollama also supports the OpenAI-compatible API, but the native API
// provides better control over embedding generation.
type ollamaProvider struct {
	base openAICompatClient
}

// NewOllama creates a provider for Ollama.
func NewOllama(cfg Config) Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	return &ollamaProvider{base: newOpenAICompatClient(cfg)}
}

func (p *ollamaProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Use OpenAI-compatible endpoint
	return p.base.chat(ctx, req)
}

func (p *ollamaProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// Use Ollama's native /api/embed endpoint for batched embeddings
	body := ollamaEmbedRequest{
		Model: p.base.cfg.Model,
		Input: texts,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := p.base.cfg.BaseURL + "/api/embed"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.base.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama embed request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading ollama response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed error %d: %s", resp.StatusCode, string(respBody))
	}

	var embedResp ollamaEmbedResponse
	if err := json.Unmarshal(respBody, &embedResp); err != nil {
		return nil, fmt.Errorf("decoding ollama embed response: %w", err)
	}

	result := make([][]float32, len(embedResp.Embeddings))
	for i, emb := range embedResp.Embeddings {
		result[i] = float64sToFloat32s(emb)
	}
	return result, nil
}

func (p *ollamaProvider) ChatWithImages(ctx context.Context, req VisionChatRequest) (*ChatResponse, error) {
	return p.base.chatWithImages(ctx, req)
}

type ollamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func float64sToFloat32s(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32
}
