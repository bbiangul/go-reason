// Command eval runs evaluation suites against a GoReason engine.
//
// ALTAVision usage:
//
//	go run -tags sqlite_fts5 ./cmd/eval \
//	  --pdf ./docs/ALTAVision.pdf \
//	  --chat-provider groq \
//	  --chat-model openai/gpt-oss-120b \
//	  --difficulty easy
//
// LegalBench-RAG usage:
//
//	go run -tags sqlite_fts5 ./cmd/eval \
//	  --dataset-type legalbench \
//	  --corpus-dir ./data/legalbench-rag-mini/corpus \
//	  --benchmark-file ./data/legalbench-rag-mini/benchmarks/cuad.json \
//	  --benchmark-file ./data/legalbench-rag-mini/benchmarks/contractnli.json \
//	  --chat-provider groq \
//	  --chat-model openai/gpt-oss-120b
//
// GDPR usage (Graph RAG):
//
//	go run -tags sqlite_fts5 ./cmd/eval \
//	  --dataset-type gdpr \
//	  --pdf ~/Downloads/CELEX_32016R0679_EN_TXT.pdf \
//	  --chat-provider ollama --chat-model llama3.1:8b \
//	  --embed-provider openai --embed-model text-embedding-3-small \
//	  --difficulty all
//
// GDPR full-context baseline (Gemini):
//
//	go run -tags sqlite_fts5 ./cmd/eval \
//	  --dataset-type gdpr \
//	  --pdf ~/Downloads/CELEX_32016R0679_EN_TXT.pdf \
//	  --full-context \
//	  --fc-provider gemini --fc-model gemini-2.0-flash \
//	  --difficulty all
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
	"github.com/bbiangul/go-reason/llm"
	"github.com/bbiangul/go-reason/parser"
)

// stringSlice implements flag.Value for multi-value string flags.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(val string) error {
	*s = append(*s, val)
	return nil
}

func main() {
	var benchmarkFiles stringSlice

	var (
		pdfPath       = flag.String("pdf", "", "Path to document file (for ALTAVision/GDPR)")
		corpusDir     = flag.String("corpus-dir", "", "Path to corpus directory (for LegalBench-RAG)")
		datasetType   = flag.String("dataset-type", "altavision", "Dataset type: altavision, legalbench, gdpr")
		fullContext   = flag.Bool("full-context", false, "Run full-context baseline (send entire doc to LLM, no RAG)")
		fcProvider    = flag.String("fc-provider", "gemini", "Full-context LLM provider")
		fcModel       = flag.String("fc-model", "gemini-2.0-flash", "Full-context LLM model")
		fcAPIKey      = flag.String("fc-api-key", "", "Full-context provider API key (default: from env)")
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
		skipGraph     = flag.Bool("skip-graph", false, "Skip knowledge graph extraction during ingestion (faster)")
		maxTests      = flag.Int("max-tests", 0, "Max tests per benchmark file (0=all; 194 matches LegalBench-RAG-mini)")
		judgeProvider = flag.String("judge-provider", "", "LLM provider for accuracy judge (enables LLM-as-judge; e.g., gemini)")
		judgeModel    = flag.String("judge-model", "", "Judge LLM model name (e.g., gemini-2.0-flash-lite)")
		judgeAPIKey   = flag.String("judge-api-key", "", "Judge provider API key (default: from env)")
	)
	flag.Var(&benchmarkFiles, "benchmark-file", "Path to benchmark JSON file (repeatable, for LegalBench-RAG)")
	flag.Parse()

	// Validate flags based on dataset type
	switch strings.ToLower(*datasetType) {
	case "altavision":
		if *pdfPath == "" && !*skipIngest {
			log.Fatal("--pdf flag is required for altavision (or use --skip-ingest with --db)")
		}
	case "gdpr":
		if *pdfPath == "" && !*skipIngest && !*fullContext {
			log.Fatal("--pdf flag is required for gdpr (or use --skip-ingest with --db, or --full-context)")
		}
		if *fullContext && *pdfPath == "" {
			log.Fatal("--pdf is required for --full-context (used to extract document text)")
		}
	case "legalbench":
		if *corpusDir == "" && !*skipIngest {
			log.Fatal("--corpus-dir is required for legalbench (or use --skip-ingest with --db)")
		}
		if len(benchmarkFiles) == 0 {
			log.Fatal("at least one --benchmark-file is required for legalbench")
		}
	default:
		log.Fatalf("unknown --dataset-type: %s (use: altavision, legalbench, gdpr)", *datasetType)
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
		case "gemini":
			apiKey = os.Getenv("GEMINI_API_KEY")
		}
	}
	if apiKey == "" && *chatProvider != "ollama" && *chatProvider != "lmstudio" && !*fullContext {
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
		case "gemini":
			embedKey = os.Getenv("GEMINI_API_KEY")
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
		case "gemini":
			chatURL = "https://generativelanguage.googleapis.com/v1beta/openai"
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
		case "gemini":
			embedURL = "https://generativelanguage.googleapis.com/v1beta/openai"
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
		"dataset_type":      *datasetType,
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
		"difficulty":  *difficulty,
	}
	if *pdfPath != "" {
		meta["pdf"] = filepath.Base(*pdfPath)
	}
	if *corpusDir != "" {
		meta["corpus_dir"] = *corpusDir
	}
	if len(benchmarkFiles) > 0 {
		meta["benchmark_files"] = []string(benchmarkFiles)
	}
	if *maxTests > 0 {
		meta["max_tests_per_benchmark"] = *maxTests
	}
	if *fullContext {
		meta["full_context"] = true
		meta["fc_provider"] = *fcProvider
		meta["fc_model"] = *fcModel
	}
	writeJSON(filepath.Join(runDir, "metadata.json"), meta)

	ctx := context.Background()

	// --- Full-context evaluation path (no engine needed) ---
	if *fullContext {
		runFullContext(ctx, *pdfPath, *fcProvider, *fcModel, *fcAPIKey, *difficulty, *maxTests, runDir, meta, *outputFile)
		return
	}

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
		SkipGraph:           *skipGraph,
		GraphConcurrency:    *graphConc,
	}

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
	} else if *corpusDir != "" {
		// Directory ingestion (LegalBench-RAG)
		// When --max-tests is set, only ingest documents referenced by selected tests.
		var usedFiles map[string]struct{}
		if *maxTests > 0 && len(benchmarkFiles) > 0 {
			lbCfg := eval.LegalBenchConfig{
				BenchmarkFiles:       []string(benchmarkFiles),
				CorpusDir:            *corpusDir,
				MaxTestsPerBenchmark: *maxTests,
			}
			var err error
			usedFiles, err = eval.UsedCorpusFiles(lbCfg)
			if err != nil {
				log.Fatalf("computing used corpus files: %v", err)
			}
			fmt.Fprintf(os.Stderr, "Mini subset: ingesting %d referenced documents (of full corpus)\n", len(usedFiles))
		}

		fmt.Fprintf(os.Stderr, "Ingesting corpus directory: %s\n", *corpusDir)
		ingestStart := time.Now()
		docCount := 0
		err := filepath.Walk(*corpusDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".txt" && ext != ".pdf" && ext != ".docx" {
				return nil
			}
			// If we have a used-files filter, skip unreferenced documents.
			if usedFiles != nil {
				relPath, relErr := filepath.Rel(*corpusDir, path)
				if relErr != nil {
					return nil
				}
				if _, ok := usedFiles[relPath]; !ok {
					return nil
				}
			}
			docCount++
			fmt.Fprintf(os.Stderr, "  [%d] Ingesting %s\n", docCount, filepath.Base(path))
			_, ingestErr := engine.Ingest(ctx, path)
			if ingestErr != nil {
				slog.Warn("ingest: skipping file", "path", path, "error", ingestErr)
			}
			return nil
		})
		if err != nil {
			log.Fatalf("walking corpus directory: %v", err)
		}
		ingestElapsed = time.Since(ingestStart)
		fmt.Fprintf(os.Stderr, "Ingested %d documents in %s\n", docCount, ingestElapsed.Round(time.Millisecond))
	} else if *pdfPath != "" {
		// Single file ingestion (ALTAVision)
		fmt.Fprintf(os.Stderr, "Ingesting file: %s\n", *pdfPath)
		ingestStart := time.Now()
		docID, err := engine.Ingest(ctx, *pdfPath)
		if err != nil {
			log.Fatalf("ingesting file: %v", err)
		}
		ingestElapsed = time.Since(ingestStart)
		fmt.Fprintf(os.Stderr, "Ingested document ID %d in %s\n", docID, ingestElapsed.Round(time.Millisecond))
	}

	// Select datasets based on type
	var datasets []eval.Dataset
	var groundTruth map[string][]eval.GroundTruthSpan

	switch strings.ToLower(*datasetType) {
	case "legalbench":
		lbCfg := eval.LegalBenchConfig{
			BenchmarkFiles:       []string(benchmarkFiles),
			CorpusDir:            *corpusDir,
			MaxTestsPerBenchmark: *maxTests,
		}
		var err error
		datasets, err = eval.LoadLegalBenchDatasets(lbCfg)
		if err != nil {
			log.Fatalf("loading LegalBench-RAG datasets: %v", err)
		}
		groundTruth, err = eval.LoadLegalBenchGroundTruth(lbCfg)
		if err != nil {
			log.Fatalf("loading LegalBench-RAG ground truth: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Loaded %d LegalBench-RAG datasets with %d ground-truth queries\n",
			len(datasets), len(groundTruth))
	case "gdpr":
		datasets = selectDatasets(eval.GDPRAllDatasets(), *difficulty)
		if len(datasets) == 0 {
			log.Fatalf("unknown difficulty: %s (use: easy, medium, hard, super-hard, all)", *difficulty)
		}
		// Apply --max-tests limit if set
		if *maxTests > 0 {
			datasets = limitDatasetTests(datasets, *maxTests)
		}
	default:
		datasets = selectDatasets(eval.ALTAVisionAllDatasets(), *difficulty)
		if len(datasets) == 0 {
			log.Fatalf("unknown difficulty: %s (use: easy, medium, hard, super-hard, all)", *difficulty)
		}
		// Apply --max-tests limit if set
		if *maxTests > 0 {
			datasets = limitDatasetTests(datasets, *maxTests)
		}
	}

	evaluator := eval.NewEvaluator(engine)
	if groundTruth != nil {
		evaluator.SetGroundTruth(groundTruth)
	}

	// Setup LLM judge if configured
	if *judgeProvider != "" {
		judgeKey := *judgeAPIKey
		if judgeKey == "" {
			switch *judgeProvider {
			case "gemini":
				judgeKey = os.Getenv("GEMINI_API_KEY")
			case "openai":
				judgeKey = os.Getenv("OPENAI_API_KEY")
			case "groq":
				judgeKey = os.Getenv("GROQ_API_KEY")
			case "openrouter":
				judgeKey = os.Getenv("OPENROUTER_API_KEY")
			}
		}

		var judgeBaseURL string
		switch *judgeProvider {
		case "openrouter":
			judgeBaseURL = "https://openrouter.ai/api"
		case "openai":
			judgeBaseURL = "https://api.openai.com"
		case "groq":
			judgeBaseURL = "https://api.groq.com/openai"
		case "gemini":
			judgeBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
		case "ollama":
			judgeBaseURL = "http://localhost:11434"
		case "lmstudio":
			judgeBaseURL = "http://localhost:1234"
		}

		judge, err := llm.NewProvider(llm.Config{
			Provider: *judgeProvider,
			Model:    *judgeModel,
			BaseURL:  judgeBaseURL,
			APIKey:   judgeKey,
		})
		if err != nil {
			log.Fatalf("creating judge LLM provider: %v", err)
		}
		evaluator.SetJudge(judge, *judgeModel)
		fmt.Fprintf(os.Stderr, "LLM judge enabled: %s/%s\n", *judgeProvider, *judgeModel)

		meta["judge_provider"] = *judgeProvider
		meta["judge_model"] = *judgeModel
		writeJSON(filepath.Join(runDir, "metadata.json"), meta)
	}

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

func selectDatasets(all map[string]eval.Dataset, difficulty string) []eval.Dataset {
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
	case "graph-test":
		return []eval.Dataset{all[eval.DifficultyGraphTest]}
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

// limitDatasetTests truncates each dataset's test list to maxTests.
func limitDatasetTests(datasets []eval.Dataset, maxTests int) []eval.Dataset {
	result := make([]eval.Dataset, len(datasets))
	for i, ds := range datasets {
		result[i] = ds
		if len(ds.Tests) > maxTests {
			result[i].Tests = ds.Tests[:maxTests]
		}
	}
	return result
}

// runFullContext runs the full-context baseline evaluation (no RAG engine).
func runFullContext(ctx context.Context, pdfPath, providerName, model, apiKey, difficulty string, maxTests int, runDir string, meta map[string]interface{}, outputFile string) {
	totalStart := time.Now()

	// Resolve API key from env if not provided
	if apiKey == "" {
		switch providerName {
		case "gemini":
			apiKey = os.Getenv("GEMINI_API_KEY")
		case "openai":
			apiKey = os.Getenv("OPENAI_API_KEY")
		case "groq":
			apiKey = os.Getenv("GROQ_API_KEY")
		case "openrouter":
			apiKey = os.Getenv("OPENROUTER_API_KEY")
		}
	}
	if apiKey == "" && providerName != "ollama" && providerName != "lmstudio" {
		log.Fatalf("API key required for full-context provider %q", providerName)
	}

	// Resolve base URL
	var baseURL string
	switch providerName {
	case "openrouter":
		baseURL = "https://openrouter.ai/api"
	case "openai":
		baseURL = "https://api.openai.com"
	case "groq":
		baseURL = "https://api.groq.com/openai"
	case "gemini":
		baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	case "ollama":
		baseURL = "http://localhost:11434"
	case "lmstudio":
		baseURL = "http://localhost:1234"
	}

	provider, err := llm.NewProvider(llm.Config{
		Provider: providerName,
		Model:    model,
		BaseURL:  baseURL,
		APIKey:   apiKey,
	})
	if err != nil {
		log.Fatalf("creating full-context LLM provider: %v", err)
	}

	// Extract document text
	docText := extractDocText(ctx, pdfPath)
	fmt.Fprintf(os.Stderr, "Extracted %d characters from %s\n", len(docText), filepath.Base(pdfPath))

	// Select datasets
	allDatasets := eval.GDPRAllDatasets()
	datasets := selectDatasets(allDatasets, difficulty)
	if len(datasets) == 0 {
		log.Fatalf("unknown difficulty: %s (use: easy, medium, hard, super-hard, all)", difficulty)
	}
	// Apply --max-tests limit if set
	if maxTests > 0 {
		datasets = limitDatasetTests(datasets, maxTests)
	}

	fce := eval.NewFullContextEvaluator(provider, docText)

	var allReports []*eval.Report
	evalStart := time.Now()

	for _, ds := range datasets {
		fmt.Fprintf(os.Stderr, "\nRunning full-context %s (%d tests)...\n", ds.Name, len(ds.Tests))
		report, err := fce.Run(ctx, ds)
		if err != nil {
			log.Fatalf("running full-context %s: %v", ds.Name, err)
		}
		allReports = append(allReports, report)
		fmt.Println(eval.FormatReport(report))
		fmt.Println()
	}

	evalElapsed := time.Since(evalStart)
	totalElapsed := time.Since(totalStart)

	meta["eval_elapsed"] = evalElapsed.Round(time.Millisecond).String()
	meta["total_elapsed"] = totalElapsed.Round(time.Millisecond).String()
	writeJSON(filepath.Join(runDir, "metadata.json"), meta)

	reportPath := filepath.Join(runDir, "eval-report.json")
	writeJSON(reportPath, allReports)
	fmt.Fprintf(os.Stderr, "Eval report written to: %s\n", reportPath)

	if outputFile != "" {
		writeJSON(outputFile, allReports)
		fmt.Fprintf(os.Stderr, "JSON report also written to: %s\n", outputFile)
	}

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

// extractDocText parses a PDF/text file and returns its full text content.
func extractDocText(ctx context.Context, path string) string {
	reg := parser.NewRegistry()
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	p, err := reg.Get(ext)
	if err != nil {
		log.Fatalf("no parser for %q: %v", ext, err)
	}
	result, err := p.Parse(ctx, path)
	if err != nil {
		log.Fatalf("parsing %s: %v", path, err)
	}
	var sb strings.Builder
	for _, sec := range result.Sections {
		if sec.Heading != "" {
			sb.WriteString(sec.Heading)
			sb.WriteByte('\n')
		}
		sb.WriteString(sec.Content)
		sb.WriteString("\n\n")
	}
	return sb.String()
}
