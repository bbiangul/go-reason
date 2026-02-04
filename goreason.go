package goreason

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/brunobiangulo/goreason/chunker"
	"github.com/brunobiangulo/goreason/graph"
	"github.com/brunobiangulo/goreason/llm"
	"github.com/brunobiangulo/goreason/parser"
	"github.com/brunobiangulo/goreason/reasoning"
	"github.com/brunobiangulo/goreason/retrieval"
	"github.com/brunobiangulo/goreason/store"
)

// Engine is the main entry point for the Graph RAG engine.
type Engine interface {
	// Ingest parses, chunks, embeds, and builds graph for a document.
	// Returns document ID. Skips if content hash unchanged.
	Ingest(ctx context.Context, path string, opts ...IngestOption) (int64, error)

	// Query runs a question through hybrid retrieval + multi-round reasoning.
	Query(ctx context.Context, question string, opts ...QueryOption) (*Answer, error)

	// Update re-checks a document by hash. Re-ingests if changed.
	Update(ctx context.Context, path string) (bool, error)

	// UpdateAll checks all ingested documents for changes.
	UpdateAll(ctx context.Context) ([]UpdateResult, error)

	// Delete removes a document and all associated data.
	Delete(ctx context.Context, documentID int64) error

	// ListDocuments returns all ingested documents.
	ListDocuments(ctx context.Context) ([]Document, error)

	// Store returns the underlying store for diagnostic access (e.g. eval ground-truth checks).
	Store() *store.Store

	// Close cleanly shuts down the engine.
	Close() error
}

// Answer represents the result of a query.
type Answer struct {
	Text             string                `json:"text"`
	Confidence       float64               `json:"confidence"`
	Sources          []Source              `json:"sources"`
	Reasoning        []Step                `json:"reasoning"`
	RetrievalTrace   *retrieval.SearchTrace `json:"retrieval_trace,omitempty"`
	ModelUsed        string                `json:"model_used"`
	Rounds           int                   `json:"rounds"`
	PromptTokens     int                   `json:"prompt_tokens"`
	CompletionTokens int                   `json:"completion_tokens"`
	TotalTokens      int                   `json:"total_tokens"`
}

// Source represents a retrieved source chunk backing an answer.
type Source struct {
	ChunkID    int64   `json:"chunk_id"`
	DocumentID int64   `json:"document_id"`
	Filename   string  `json:"filename"`
	Content    string  `json:"content"`
	Heading    string  `json:"heading"`
	PageNumber int     `json:"page_number"`
	Score      float64 `json:"score"`
}

// Step represents a single reasoning round in the multi-round pipeline.
type Step struct {
	Round      int      `json:"round"`
	Action     string   `json:"action"`
	Input      string   `json:"input,omitempty"`
	Output     string   `json:"output,omitempty"`
	Prompt     string   `json:"prompt,omitempty"`
	Response   string   `json:"response,omitempty"`
	Validation string   `json:"validation,omitempty"`
	ChunksUsed int      `json:"chunks_used,omitempty"`
	Tokens     int      `json:"tokens,omitempty"`
	ElapsedMs  int64    `json:"elapsed_ms,omitempty"`
	Issues     []string `json:"issues,omitempty"`
}

// Document represents an ingested document.
type Document struct {
	ID          int64             `json:"id"`
	Path        string            `json:"path"`
	Filename    string            `json:"filename"`
	Format      string            `json:"format"`
	ContentHash string            `json:"content_hash"`
	ParseMethod string            `json:"parse_method"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// UpdateResult reports the outcome of a document update check.
type UpdateResult struct {
	DocumentID int64  `json:"document_id"`
	Path       string `json:"path"`
	Changed    bool   `json:"changed"`
	Error      error  `json:"error,omitempty"`
}

// IngestOption configures ingestion behavior.
type IngestOption func(*ingestOptions)

type ingestOptions struct {
	forceReparse bool
	parseMethod  string
	metadata     map[string]string
}

// WithForceReparse forces re-parsing even if the hash hasn't changed.
func WithForceReparse() IngestOption {
	return func(o *ingestOptions) { o.forceReparse = true }
}

// WithParseMethod overrides the automatic parse method selection.
func WithParseMethod(method string) IngestOption {
	return func(o *ingestOptions) { o.parseMethod = method }
}

// WithMetadata attaches custom metadata to the ingested document.
func WithMetadata(metadata map[string]string) IngestOption {
	return func(o *ingestOptions) { o.metadata = metadata }
}

// QueryOption configures query behavior.
type QueryOption func(*queryOptions)

type queryOptions struct {
	maxResults int
	maxRounds  int
	weightVec  float64
	weightFTS  float64
	weightGraph float64
}

// WithMaxResults sets the maximum number of chunks to retrieve.
func WithMaxResults(n int) QueryOption {
	return func(o *queryOptions) { o.maxResults = n }
}

// WithMaxRounds overrides the maximum reasoning rounds for this query.
func WithMaxRounds(n int) QueryOption {
	return func(o *queryOptions) { o.maxRounds = n }
}

// WithWeights overrides the retrieval weights for this query.
func WithWeights(vec, fts, graph float64) QueryOption {
	return func(o *queryOptions) {
		o.weightVec = vec
		o.weightFTS = fts
		o.weightGraph = graph
	}
}

// engine is the concrete implementation of Engine.
type engine struct {
	cfg       Config
	store     *store.Store
	chatLLM   llm.Provider
	embedLLM  llm.Provider
	visionLLM llm.Provider
	parsers   *parser.Registry
	chunkr    *chunker.Chunker
	graphB    *graph.Builder
	retriever *retrieval.Engine
	reasoner  *reasoning.Engine
}

// New creates a new GoReason engine with the given configuration.
func New(cfg Config) (Engine, error) {
	// Resolve database path from config (DBPath > DBName+StorageDir > default)
	dbPath := cfg.resolveDBPath()

	// Apply defaults for zero values
	if cfg.EmbeddingDim == 0 {
		cfg.EmbeddingDim = 768
	}

	// Open store
	s, err := store.New(dbPath, cfg.EmbeddingDim)
	if err != nil {
		return nil, fmt.Errorf("opening store: %w", err)
	}

	// Create LLM providers
	chatLLM, err := llm.NewProvider(llm.Config{
		Provider: cfg.Chat.Provider,
		Model:    cfg.Chat.Model,
		BaseURL:  cfg.Chat.BaseURL,
		APIKey:   cfg.Chat.APIKey,
	})
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("creating chat provider: %w", err)
	}

	embedLLM, err := llm.NewProvider(llm.Config{
		Provider: cfg.Embedding.Provider,
		Model:    cfg.Embedding.Model,
		BaseURL:  cfg.Embedding.BaseURL,
		APIKey:   cfg.Embedding.APIKey,
	})
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("creating embedding provider: %w", err)
	}

	var visionLLM llm.Provider
	if cfg.Vision.Provider != "" {
		visionLLM, err = llm.NewProvider(llm.Config{
			Provider: cfg.Vision.Provider,
			Model:    cfg.Vision.Model,
			BaseURL:  cfg.Vision.BaseURL,
			APIKey:   cfg.Vision.APIKey,
		})
		if err != nil {
			s.Close()
			return nil, fmt.Errorf("creating vision provider: %w", err)
		}
	}

	// Create parser registry
	reg := parser.NewRegistry()
	if cfg.LlamaParse != nil {
		reg.SetLlamaParse(parser.LlamaParseConfig{
			APIKey:  cfg.LlamaParse.APIKey,
			BaseURL: cfg.LlamaParse.BaseURL,
		})
	}

	// Create chunker
	chunkr := chunker.New(chunker.Config{
		MaxTokens: cfg.MaxChunkTokens,
		Overlap:   cfg.ChunkOverlap,
	})

	// Create graph builder
	graphB := graph.NewBuilder(s, chatLLM, embedLLM, cfg.GraphConcurrency)

	// Create retrieval engine (chatLLM enables cross-language query translation)
	retriever := retrieval.New(s, embedLLM, chatLLM, retrieval.Config{
		WeightVector: cfg.WeightVector,
		WeightFTS:    cfg.WeightFTS,
		WeightGraph:  cfg.WeightGraph,
	})

	// Create reasoning engine
	reasoner := reasoning.New(chatLLM, reasoning.Config{
		MaxRounds:           cfg.MaxRounds,
		ConfidenceThreshold: cfg.ConfidenceThreshold,
	})

	return &engine{
		cfg:       cfg,
		store:     s,
		chatLLM:   chatLLM,
		embedLLM:  embedLLM,
		visionLLM: visionLLM,
		parsers:   reg,
		chunkr:    chunkr,
		graphB:    graphB,
		retriever: retriever,
		reasoner:  reasoner,
	}, nil
}

// Ingest processes a document through the full pipeline.
func (e *engine) Ingest(ctx context.Context, path string, opts ...IngestOption) (int64, error) {
	options := &ingestOptions{}
	for _, o := range opts {
		o(options)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return 0, fmt.Errorf("resolving path: %w", err)
	}

	// Compute file hash
	hash, err := fileHash(absPath)
	if err != nil {
		return 0, fmt.Errorf("hashing file: %w", err)
	}

	// Check if document already exists with same hash
	if !options.forceReparse {
		existing, err := e.store.GetDocumentByPath(ctx, absPath)
		if err == nil && existing.ContentHash == hash {
			return existing.ID, nil // no change
		}
	}

	// Determine format
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(absPath), "."))
	format := ext

	// Serialize metadata if present
	var metadataJSON string
	if options.metadata != nil {
		data, _ := json.Marshal(options.metadata)
		metadataJSON = string(data)
	}

	// Set status to processing
	filename := filepath.Base(absPath)
	docID, err := e.store.UpsertDocument(ctx, store.Document{
		Path:        absPath,
		Filename:    filename,
		Format:      format,
		ContentHash: hash,
		ParseMethod: "pending",
		Status:      "processing",
		Metadata:    metadataJSON,
	})
	if err != nil {
		return 0, fmt.Errorf("upserting document: %w", err)
	}

	// Parse
	parseMethod := options.parseMethod
	if parseMethod == "" {
		parseMethod = "native"
	}

	slog.Info("ingest: parsing document", "file", filename, "format", format, "doc_id", docID)
	parseStart := time.Now()

	p, err := e.parsers.Get(format)
	if err != nil {
		e.store.UpdateDocumentStatus(ctx, docID, "error")
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}

	parsed, err := p.Parse(ctx, absPath)
	if err != nil {
		e.store.UpdateDocumentStatus(ctx, docID, "error")
		return 0, fmt.Errorf("%w: %v", ErrParsingFailed, err)
	}
	parseMethod = parsed.Method

	slog.Info("ingest: parsing complete",
		"file", filename, "method", parseMethod,
		"sections", len(parsed.Sections), "elapsed", time.Since(parseStart).Round(time.Millisecond))

	// Update parse method
	e.store.UpdateDocumentParseMethod(ctx, docID, parseMethod)

	// Chunk
	chunkStart := time.Now()
	chunks := e.chunkr.Chunk(parsed.Sections)
	slog.Info("ingest: chunking complete",
		"file", filename, "chunks", len(chunks),
		"max_tokens", e.cfg.MaxChunkTokens, "overlap", e.cfg.ChunkOverlap,
		"elapsed", time.Since(chunkStart).Round(time.Millisecond))

	// Delete old chunks/embeddings/entities for this document (re-ingest)
	if err := e.store.DeleteDocumentData(ctx, docID); err != nil {
		return 0, fmt.Errorf("cleaning old data: %w", err)
	}

	// Store chunks and generate embeddings
	for i := range chunks {
		chunks[i].DocumentID = docID
	}

	chunkIDs, err := e.store.InsertChunks(ctx, chunks)
	if err != nil {
		e.store.UpdateDocumentStatus(ctx, docID, "error")
		return 0, fmt.Errorf("inserting chunks: %w", err)
	}

	// Generate embeddings concurrently
	slog.Info("ingest: generating embeddings", "file", filename, "chunks", len(chunks))
	embedStart := time.Now()
	if err := e.embedChunks(ctx, chunks, chunkIDs); err != nil {
		e.store.UpdateDocumentStatus(ctx, docID, "error")
		return 0, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}
	slog.Info("ingest: embeddings complete",
		"file", filename, "chunks", len(chunks),
		"elapsed", time.Since(embedStart).Round(time.Millisecond))

	// Build knowledge graph (optional — can be skipped for faster ingestion).
	if !e.cfg.SkipGraph {
		slog.Info("ingest: building knowledge graph", "file", filename, "chunks", len(chunks),
			"concurrency", e.cfg.GraphConcurrency)
		graphStart := time.Now()
		if err := e.graphB.Build(ctx, docID, chunks, chunkIDs); err != nil {
			slog.Warn("graph build had errors (non-fatal)", "doc_id", docID, "error", err)
		}
		slog.Info("ingest: graph build complete",
			"file", filename, "elapsed", time.Since(graphStart).Round(time.Millisecond))

		// Run community detection on the updated graph.
		slog.Info("ingest: detecting communities", "file", filename)
		communities, err := graph.DetectCommunities(ctx, e.store)
		if err != nil {
			slog.Warn("community detection failed (non-fatal)", "error", err)
		} else if len(communities) > 0 {
			slog.Info("ingest: summarizing communities", "count", len(communities))
			if err := graph.SummarizeCommunities(ctx, e.store, e.chatLLM, communities); err != nil {
				slog.Warn("community summarization failed (non-fatal)", "error", err)
			}
		}
	} else {
		slog.Info("ingest: graph building skipped (skip_graph=true)", "doc_id", docID)
	}

	totalElapsed := time.Since(parseStart)
	slog.Info("ingest: document ready",
		"file", filename, "doc_id", docID,
		"total_elapsed", totalElapsed.Round(time.Millisecond))
	e.store.UpdateDocumentStatus(ctx, docID, "ready")
	return docID, nil
}

// Query runs hybrid retrieval and multi-round reasoning.
func (e *engine) Query(ctx context.Context, question string, opts ...QueryOption) (*Answer, error) {
	options := &queryOptions{
		maxResults:  20,
		maxRounds:   e.cfg.MaxRounds,
		weightVec:   e.cfg.WeightVector,
		weightFTS:   e.cfg.WeightFTS,
		weightGraph: e.cfg.WeightGraph,
	}
	for _, o := range opts {
		o(options)
	}

	// Hybrid retrieval
	results, searchTrace, err := e.retriever.Search(ctx, question, retrieval.SearchOptions{
		MaxResults:  options.maxResults,
		WeightVec:   options.weightVec,
		WeightFTS:   options.weightFTS,
		WeightGraph: options.weightGraph,
	})
	if err != nil {
		return nil, fmt.Errorf("retrieval: %w", err)
	}
	if len(results) == 0 {
		return nil, ErrNoResults
	}

	// Multi-round reasoning
	rAnswer, err := e.reasoner.Reason(ctx, question, results, reasoning.Options{
		MaxRounds: options.maxRounds,
	})
	if err != nil {
		return nil, fmt.Errorf("reasoning: %w", err)
	}

	// Follow-up retrieval for synthesis queries with a full initial window.
	// When the first retrieval filled the entire result window, there are
	// likely more relevant chunks we didn't see. Extract identifiers from
	// the round-1 answer that don't appear in retrieved chunks (these may
	// be hallucinated or from LLM prior knowledge) and do a targeted FTS
	// search to find supporting evidence or disprove them.
	//
	// Gate: compare against FusedResults (the actual window size after
	// synthesis widening) rather than the caller's original maxResults,
	// so we only fire when the widened window was truly filled.
	if searchTrace != nil && searchTrace.SynthesisMode && searchTrace.FusedResults >= searchTrace.MaxRequested {
		// The widened window was filled — there are likely more chunks.
		missing := extractMissingTerms(rAnswer.Text, results)
		if len(missing) > 0 {
			slog.Debug("retrieval: synthesis follow-up",
				"missing_terms", missing, "count", len(missing))

			// Replace hyphens with spaces so FTS tokenisation matches the
			// index (FTS5 treats hyphens as separators). E.g. "ISO 13849-1"
			// becomes "ISO 13849 1" → tokens match the indexed content.
			ftsTerms := make([]string, len(missing))
			for i, m := range missing {
				ftsTerms[i] = strings.ReplaceAll(m, "-", " ")
			}
			ftsQuery := strings.Join(ftsTerms, " OR ")

			extraResults, followTrace, ferr := e.retriever.Search(ctx, ftsQuery, retrieval.SearchOptions{
				MaxResults:  15,
				WeightFTS:   2.0,
				WeightVec:   0.5,
				WeightGraph: 1.0,
			})

			// Record follow-up in the original trace for diagnostics.
			searchTrace.FollowUpTerms = missing
			if followTrace != nil {
				searchTrace.FollowUpResults = followTrace.FusedResults
			}

			if ferr == nil && len(extraResults) > 0 {
				merged := mergeResults(results, extraResults)
				slog.Debug("retrieval: synthesis follow-up merged",
					"extra", len(extraResults), "total", len(merged))

				// Accumulate token counts from the first reasoning call
				// so the final answer reflects total usage.
				firstPromptTokens := rAnswer.PromptTokens
				firstCompletionTokens := rAnswer.CompletionTokens

				// Re-run reasoning with expanded context
				rAnswer2, rerr := e.reasoner.Reason(ctx, question, merged, reasoning.Options{
					MaxRounds: options.maxRounds,
				})
				if rerr == nil {
					rAnswer2.PromptTokens += firstPromptTokens
					rAnswer2.CompletionTokens += firstCompletionTokens
					rAnswer2.TotalTokens = rAnswer2.PromptTokens + rAnswer2.CompletionTokens
					rAnswer2.Rounds += rAnswer.Rounds
					rAnswer = rAnswer2
					results = merged
				}
			}
		}
	}

	// Convert reasoning.Answer -> goreason.Answer
	answer := &Answer{
		Text:             rAnswer.Text,
		Confidence:       rAnswer.Confidence,
		RetrievalTrace:   searchTrace,
		ModelUsed:        rAnswer.ModelUsed,
		Rounds:           rAnswer.Rounds,
		PromptTokens:     rAnswer.PromptTokens,
		CompletionTokens: rAnswer.CompletionTokens,
		TotalTokens:      rAnswer.TotalTokens,
	}
	for _, s := range rAnswer.Sources {
		answer.Sources = append(answer.Sources, Source{
			ChunkID:    s.ChunkID,
			DocumentID: s.DocumentID,
			Filename:   s.Filename,
			Content:    s.Content,
			Heading:    s.Heading,
			PageNumber: s.PageNumber,
			Score:      s.Score,
		})
	}
	for _, s := range rAnswer.Reasoning {
		answer.Reasoning = append(answer.Reasoning, Step{
			Round:      s.Round,
			Action:     s.Action,
			Input:      s.Input,
			Output:     s.Output,
			Prompt:     s.Prompt,
			Response:   s.Response,
			Validation: s.Validation,
			ChunksUsed: s.ChunksUsed,
			Tokens:     s.Tokens,
			ElapsedMs:  s.ElapsedMs,
			Issues:     s.Issues,
		})
	}

	// Log query
	e.store.LogQuery(ctx, store.QueryLog{
		Query:            question,
		Answer:           answer.Text,
		Confidence:       answer.Confidence,
		Sources:          answer.Sources,
		RetrievalMethod:  "hybrid",
		ModelUsed:        answer.ModelUsed,
		Rounds:           answer.Rounds,
		PromptTokens:     answer.PromptTokens,
		CompletionTokens: answer.CompletionTokens,
		TotalTokens:      answer.TotalTokens,
	})

	return answer, nil
}

// Update checks if a document has changed and re-ingests if needed.
func (e *engine) Update(ctx context.Context, path string) (bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("resolving path: %w", err)
	}

	doc, err := e.store.GetDocumentByPath(ctx, absPath)
	if err != nil {
		return false, fmt.Errorf("%w: %s", ErrDocumentNotFound, absPath)
	}

	hash, err := fileHash(absPath)
	if err != nil {
		return false, fmt.Errorf("hashing file: %w", err)
	}

	if hash == doc.ContentHash {
		return false, nil
	}

	_, err = e.Ingest(ctx, absPath, WithForceReparse())
	if err != nil {
		return false, err
	}
	return true, nil
}

// UpdateAll checks all documents for changes.
func (e *engine) UpdateAll(ctx context.Context) ([]UpdateResult, error) {
	docs, err := e.store.ListDocuments(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]UpdateResult, 0, len(docs))
	for _, doc := range docs {
		changed, err := e.Update(ctx, doc.Path)
		results = append(results, UpdateResult{
			DocumentID: doc.ID,
			Path:       doc.Path,
			Changed:    changed,
			Error:      err,
		})
	}
	return results, nil
}

// Delete removes a document and all its associated data.
func (e *engine) Delete(ctx context.Context, documentID int64) error {
	return e.store.DeleteDocument(ctx, documentID)
}

// ListDocuments returns all ingested documents.
func (e *engine) ListDocuments(ctx context.Context) ([]Document, error) {
	docs, err := e.store.ListDocuments(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Document, len(docs))
	for i, d := range docs {
		result[i] = Document{
			ID:          d.ID,
			Path:        d.Path,
			Filename:    d.Filename,
			Format:      d.Format,
			ContentHash: d.ContentHash,
			ParseMethod: d.ParseMethod,
			Status:      d.Status,
			CreatedAt:   d.CreatedAt,
			UpdatedAt:   d.UpdatedAt,
		}
		if d.Metadata != "" {
			_ = json.Unmarshal([]byte(d.Metadata), &result[i].Metadata)
		}
	}
	return result, nil
}

// Store returns the underlying store for diagnostic access.
func (e *engine) Store() *store.Store {
	return e.store
}

// Close shuts down the engine.
func (e *engine) Close() error {
	return e.store.Close()
}

// maxEmbedChars is the maximum character length for a single text sent to the
// embedding model. Most embedding models have a context window of 8192 tokens;
// using ~24000 chars (~6000 tokens) leaves headroom for varied tokenisers and
// languages where token/char ratios differ from English.
const maxEmbedChars = 24000

// truncateForEmbed truncates text to maxEmbedChars on a word boundary.
func truncateForEmbed(text string) string {
	if len(text) <= maxEmbedChars {
		return text
	}
	// Cut at the last space before the limit to avoid splitting a word.
	cut := strings.LastIndex(text[:maxEmbedChars], " ")
	if cut <= 0 {
		cut = maxEmbedChars
	}
	return text[:cut]
}

// embedChunks generates embeddings for chunks in batches.
// Individual batch failures trigger per-text fallback so a single oversized
// text does not cause the entire batch to be lost.
func (e *engine) embedChunks(ctx context.Context, chunks []store.Chunk, chunkIDs []int64) error {
	const batchSize = 32
	var failed int

	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		texts := make([]string, end-i)
		for j := i; j < end; j++ {
			prefix := ""
			if chunks[j].Heading != "" {
				prefix = chunks[j].Heading + ": "
			}
			texts[j-i] = truncateForEmbed(prefix + chunks[j].Content)
		}

		embeddings, err := e.embedLLM.Embed(ctx, texts)
		if err != nil {
			// Batch failed — fall back to embedding each text individually
			// so one oversized text doesn't lose the entire batch.
			slog.Warn("embedding batch failed, falling back to individual",
				"batch_start", i, "batch_end", end, "error", err)
			for j, text := range texts {
				single, serr := e.embedLLM.Embed(ctx, []string{text})
				if serr != nil {
					slog.Warn("embedding single text failed",
						"chunk_id", chunkIDs[i+j], "error", serr)
					failed++
					continue
				}
				if len(single) == 0 || len(single[0]) == 0 {
					failed++
					continue
				}
				if serr := e.store.InsertEmbedding(ctx, chunkIDs[i+j], single[0]); serr != nil {
					slog.Warn("storing embedding failed",
						"chunk_id", chunkIDs[i+j], "error", serr)
					failed++
				}
			}
			continue
		}

		for j, emb := range embeddings {
			if err := e.store.InsertEmbedding(ctx, chunkIDs[i+j], emb); err != nil {
				slog.Warn("storing embedding failed",
					"chunk_id", chunkIDs[i+j], "error", err)
				failed++
			}
		}
	}

	if failed == len(chunks) {
		return fmt.Errorf("all %d chunks failed embedding", len(chunks))
	}
	if failed > 0 {
		slog.Warn("some embeddings failed", "failed", failed, "total", len(chunks))
	}
	return nil
}

// Regex patterns for extracting technical identifiers from answer text.
// Mirrors the patterns in graph/builder.go for consistency.
var answerIdentifierPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:ISO|EN|IEC|MIL-STD|ASTM|IEEE|NIST|AS|BS)\s*[-]?\s*\d[\w.-]*`),
	regexp.MustCompile(`(?i)(?:PN[:\s]*|P/N[:\s]*)?[A-Z]{1,3}[-]?\d{3,6}`),
	regexp.MustCompile(`(?i)Rev\.?\s*[A-Z0-9]{1,5}`),
	regexp.MustCompile(`\b[A-Z]{2,4}-[A-Z]{1,4}\b`),
	regexp.MustCompile(`(?i)\d+(?:\.\d+)?\s*[Vv](?:AC|DC|ac|dc)?\b`),
	regexp.MustCompile(`(?i)IP\s*\d{2}\b`),                          // IP ratings like IP54
	regexp.MustCompile(`(?i)(?:UNE|NTP|ANSI|DIN|JIS|NF)\s*[-]?\s*\d[\w.-]*`), // additional standard prefixes
}

// falsePositivePrefixes filters out regex matches that are common in LLM
// prose but are not real technical identifiers.
var falsePositivePrefixes = []string{
	"figure ", "fig ", "table ", "step ", "page ", "section ",
	"chapter ", "item ", "part ", "ref ",
}

// isFalsePositiveIdentifier returns true if the matched string is likely
// a document cross-reference rather than a real technical identifier.
func isFalsePositiveIdentifier(ctx string, match string) bool {
	// Check if the match is preceded by a prose prefix in the surrounding text.
	idx := strings.Index(strings.ToLower(ctx), strings.ToLower(match))
	if idx <= 0 {
		return false
	}
	before := strings.ToLower(ctx[max(0, idx-10):idx])
	for _, p := range falsePositivePrefixes {
		if strings.HasSuffix(before, p) {
			return true
		}
	}
	return false
}

// extractMissingTerms finds technical identifiers in the answer text that do
// not appear in any of the retrieved chunks. These are candidates for targeted
// follow-up retrieval — they may be hallucinated or sourced from the LLM's
// prior knowledge, and finding supporting chunks improves answer grounding.
func extractMissingTerms(answer string, chunks []store.RetrievalResult) []string {
	// Build a single lowercase string of all retrieved content for fast lookup.
	var buf strings.Builder
	for _, c := range chunks {
		buf.WriteString(strings.ToLower(c.Content))
		buf.WriteByte(' ')
	}
	chunkContent := buf.String()

	seen := make(map[string]bool)
	var missing []string
	for _, p := range answerIdentifierPatterns {
		for _, m := range p.FindAllString(answer, -1) {
			key := strings.ToLower(strings.TrimSpace(m))
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			if isFalsePositiveIdentifier(answer, m) {
				continue
			}
			if !strings.Contains(chunkContent, key) {
				missing = append(missing, m)
			}
		}
	}
	return missing
}

// mergeResults appends extra retrieval results to the existing set,
// deduplicating by ChunkID. New results are appended at the end (lower
// priority than the original set).
func mergeResults(existing, extra []store.RetrievalResult) []store.RetrievalResult {
	seen := make(map[int64]bool, len(existing))
	for _, r := range existing {
		seen[r.ChunkID] = true
	}
	merged := make([]store.RetrievalResult, len(existing))
	copy(merged, existing)
	for _, r := range extra {
		if !seen[r.ChunkID] {
			seen[r.ChunkID] = true
			merged = append(merged, r)
		}
	}
	return merged
}

// fileHash computes the SHA-256 hash of a file's content.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
