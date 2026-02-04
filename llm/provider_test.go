package llm

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		provider string
		wantType string
	}{
		{"ollama", "*llm.ollamaProvider"},
		{"lmstudio", "*llm.lmStudioProvider"},
		{"openrouter", "*llm.openRouterProvider"},
		{"xai", "*llm.xaiProvider"},
		{"custom", "*llm.openAICompatProvider"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			cfg := Config{
				Provider: tt.provider,
				Model:    "test-model",
			}
			p, err := NewProvider(cfg)
			if err != nil {
				t.Fatalf("NewProvider(%q) returned error: %v", tt.provider, err)
			}
			gotType := fmt.Sprintf("%T", p)
			if gotType != tt.wantType {
				t.Errorf("NewProvider(%q) type = %s, want %s", tt.provider, gotType, tt.wantType)
			}
		})
	}
}

func TestNewProviderUnknown(t *testing.T) {
	cfg := Config{
		Provider: "doesnotexist",
		Model:    "test-model",
	}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	want := "unknown llm provider: doesnotexist"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestNewProviderEmpty(t *testing.T) {
	cfg := Config{
		Provider: "",
		Model:    "test-model",
	}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected error for empty provider, got nil")
	}
	want := "llm provider not specified"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// TestDefaultBaseURLs verifies that when BaseURL is empty in the config,
// each provider constructor sets the correct default.
func TestDefaultBaseURLs(t *testing.T) {
	tests := []struct {
		provider   string
		wantURL    string
		fieldPath  string // path to base.cfg.BaseURL inside the provider struct
	}{
		{"ollama", "http://localhost:11434", "base.cfg.BaseURL"},
		{"lmstudio", "http://localhost:1234", "base.cfg.BaseURL"},
		{"openrouter", "https://openrouter.ai/api", "base.cfg.BaseURL"},
		{"xai", "https://api.x.ai", "base.cfg.BaseURL"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			cfg := Config{
				Provider: tt.provider,
				Model:    "test-model",
				// BaseURL intentionally left empty.
			}
			p, err := NewProvider(cfg)
			if err != nil {
				t.Fatalf("NewProvider(%q): %v", tt.provider, err)
			}

			// Use reflection to reach base.cfg.BaseURL on the concrete type.
			v := reflect.ValueOf(p).Elem()
			base := v.FieldByName("base")
			cfgField := base.FieldByName("cfg")
			gotURL := cfgField.FieldByName("BaseURL").String()

			if gotURL != tt.wantURL {
				t.Errorf("default BaseURL for %q = %q, want %q", tt.provider, gotURL, tt.wantURL)
			}
		})
	}
}

// TestCustomProviderNoDefaultURL confirms the custom provider does not
// override an empty BaseURL with a default.
func TestCustomProviderNoDefaultURL(t *testing.T) {
	cfg := Config{
		Provider: "custom",
		Model:    "test-model",
		BaseURL:  "",
	}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider(custom): %v", err)
	}

	v := reflect.ValueOf(p).Elem()
	base := v.FieldByName("base")
	cfgField := base.FieldByName("cfg")
	gotURL := cfgField.FieldByName("BaseURL").String()

	if gotURL != "" {
		t.Errorf("custom provider BaseURL = %q, want empty", gotURL)
	}
}

// TestExplicitBaseURLPreserved verifies that a user-supplied BaseURL
// is not overwritten by the default.
func TestExplicitBaseURLPreserved(t *testing.T) {
	customURL := "http://my-server:9999"

	tests := []string{"ollama", "lmstudio", "openrouter", "xai", "custom"}
	for _, provider := range tests {
		t.Run(provider, func(t *testing.T) {
			cfg := Config{
				Provider: provider,
				Model:    "test-model",
				BaseURL:  customURL,
			}
			p, err := NewProvider(cfg)
			if err != nil {
				t.Fatalf("NewProvider(%q): %v", provider, err)
			}

			v := reflect.ValueOf(p).Elem()
			base := v.FieldByName("base")
			cfgField := base.FieldByName("cfg")
			gotURL := cfgField.FieldByName("BaseURL").String()

			if gotURL != customURL {
				t.Errorf("provider %q BaseURL = %q, want %q", provider, gotURL, customURL)
			}
		})
	}
}

// TestProviderImplementsInterface confirms that every provider
// returned by NewProvider satisfies the Provider interface.
func TestProviderImplementsInterface(t *testing.T) {
	providers := []string{"ollama", "lmstudio", "openrouter", "xai", "custom"}

	for _, name := range providers {
		t.Run(name, func(t *testing.T) {
			cfg := Config{Provider: name, Model: "m"}
			p, err := NewProvider(cfg)
			if err != nil {
				t.Fatalf("NewProvider(%q): %v", name, err)
			}

			// Compile-time check is implicit because NewProvider returns Provider,
			// but verify the value is non-nil and usable.
			var _ Provider = p
			if p == nil {
				t.Fatal("provider is nil")
			}
		})
	}
}

// TestModelPassedThrough verifies the model from Config is stored
// inside the provider.
func TestModelPassedThrough(t *testing.T) {
	cfg := Config{
		Provider: "ollama",
		Model:    "llama3:latest",
	}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	v := reflect.ValueOf(p).Elem()
	base := v.FieldByName("base")
	cfgField := base.FieldByName("cfg")
	gotModel := cfgField.FieldByName("Model").String()

	if gotModel != "llama3:latest" {
		t.Errorf("model = %q, want %q", gotModel, "llama3:latest")
	}
}

// TestAPIKeyPassedThrough verifies the API key from Config is stored
// inside the provider.
func TestAPIKeyPassedThrough(t *testing.T) {
	cfg := Config{
		Provider: "openrouter",
		Model:    "test",
		APIKey:   "sk-test-key-123",
	}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	v := reflect.ValueOf(p).Elem()
	base := v.FieldByName("base")
	cfgField := base.FieldByName("cfg")
	gotKey := cfgField.FieldByName("APIKey").String()

	if gotKey != "sk-test-key-123" {
		t.Errorf("api key = %q, want %q", gotKey, "sk-test-key-123")
	}
}
