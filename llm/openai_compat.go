package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// openAICompatClient is the shared base for all OpenAI-compatible providers.
type openAICompatClient struct {
	cfg        Config
	client     *http.Client
	pathPrefix string // API path prefix, defaults to "/v1"
}

func newOpenAICompatClient(cfg Config) openAICompatClient {
	return newOpenAICompatClientPrefix(cfg, "/v1")
}

func newOpenAICompatClientPrefix(cfg Config, prefix string) openAICompatClient {
	// Timeout for individual HTTP requests. Kept generous for local providers
	// (Ollama, LM Studio) which may load models on first request, but
	// reasonable enough to avoid multi-minute hangs on stalled connections.
	timeout := 120 * time.Second
	return openAICompatClient{
		cfg:        cfg,
		pathPrefix: prefix,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// NewOpenAICompat creates a generic OpenAI-compatible provider.
func NewOpenAICompat(cfg Config) Provider {
	return &openAICompatProvider{base: newOpenAICompatClient(cfg)}
}

type openAICompatProvider struct {
	base openAICompatClient
}

func (p *openAICompatProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return p.base.chat(ctx, req)
}

func (p *openAICompatProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.base.embed(ctx, texts)
}

func (p *openAICompatProvider) ChatWithImages(ctx context.Context, req VisionChatRequest) (*ChatResponse, error) {
	return p.base.chatWithImages(ctx, req)
}

// --- shared implementation ---

type chatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       json.RawMessage `json:"messages"`
	Temperature    float64         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *responseFormat  `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func (c *openAICompatClient) chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	msgs, err := json.Marshal(req.Messages)
	if err != nil {
		return nil, err
	}

	model := req.Model
	if model == "" {
		model = c.cfg.Model
	}

	body := chatCompletionRequest{
		Model:       model,
		Messages:    msgs,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	if req.ResponseFormat == "json_object" {
		body.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	respBody, err := c.doPost(ctx, c.pathPrefix+"/chat/completions", body)
	if err != nil {
		return nil, err
	}

	var resp chatCompletionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("decoding chat response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		Content:          resp.Choices[0].Message.Content,
		Model:            resp.Model,
		FinishReason:     resp.Choices[0].FinishReason,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}, nil
}

func (c *openAICompatClient) embed(ctx context.Context, texts []string) ([][]float32, error) {
	body := embeddingRequest{
		Model: c.cfg.Model,
		Input: texts,
	}

	respBody, err := c.doPost(ctx, c.pathPrefix+"/embeddings", body)
	if err != nil {
		return nil, err
	}

	var resp embeddingResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("decoding embedding response: %w", err)
	}

	// Sort by index to ensure correct ordering
	embeddings := make([][]float32, len(texts))
	for _, d := range resp.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}
	return embeddings, nil
}

func (c *openAICompatClient) chatWithImages(ctx context.Context, req VisionChatRequest) (*ChatResponse, error) {
	msgs, err := json.Marshal(req.Messages)
	if err != nil {
		return nil, err
	}

	model := req.Model
	if model == "" {
		model = c.cfg.Model
	}

	body := chatCompletionRequest{
		Model:       model,
		Messages:    msgs,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	respBody, err := c.doPost(ctx, c.pathPrefix+"/chat/completions", body)
	if err != nil {
		return nil, err
	}

	var resp chatCompletionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("decoding vision response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		Content:          resp.Choices[0].Message.Content,
		Model:            resp.Model,
		FinishReason:     resp.Choices[0].FinishReason,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}, nil
}

const (
	maxRetries         = 6
	baseRetryDelay     = 2 * time.Second
	minRateLimitDelay  = 5 * time.Second // minimum delay for 429 errors
)

// retryableStatusCode returns true for HTTP status codes that warrant a retry.
func retryableStatusCode(code int) bool {
	return code == http.StatusTooManyRequests ||
		code == http.StatusBadGateway ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout
}

func (c *openAICompatClient) doPost(ctx context.Context, path string, body interface{}) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := c.cfg.BaseURL + path

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseRetryDelay * time.Duration(1<<(attempt-1)) // 1s, 2s, 4s
			slog.Warn("llm: retrying request",
				"url", url,
				"attempt", attempt,
				"delay", delay,
				"error", lastErr,
			)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		if c.cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			// Retry on network/timeout errors (not context cancellation).
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("request to %s failed: %w", url, err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading response body: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return respBody, nil
		}

		lastErr = fmt.Errorf("LLM API error %d: %s", resp.StatusCode, string(respBody))

		if !retryableStatusCode(resp.StatusCode) {
			return nil, lastErr
		}

		// Handle 429 rate limiting with longer delays.
		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitDelay := minRateLimitDelay * time.Duration(1<<attempt) // 5s, 10s, 20s, 40s...
			// Respect Retry-After header if provided.
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if seconds, err := strconv.Atoi(ra); err == nil && seconds > 0 {
					headerDelay := time.Duration(seconds) * time.Second
					if headerDelay > rateLimitDelay {
						rateLimitDelay = headerDelay
					}
				}
			}
			slog.Warn("llm: rate limited, waiting before retry",
				"url", url,
				"attempt", attempt+1,
				"delay", rateLimitDelay,
			)
			select {
			case <-time.After(rateLimitDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
