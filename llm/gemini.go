package llm

import "context"

// geminiProvider implements Provider for Google's Gemini API using the
// OpenAI-compatible endpoint. Gemini uses a different path prefix than
// standard OpenAI providers (no /v1).
//
// Supported chat models:
//
//	gemini-2.5-flash       — fast, cost-effective
//	gemini-2.5-pro         — highest capability
//	gemini-2.0-flash       — previous gen fast
//
// Supported embedding models:
//
//	gemini-embedding-001   (3072 dim, free tier available)
//
// API key: set via config or GEMINI_API_KEY env var.
type geminiProvider struct {
	base openAICompatClient
}

// NewGemini creates a provider for Google Gemini.
func NewGemini(cfg Config) Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	}
	return &geminiProvider{base: newOpenAICompatClientPrefix(cfg, "")}
}

func (p *geminiProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return p.base.chat(ctx, req)
}

func (p *geminiProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.base.embed(ctx, texts)
}

func (p *geminiProvider) ChatWithImages(ctx context.Context, req VisionChatRequest) (*ChatResponse, error) {
	return p.base.chatWithImages(ctx, req)
}
