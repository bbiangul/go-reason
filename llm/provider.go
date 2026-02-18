package llm

import (
	"context"
	"fmt"
)

// Provider is the interface for LLM interactions.
type Provider interface {
	// Chat sends a chat completion request.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// Embed generates embeddings for a batch of texts.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// VisionProvider extends Provider with image understanding.
type VisionProvider interface {
	Provider
	// ChatWithImages sends a chat request that includes images.
	ChatWithImages(ctx context.Context, req VisionChatRequest) (*ChatResponse, error)
}

// ChatRequest is a chat completion request.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	// ResponseFormat can be set to "json_object" for JSON mode.
	ResponseFormat string `json:"response_format,omitempty"`
}

// VisionChatRequest is a chat request with image content.
type VisionChatRequest struct {
	Model       string          `json:"model"`
	Messages    []VisionMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// VisionMessage represents a chat message that may contain images.
type VisionMessage struct {
	Role    string          `json:"role"`
	Content []ContentPart   `json:"content"`
}

// ContentPart is either text or an image in a vision message.
type ContentPart struct {
	Type     string    `json:"type"` // "text" or "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL contains a base64 or URL reference to an image.
type ImageURL struct {
	URL string `json:"url"`
}

// ChatResponse is the response from a chat completion.
type ChatResponse struct {
	Content          string `json:"content"`
	Model            string `json:"model"`
	FinishReason     string `json:"finish_reason"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
}

// Config configures an LLM provider.
type Config struct {
	Provider string `json:"provider"` // ollama, lmstudio, openrouter, openai, groq, xai, gemini, custom
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
}

// NewProvider creates an LLM provider from configuration.
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case "ollama":
		return NewOllama(cfg), nil
	case "lmstudio":
		return NewLMStudio(cfg), nil
	case "openrouter":
		return NewOpenRouter(cfg), nil
	case "openai":
		return NewOpenAI(cfg), nil
	case "groq":
		return NewGroq(cfg), nil
	case "xai":
		return NewXAI(cfg), nil
	case "gemini":
		return NewGemini(cfg), nil
	case "custom":
		return NewOpenAICompat(cfg), nil
	case "":
		return nil, fmt.Errorf("llm provider not specified")
	default:
		return nil, fmt.Errorf("unknown llm provider: %s", cfg.Provider)
	}
}
