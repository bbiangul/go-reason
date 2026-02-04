//go:build eval && cgo

package eval

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bbiangul/go-reason"
)

// difficultyOrder defines a fixed iteration order for evaluation runs,
// avoiding non-deterministic map iteration.
var difficultyOrder = []string{
	DifficultyEasy,
	DifficultyMedium,
	DifficultyHard,
	DifficultySuperHard,
}

// TestALTAVisionEvaluation runs the full ALTAVision evaluation suite.
// Requires:
//   - Build tags: eval,cgo
//   - Environment variable: OPENROUTER_API_KEY (for chat inference)
//   - Environment variable: ALTAVISION_PDF_PATH (path to the ALTAVision PDF)
//   - Running Ollama instance for embeddings
//
// Run with:
//
//	CGO_ENABLED=1 go test -v -tags "eval sqlite_fts5" -timeout 60m \
//	  -run TestALTAVisionEvaluation ./eval/
func TestALTAVisionEvaluation(t *testing.T) {
	pdfPath := os.Getenv("ALTAVISION_PDF_PATH")
	if pdfPath == "" {
		t.Skip("ALTAVISION_PDF_PATH not set; skipping integration test")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY not set; skipping integration test")
	}

	chatProvider := envOr("EVAL_CHAT_PROVIDER", "openrouter")
	chatModel := envOr("EVAL_CHAT_MODEL", "qwen/qwen3-30b-a3b")
	chatBaseURL := envOr("EVAL_CHAT_BASE_URL", "https://openrouter.ai/api")
	embedProvider := envOr("EVAL_EMBED_PROVIDER", "ollama")
	embedModel := envOr("EVAL_EMBED_MODEL", "nomic-embed-text")
	embedBaseURL := envOr("EVAL_EMBED_BASE_URL", "http://localhost:11434")
	embedDim := 768

	// Create a temp database for the test
	tmpDB, err := os.CreateTemp("", "goreason-altavision-test-*.db")
	if err != nil {
		t.Fatalf("creating temp db: %v", err)
	}
	tmpDB.Close()
	defer os.Remove(tmpDB.Name())

	cfg := goreason.Config{
		DBPath: tmpDB.Name(),
		Chat: goreason.LLMConfig{
			Provider: chatProvider,
			Model:    chatModel,
			BaseURL:  chatBaseURL,
			APIKey:   apiKey,
		},
		Embedding: goreason.LLMConfig{
			Provider: embedProvider,
			Model:    embedModel,
			BaseURL:  embedBaseURL,
		},
		EmbeddingDim:        embedDim,
		MaxRounds:           3,
		ConfidenceThreshold: 0.5,
		WeightVector:        1.0,
		WeightFTS:           1.0,
		WeightGraph:         0.5,
		MaxChunkTokens:      512,
		ChunkOverlap:        64,
	}

	// Global context for ingestion (generous timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	// Create engine
	engine, err := goreason.New(cfg)
	if err != nil {
		t.Fatalf("creating engine: %v", err)
	}
	defer engine.Close()

	// Ingest the ALTAVision PDF
	t.Logf("Ingesting PDF: %s", pdfPath)
	ingestStart := time.Now()
	docID, err := engine.Ingest(ctx, pdfPath)
	if err != nil {
		t.Fatalf("ingesting PDF: %v", err)
	}
	t.Logf("Ingested document ID %d in %s", docID, time.Since(ingestStart).Round(time.Millisecond))

	evaluator := NewEvaluator(engine)
	queryOpts := []goreason.QueryOption{
		goreason.WithMaxResults(20),
		goreason.WithMaxRounds(3),
	}

	// Expected minimum pass rates per difficulty
	thresholds := map[string]float64{
		DifficultyEasy:      0.70,
		DifficultyMedium:    0.50,
		DifficultyHard:      0.30,
		DifficultySuperHard: 0.20,
	}

	datasets := ALTAVisionAllDatasets()

	// Use fixed order so results are deterministic and earlier (easier)
	// difficulty levels always run before harder ones.
	for _, difficulty := range difficultyOrder {
		ds := datasets[difficulty]
		t.Run(difficulty, func(t *testing.T) {
			// Per-difficulty timeout: 12 minutes each (30 tests * ~20s/query)
			diffCtx, diffCancel := context.WithTimeout(ctx, 12*time.Minute)
			defer diffCancel()

			t.Logf("Running %s (%d tests)...", ds.Name, len(ds.Tests))
			report, err := evaluator.Run(diffCtx, ds, queryOpts...)
			if err != nil {
				t.Fatalf("running evaluation: %v", err)
			}

			t.Log(FormatReport(report))

			// Check pass rate
			pr := 0.0
			if report.TotalTests > 0 {
				pr = float64(report.Passed) / float64(report.TotalTests)
			}

			threshold := thresholds[difficulty]
			if pr < threshold {
				t.Errorf("%s pass rate %.1f%% below threshold %.1f%%",
					difficulty, pr*100, threshold*100)
			} else {
				t.Logf("%s pass rate: %.1f%% (threshold: %.1f%%)",
					difficulty, pr*100, threshold*100)
			}
		})
	}
}

// TestALTAVisionDatasetStructure validates the dataset structure without running queries.
func TestALTAVisionDatasetStructure(t *testing.T) {
	datasets := ALTAVisionAllDatasets()

	expectedCounts := map[string]int{
		DifficultyEasy:      30,
		DifficultyMedium:    30,
		DifficultyHard:      30,
		DifficultySuperHard: 50,
	}

	for _, difficulty := range difficultyOrder {
		ds := datasets[difficulty]
		t.Run(difficulty, func(t *testing.T) {
			expected := expectedCounts[difficulty]
			if len(ds.Tests) != expected {
				t.Errorf("expected %d tests for %s, got %d", expected, difficulty, len(ds.Tests))
			}

			if ds.Name == "" {
				t.Error("dataset name is empty")
			}

			if ds.Difficulty != difficulty {
				t.Errorf("difficulty mismatch: got %q, want %q", ds.Difficulty, difficulty)
			}

			for i, tc := range ds.Tests {
				if tc.Question == "" {
					t.Errorf("test %d has empty question", i)
				}
				if len(tc.ExpectedFacts) == 0 {
					t.Errorf("test %d (%s) has no expected facts", i, tc.Question)
				}
				if tc.Category == "" {
					t.Errorf("test %d (%s) has empty category", i, tc.Question)
				}
			}
		})
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
