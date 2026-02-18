package llm

import "context"

// lmStudioProvider implements Provider for LM Studio.
// LM Studio exposes an OpenAI-compatible API.
type lmStudioProvider struct {
	base openAICompatClient
}

// NewLMStudio creates a provider for LM Studio.
func NewLMStudio(cfg Config) Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:1234"
	}
	return &lmStudioProvider{base: newOpenAICompatClient(cfg)}
}

func (p *lmStudioProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return p.base.chat(ctx, req)
}

func (p *lmStudioProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.base.embed(ctx, texts)
}

func (p *lmStudioProvider) ChatWithImages(ctx context.Context, req VisionChatRequest) (*ChatResponse, error) {
	return p.base.chatWithImages(ctx, req)
}
