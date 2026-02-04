package llm

import "context"

// openAIProvider implements Provider for the OpenAI API.
// Uses the standard OpenAI-compatible format for both chat and embeddings.
//
// Supported embedding models:
//
//	text-embedding-3-small  (1536 dim, $0.02/M tokens)  — default
//	text-embedding-3-large  (3072 dim, $0.13/M tokens)
//	text-embedding-ada-002  (1536 dim, $0.10/M tokens)
//
// API key: set via config, OPENAI_API_KEY env var, or the server's
// GOREASON_EMBED_API_KEY env var.
type openAIProvider struct {
	base openAICompatClient
}

// NewOpenAI creates a provider for OpenAI.
func NewOpenAI(cfg Config) Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com"
	}
	if cfg.Model == "" {
		cfg.Model = "text-embedding-3-small"
	}
	return &openAIProvider{base: newOpenAICompatClient(cfg)}
}

func (p *openAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return p.base.chat(ctx, req)
}

func (p *openAIProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.base.embed(ctx, texts)
}
