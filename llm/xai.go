package llm

import "context"

// xaiProvider implements Provider for xAI (Grok).
// xAI uses the OpenAI-compatible API format.
type xaiProvider struct {
	base openAICompatClient
}

// NewXAI creates a provider for xAI (Grok).
func NewXAI(cfg Config) Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.x.ai"
	}
	return &xaiProvider{base: newOpenAICompatClient(cfg)}
}

func (p *xaiProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return p.base.chat(ctx, req)
}

func (p *xaiProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.base.embed(ctx, texts)
}
