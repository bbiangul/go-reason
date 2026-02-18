package goreason

import (
	"os"
	"path/filepath"
)

// Config holds all configuration for the GoReason engine.
type Config struct {
	// DBPath is the full path to the SQLite database file.
	// If empty, defaults to ~/.goreason/<DBName>.db
	DBPath string `json:"db_path" yaml:"db_path"`

	// DBName is the name for the database (used when DBPath is empty).
	// Defaults to "goreason". The file will be <DBName>.db inside the
	// storage directory (~/.goreason/ or working dir).
	DBName string `json:"db_name" yaml:"db_name"`

	// StorageDir controls where the database is created when DBPath
	// is not explicitly set. Options: "home" (default) uses ~/.goreason/,
	// "local" uses the current working directory.
	StorageDir string `json:"storage_dir" yaml:"storage_dir"`

	// LLM providers
	Chat        LLMConfig `json:"chat" yaml:"chat"`
	Embedding   LLMConfig `json:"embedding" yaml:"embedding"`
	Vision      LLMConfig `json:"vision" yaml:"vision"`
	Translation LLMConfig `json:"translation" yaml:"translation"` // optional: fast model for query translation (defaults to Chat)

	// Retrieval weights for RRF
	WeightVector float64 `json:"weight_vector" yaml:"weight_vector"`
	WeightFTS    float64 `json:"weight_fts" yaml:"weight_fts"`
	WeightGraph  float64 `json:"weight_graph" yaml:"weight_graph"`

	// Chunking
	MaxChunkTokens int `json:"max_chunk_tokens" yaml:"max_chunk_tokens"`
	ChunkOverlap   int `json:"chunk_overlap" yaml:"chunk_overlap"`

	// Graph building
	SkipGraph        bool `json:"skip_graph" yaml:"skip_graph"`                 // Skip knowledge graph extraction during ingest
	GraphConcurrency int  `json:"graph_concurrency" yaml:"graph_concurrency"`   // Max parallel LLM calls for graph extraction (default 16)

	// Reasoning
	MaxRounds           int     `json:"max_rounds" yaml:"max_rounds"`
	ConfidenceThreshold float64 `json:"confidence_threshold" yaml:"confidence_threshold"`

	// Image captioning
	CaptionImages bool `json:"caption_images" yaml:"caption_images"` // Opt-in: caption extracted images via vision LLM

	// External parsing
	LlamaParse *LlamaParseConfig `json:"llamaparse,omitempty" yaml:"llamaparse,omitempty"`

	// Embedding dimensions (must match model)
	EmbeddingDim int `json:"embedding_dim" yaml:"embedding_dim"`
}

// LLMConfig configures a single LLM provider endpoint.
type LLMConfig struct {
	Provider string `json:"provider" yaml:"provider"` // ollama, lmstudio, openrouter, xai, gemini, custom
	Model    string `json:"model" yaml:"model"`
	BaseURL  string `json:"base_url" yaml:"base_url"`
	APIKey   string `json:"api_key" yaml:"api_key"`
}

// LlamaParseConfig configures the LlamaParse external parsing service.
type LlamaParseConfig struct {
	APIKey  string `json:"api_key" yaml:"api_key"`
	BaseURL string `json:"base_url" yaml:"base_url"`
}

// DefaultConfig returns a Config with sensible defaults for local inference.
// Database is stored in ~/.goreason/goreason.db by default.
func DefaultConfig() Config {
	return Config{
		DBName:     "goreason",
		StorageDir: "home",
		Chat: LLMConfig{
			Provider: "ollama",
			Model:    "llama3.1:8b",
			BaseURL:  "http://localhost:11434",
		},
		Embedding: LLMConfig{
			Provider: "ollama",
			Model:    "nomic-embed-text",
			BaseURL:  "http://localhost:11434",
		},
		Vision: LLMConfig{
			Provider: "ollama",
			Model:    "llama3.2-vision",
			BaseURL:  "http://localhost:11434",
		},
		WeightVector:        1.0,
		WeightFTS:           1.0,
		WeightGraph:         0.5,
		MaxChunkTokens:      1024,
		ChunkOverlap:        128,
		MaxRounds:           3,
		ConfidenceThreshold: 0.7,
		EmbeddingDim:        768,
	}
}

// resolveDBPath computes the final database path from config fields.
func (c *Config) resolveDBPath() string {
	if c.DBPath != "" {
		return c.DBPath
	}

	name := c.DBName
	if name == "" {
		name = "goreason"
	}

	switch c.StorageDir {
	case "local", "cwd":
		return name + ".db"
	default: // "home" or empty
		home, err := os.UserHomeDir()
		if err != nil {
			return name + ".db" // fallback to cwd
		}
		dir := filepath.Join(home, ".goreason")
		return filepath.Join(dir, name+".db")
	}
}
