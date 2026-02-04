package llm

import "context"

// openRouterProvider implements Provider for OpenRouter.
// OpenRouter uses the OpenAI-compatible API format.
type openRouterProvider struct {
	base openAICompatClient
}

// NewOpenRouter creates a provider for OpenRouter.
func NewOpenRouter(cfg Config) Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://openrouter.ai/api"
	}
	return &openRouterProvider{base: newOpenAICompatClient(cfg)}
}

func (p *openRouterProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return p.base.chat(ctx, req)
}

func (p *openRouterProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.base.embed(ctx, texts)
}
