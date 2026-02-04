package llm

import "context"

// groqProvider implements Provider for Groq's inference API.
// Groq uses the OpenAI-compatible API format and provides extremely
// fast inference for open-source models (Llama, Mixtral, Gemma, etc.).
//
// API key: set via config, GROQ_API_KEY env var, or the server's
// GOREASON_CHAT_API_KEY env var.
type groqProvider struct {
	base openAICompatClient
}

// NewGroq creates a provider for Groq.
func NewGroq(cfg Config) Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.groq.com/openai"
	}
	if cfg.Model == "" {
		cfg.Model = "llama-3.3-70b-versatile"
	}
	return &groqProvider{base: newOpenAICompatClient(cfg)}
}

func (p *groqProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return p.base.chat(ctx, req)
}

func (p *groqProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.base.embed(ctx, texts)
}
