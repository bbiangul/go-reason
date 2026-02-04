// Command eval runs the ALTAVision evaluation suite against a GoReason engine.
//
// Usage:
//
//	go run -tags sqlite_fts5 ./cmd/eval \
//	  --pdf ./docs/ALTAVision.pdf \
//	  --chat-provider openrouter \
//	  --chat-model qwen/qwen3-30b \
//	  --embed-provider ollama \
//	  --embed-model qwen/qwen3-embedding \
//	  --difficulty easy
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bbiangul/go-reason"
	"github.com/bbiangul/go-reason/eval"
)

func main() {
	var (
		pdfPath       = flag.String("pdf", "", "Path to ALTAVision PDF (required)")
		dbPath        = flag.String("db", "", "Path to SQLite database (default: inside run directory)")
		chatProvider  = flag.String("chat-provider", "groq", "Chat LLM provider")
		chatModel     = flag.String("chat-model", "openai/gpt-oss-120b", "Chat model name")
		chatBaseURL   = flag.String("chat-base-url", "", "Chat provider base URL override")
		embedProvider = flag.String("embed-provider", "openai", "Embedding provider")
		embedModel    = flag.String("embed-model", "text-embedding-3-small", "Embedding model name")
		embedBaseURL  = flag.String("embed-base-url", "", "Embedding provider base URL (auto-detected from provider)")
		embedAPIKey   = flag.String("embed-api-key", "", "Embedding provider API key (if required)")
		embedDim      = flag.Int("embed-dim", 1536, "Embedding dimension")
		difficulty    = flag.String("difficulty", "all", "Difficulty level to run: easy, medium, hard, super-hard, all")
		outputFile    = flag.String("output", "", "Path to write JSON report (default: inside run directory)")
		openrouterKey = flag.String("openrouter-key", "", "OpenRouter API key (default: $OPENROUTER_API_KEY)")
		maxRounds     = flag.Int("max-rounds", 3, "Maximum reasoning rounds per query")
		maxResults    = flag.Int("max-results", 25, "Maximum retrieval results per query")
		graphConc     = flag.Int("graph-concurrency", 16, "Max parallel LLM calls for graph extraction")
		chunkTokens   = flag.Int("chunk-max-tokens", 1024, "Maximum tokens per chunk")
		chunkOverlap  = flag.Int("chunk-overlap", 128, "Token overlap between chunks")
		weightVec     = flag.Float64("weight-vec", 1.0, "RRF vector weight")
		weightFTS     = flag.Float64("weight-fts", 1.0, "RRF FTS weight")
		weightGraph   = flag.Float64("weight-graph", 0.5, "RRF graph weight")
		skipIngest    = flag.Bool("skip-ingest", false, "Skip ingestion and reuse existing --db (eval-only mode)")
	)
	flag.Parse()

	if *pdfPath == "" && !*skipIngest {
		log.Fatal("--pdf flag is required (or use --skip-ingest with --db)")
	}
	if *skipIngest && *dbPath == "" {
		log.Fatal("--skip-ingest requires --db pointing to an existing database")
	}

	// Resolve chat API key from flag or well-known env vars.
	apiKey := *openrouterKey
	if apiKey == "" {
		switch *chatProvider {
		case "openrouter":
			apiKey = os.Getenv("OPENROUTER_API_KEY")
		case "openai":
			apiKey = os.Getenv("OPENAI_API_KEY")
		case "groq":
			apiKey = os.Getenv("GROQ_API_KEY")
		}
	}
	if apiKey == "" && *chatProvider != "ollama" && *chatProvider != "lmstudio" {
		log.Fatalf("API key required for provider %q: set --openrouter-key or the appropriate env var", *chatProvider)
	}

	// Resolve embed API key from flag or well-known env vars.
	embedKey := *embedAPIKey
	if embedKey == "" {
		switch *embedProvider {
		case "openai":
			embedKey = os.Getenv("OPENAI_API_KEY")
		case "openrouter":
			embedKey = os.Getenv("OPENROUTER_API_KEY")
		case "groq":
			embedKey = os.Getenv("GROQ_API_KEY")
		}
	}

	// Resolve base URLs for known providers.
	chatURL := *chatBaseURL
	if chatURL == "" {
		switch *chatProvider {
		case "openrouter":
			chatURL = "https://openrouter.ai/api"
		case "openai":
			chatURL = "https://api.openai.com"
		case "groq":
			chatURL = "https://api.groq.com/openai"
		case "ollama":
			chatURL = "http://localhost:11434"
		case "lmstudio":
			chatURL = "http://localhost:1234"
		}
	}
	embedURL := *embedBaseURL
	if embedURL == "" {
		switch *embedProvider {
		case "openai":
			embedURL = "https://api.openai.com"
		case "openrouter":
			embedURL = "https://openrouter.ai/api"
		case "groq":
			embedURL = "https://api.groq.com/openai"
		case "ollama":
			embedURL = "http://localhost:11434"
		case "lmstudio":
			embedURL = "http://localhost:1234"
		}
	}

	// --- Run artifact directory ---
	runDir := createRunDir()
	fmt.Fprintf(os.Stderr, "Run directory: %s\n", runDir)

	// Setup log tee: write to both stderr and eval.log
	logFile := setupLogTee(runDir)
	defer logFile.Close()

	// Resolve DB path — use run directory by default
	db := *dbPath
	if db == "" {
		db = filepath.Join(runDir, "goreason.db")
		fmt.Fprintf(os.Stderr, "Using database: %s\n", db)
	}

	// Collect metadata
	meta := map[string]interface{}{
		"git_commit":        gitCommit(),
		"go_version":        runtime.Version(),
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
		"chat_provider":     *chatProvider,
		"chat_model":        *chatModel,
		"embed_provider":    *embedProvider,
		"embed_model":       *embedModel,
		"embed_dim":         *embedDim,
		"chunk_max_tokens":  *chunkTokens,
		"chunk_overlap":     *chunkOverlap,
		"graph_concurrency": *graphConc,
		"rrf_weights": map[string]float64{
			"vector": *weightVec,
			"fts":    *weightFTS,
			"graph":  *weightGraph,
		},
		"max_results": *maxResults,
		"max_rounds":  *maxRounds,
		"skip_ingest":  *skipIngest,
		"pdf":         func() string { if *pdfPath != "" { return filepath.Base(*pdfPath) }; return "(reused DB)" }(),
		"difficulty":  *difficulty,
	}
	writeJSON(filepath.Join(runDir, "metadata.json"), meta)

	cfg := goreason.Config{
		DBPath: db,
		Chat: goreason.LLMConfig{
			Provider: *chatProvider,
			Model:    *chatModel,
			BaseURL:  chatURL,
			APIKey:   apiKey,
		},
		Embedding: goreason.LLMConfig{
			Provider: *embedProvider,
			Model:    *embedModel,
			BaseURL:  embedURL,
			APIKey:   embedKey,
		},
		EmbeddingDim:        *embedDim,
		MaxRounds:           *maxRounds,
		ConfidenceThreshold: 0.5,
		WeightVector:        *weightVec,
		WeightFTS:           *weightFTS,
		WeightGraph:         *weightGraph,
		MaxChunkTokens:      *chunkTokens,
		ChunkOverlap:        *chunkOverlap,
		GraphConcurrency:    *graphConc,
	}

	ctx := context.Background()
	totalStart := time.Now()

	fmt.Fprintf(os.Stderr, "Creating engine...\n")
	engine, err := goreason.New(cfg)
	if err != nil {
		log.Fatalf("creating engine: %v", err)
	}
	defer engine.Close()

	var ingestElapsed time.Duration
	if *skipIngest {
		fmt.Fprintf(os.Stderr, "Skipping ingestion (reusing DB: %s)\n", db)
	} else {
		fmt.Fprintf(os.Stderr, "Ingesting PDF: %s\n", *pdfPath)
		ingestStart := time.Now()
		docID, err := engine.Ingest(ctx, *pdfPath)
		if err != nil {
			log.Fatalf("ingesting PDF: %v", err)
		}
		ingestElapsed = time.Since(ingestStart)
		fmt.Fprintf(os.Stderr, "Ingested document ID %d in %s\n", docID, ingestElapsed.Round(time.Millisecond))
	}

	// Select datasets
	datasets := selectDatasets(*difficulty)
	if len(datasets) == 0 {
		log.Fatalf("unknown difficulty: %s (use: easy, medium, hard, super-hard, all)", *difficulty)
	}

	evaluator := eval.NewEvaluator(engine)
	queryOpts := []goreason.QueryOption{
		goreason.WithMaxResults(*maxResults),
		goreason.WithMaxRounds(*maxRounds),
	}

	var allReports []*eval.Report
	evalStart := time.Now()

	for _, ds := range datasets {
		fmt.Fprintf(os.Stderr, "\nRunning %s (%d tests)...\n", ds.Name, len(ds.Tests))
		report, err := evaluator.Run(ctx, ds, queryOpts...)
		if err != nil {
			log.Fatalf("running %s: %v", ds.Name, err)
		}
		allReports = append(allReports, report)

		fmt.Println(eval.FormatReport(report))
		fmt.Println()
	}

	evalElapsed := time.Since(evalStart)
	totalElapsed := time.Since(totalStart)

	// Update metadata with timing
	meta["ingestion_elapsed"] = ingestElapsed.Round(time.Millisecond).String()
	meta["eval_elapsed"] = evalElapsed.Round(time.Millisecond).String()
	meta["total_elapsed"] = totalElapsed.Round(time.Millisecond).String()
	writeJSON(filepath.Join(runDir, "metadata.json"), meta)

	// Write eval-report.json in run directory
	reportPath := filepath.Join(runDir, "eval-report.json")
	writeJSON(reportPath, allReports)
	fmt.Fprintf(os.Stderr, "Eval report written to: %s\n", reportPath)

	// Write to --output if specified (backward compat)
	if *outputFile != "" {
		writeJSON(*outputFile, allReports)
		fmt.Fprintf(os.Stderr, "JSON report also written to: %s\n", *outputFile)
	}

	// Print summary
	fmt.Println("=== Summary ===")
	totalPassed, totalTests := 0, 0
	for _, r := range allReports {
		totalPassed += r.Passed
		totalTests += r.TotalTests
		rate := 0.0
		if r.TotalTests > 0 {
			rate = float64(r.Passed) / float64(r.TotalTests) * 100
		}
		fmt.Printf("  %-45s %d/%d (%.1f%%)\n", r.Dataset, r.Passed, r.TotalTests, rate)
	}
	if totalTests > 0 {
		fmt.Printf("  %-45s %d/%d (%.1f%%)\n", "TOTAL", totalPassed, totalTests,
			float64(totalPassed)/float64(totalTests)*100)
	}

	fmt.Fprintf(os.Stderr, "\nRun directory: %s\n", runDir)
}

func selectDatasets(difficulty string) []eval.Dataset {
	all := eval.ALTAVisionAllDatasets()

	switch strings.ToLower(difficulty) {
	case "all":
		return []eval.Dataset{
			all[eval.DifficultyEasy],
			all[eval.DifficultyMedium],
			all[eval.DifficultyHard],
			all[eval.DifficultySuperHard],
		}
	case "easy":
		return []eval.Dataset{all[eval.DifficultyEasy]}
	case "medium":
		return []eval.Dataset{all[eval.DifficultyMedium]}
	case "hard":
		return []eval.Dataset{all[eval.DifficultyHard]}
	case "super-hard":
		return []eval.Dataset{all[eval.DifficultySuperHard]}
	default:
		return nil
	}
}

// createRunDir creates evals/runs/<timestamp>/ and returns its path.
func createRunDir() string {
	ts := time.Now().Format("2006-01-02_15-04-05")
	dir := filepath.Join("evals", "runs", ts)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("creating run directory: %v", err)
	}
	return dir
}

// setupLogTee configures slog to write to both stderr and eval.log in the run dir.
func setupLogTee(runDir string) *os.File {
	logPath := filepath.Join(runDir, "eval.log")
	f, err := os.Create(logPath)
	if err != nil {
		log.Fatalf("creating log file: %v", err)
	}
	w := io.MultiWriter(os.Stderr, f)
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))
	return f
}

// gitCommit returns the current git HEAD short hash, or "unknown".
func gitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// writeJSON marshals v to indented JSON and writes it to path.
func writeJSON(path string, v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("marshaling JSON for %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Fatalf("writing %s: %v", path, err)
	}
}
