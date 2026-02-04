package retrieval

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/bbiangul/go-reason/llm"
	"github.com/bbiangul/go-reason/store"
)

// ---------------------------------------------------------------------------
// Identifier detection for query routing.
// When a query contains structured identifiers (part numbers, standards, IP
// addresses, etc.) we boost FTS weight and reduce vector weight so that
// exact-match retrieval is preferred over semantic similarity.
// ---------------------------------------------------------------------------
var identifierPatterns = []*regexp.Regexp{
	// Part numbers: E1375, E-1306, PN: XXXXX, P/N XXXXX
	regexp.MustCompile(`(?i)(?:PN[:\s]*|P/N[:\s]*)?[A-Z]{1,3}[-]?\d{3,6}`),
	// Standards: ISO XXXXX, EN XXXXX, IEC XXXXX, MIL-STD-XXX, ASTM, IEEE, NIST
	regexp.MustCompile(`(?i)(?:ISO|EN|IEC|MIL-STD|ASTM|IEEE|NIST|AS|BS)\s*[-]?\s*\d[\w.-]*`),
	// IP addresses: XXX.XXX.XXX.XXX
	regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
	// Model numbers: AV-FM, AV-FF, AV-L
	regexp.MustCompile(`\b[A-Z]{2,4}-[A-Z]{1,4}\b`),
	// Revision codes: RevG02, Rev2, Rev.A
	regexp.MustCompile(`(?i)Rev\.?\s*[A-Z0-9]{1,5}`),
	// Voltage/current specs: 120VAC, 24VDC
	regexp.MustCompile(`(?i)\d+(?:\.\d+)?\s*[Vv](?:AC|DC|ac|dc)\b`),
}

// detectIdentifiers returns true if the query contains at least one
// structured identifier (part number, standard, IP, model number, etc.).
func detectIdentifiers(query string) bool {
	for _, p := range identifierPatterns {
		if p.MatchString(query) {
			return true
		}
	}
	return false
}

// Config holds retrieval engine configuration.
type Config struct {
	WeightVector float64
	WeightFTS    float64
	WeightGraph  float64
}

// SearchOptions configures a single search operation.
type SearchOptions struct {
	MaxResults  int
	WeightVec   float64
	WeightFTS   float64
	WeightGraph float64
}

// SearchTrace records the full breakdown of a hybrid search operation.
type SearchTrace struct {
	VecResults          int                `json:"vec_results"`
	FTSResults          int                `json:"fts_results"`
	GraphResults        int                `json:"graph_results"`
	FusedResults        int                `json:"fused_results"`
	VecWeight           float64            `json:"vec_weight"`
	FTSWeight           float64            `json:"fts_weight"`
	GraphWeight         float64            `json:"graph_weight"`
	IdentifiersDetected bool               `json:"identifiers_detected"`
	SynthesisMode       bool               `json:"synthesis_mode"`
	MaxRequested        int                `json:"max_requested"`
	FollowUpTerms       []string           `json:"follow_up_terms,omitempty"`
	FollowUpResults     int                `json:"follow_up_results,omitempty"`
	FTSQuery            string             `json:"fts_query"`
	GraphEntities       []string           `json:"graph_entities"`
	ElapsedMs           int64              `json:"elapsed_ms"`
	PerResult           map[int64]FusedResultInfo `json:"per_result,omitempty"`
}

// Engine performs hybrid retrieval combining vector, FTS, and graph search.
type Engine struct {
	store      *store.Store
	embedder   llm.Provider
	translator *Translator
	cfg        Config
}

// New creates a new retrieval engine. chatLLM is used for cross-language
// query translation; pass nil to disable translation.
func New(s *store.Store, embedder llm.Provider, chatLLM llm.Provider, cfg Config) *Engine {
	return &Engine{
		store:      s,
		embedder:   embedder,
		translator: NewTranslator(chatLLM, s),
		cfg:        cfg,
	}
}

// Search performs hybrid retrieval using RRF to fuse results from
// vector search, FTS5, and graph-based retrieval.
// Returns fused results and a SearchTrace with the full breakdown.
func (e *Engine) Search(ctx context.Context, query string, opts SearchOptions) ([]store.RetrievalResult, *SearchTrace, error) {
	if opts.MaxResults == 0 {
		opts.MaxResults = 20
	}
	if opts.WeightVec == 0 {
		opts.WeightVec = e.cfg.WeightVector
	}
	if opts.WeightFTS == 0 {
		opts.WeightFTS = e.cfg.WeightFTS
	}
	if opts.WeightGraph == 0 {
		opts.WeightGraph = e.cfg.WeightGraph
	}

	trace := &SearchTrace{
		VecWeight:   opts.WeightVec,
		FTSWeight:   opts.WeightFTS,
		GraphWeight: opts.WeightGraph,
	}

	// Identifier-aware query routing: when the query contains structured
	// identifiers (part numbers, standards, IPs, model numbers, etc.),
	// boost FTS weight by 2x and reduce vector weight by 0.5x so that
	// exact-match retrieval is preferred over semantic similarity.
	if detectIdentifiers(query) {
		slog.Debug("retrieval: identifiers detected in query, boosting FTS weight",
			"query", query,
			"original_fts", opts.WeightFTS,
			"original_vec", opts.WeightVec)
		opts.WeightFTS *= 2.0
		opts.WeightVec *= 0.5
		trace.IdentifiersDetected = true
		trace.VecWeight = opts.WeightVec
		trace.FTSWeight = opts.WeightFTS
	}

	// Synthesis query detection: widen retrieval window for exhaustive queries
	synthesisMode := isSynthesisQuery(query)
	if synthesisMode {
		if opts.MaxResults < 40 {
			opts.MaxResults = 40
		}
		trace.SynthesisMode = true
		slog.Debug("retrieval: synthesis mode activated, widened retrieval window",
			"query", query, "max_results", opts.MaxResults)
	}

	// Run all three retrieval methods concurrently
	slog.Debug("retrieval: starting hybrid search",
		"query_len", len(query), "max_results", opts.MaxResults,
		"weights", fmt.Sprintf("vec=%.1f fts=%.1f graph=%.1f", opts.WeightVec, opts.WeightFTS, opts.WeightGraph))
	searchStart := time.Now()

	// Cross-language expansion: translate significant query terms to
	// the document language so FTS and graph search can match content
	// written in a different language than the query.
	translated := e.translator.TranslateTerms(ctx, extractSignificantTerms(query))

	// Capture FTS query for trace
	ftsQuery := sanitizeFTSQuery(query, translated)
	trace.FTSQuery = ftsQuery

	// Capture graph entities for trace
	graphEntities := extractQueryEntities(query, translated)
	trace.GraphEntities = graphEntities

	type result struct {
		results []store.RetrievalResult
		err     error
	}

	vecCh := make(chan result, 1)
	ftsCh := make(chan result, 1)
	graphCh := make(chan result, 1)

	// Vector search
	go func() {
		r, err := e.vectorSearch(ctx, query, opts.MaxResults)
		vecCh <- result{r, err}
	}()

	// FTS search
	go func() {
		r, err := e.store.FTSSearch(ctx, ftsQuery, opts.MaxResults)
		ftsCh <- result{r, err}
	}()

	// Graph search
	go func() {
		r, err := e.graphSearchWithEntities(ctx, graphEntities, opts.MaxResults, synthesisMode)
		graphCh <- result{r, err}
	}()

	vecRes := <-vecCh
	ftsRes := <-ftsCh
	graphRes := <-graphCh

	if vecRes.err != nil {
		slog.Warn("retrieval: vector search failed", "error", vecRes.err)
	}
	trace.VecResults = len(vecRes.results)
	trace.FTSResults = len(ftsRes.results)
	trace.GraphResults = len(graphRes.results)

	slog.Debug("retrieval: searches complete",
		"vec_results", len(vecRes.results), "fts_results", len(ftsRes.results),
		"graph_results", len(graphRes.results),
		"elapsed", time.Since(searchStart).Round(time.Millisecond))

	// Fuse results with RRF
	fused, infoMap := fuseRRF(
		vecRes.results, ftsRes.results, graphRes.results,
		opts.WeightVec, opts.WeightFTS, opts.WeightGraph,
		opts.MaxResults,
	)

	trace.FusedResults = len(fused)
	trace.MaxRequested = opts.MaxResults
	trace.PerResult = infoMap
	trace.ElapsedMs = time.Since(searchStart).Milliseconds()

	if len(fused) == 0 {
		// If all methods failed, return the first error
		if vecRes.err != nil {
			return nil, trace, fmt.Errorf("vector search: %w", vecRes.err)
		}
		if ftsRes.err != nil {
			return nil, trace, fmt.Errorf("fts search: %w", ftsRes.err)
		}
		if graphRes.err != nil {
			return nil, trace, fmt.Errorf("graph search: %w", graphRes.err)
		}
	}

	return fused, trace, nil
}

// vectorSearch generates an embedding for the query and searches vec_chunks.
func (e *Engine) vectorSearch(ctx context.Context, query string, k int) ([]store.RetrievalResult, error) {
	embeddings, err := e.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}
	return e.store.VectorSearch(ctx, embeddings[0], k)
}

// ftsSearch performs FTS5 full-text search.
func (e *Engine) ftsSearch(ctx context.Context, query string, translated []string, limit int) ([]store.RetrievalResult, error) {
	ftsQuery := sanitizeFTSQuery(query, translated)
	return e.store.FTSSearch(ctx, ftsQuery, limit)
}

// graphSearch extracts entities from the query and traverses the graph.
func (e *Engine) graphSearch(ctx context.Context, query string, translated []string, limit int) ([]store.RetrievalResult, error) {
	entities := extractQueryEntities(query, translated)
	return e.graphSearchWithEntities(ctx, entities, limit, false)
}

// graphSearchWithEntities traverses the graph using pre-extracted entity names.
// Uses both exact and substring matching: exact match first (fast), then
// substring match (broader) to find multi-word entity names containing the
// query terms. This is critical for cross-language queries where single-word
// English/Spanish terms need to match multi-word entity names like
// "rechazador de envases" from a query containing "rejected"/"rechazado".
//
// When synthesisMode is true, performs an additional 1-hop relationship
// expansion to discover entities connected to the initial matches but not
// directly matched by name. This helps synthesis queries find scattered facts.
func (e *Engine) graphSearchWithEntities(ctx context.Context, entities []string, limit int, synthesisMode bool) ([]store.RetrievalResult, error) {
	if len(entities) == 0 {
		return nil, nil
	}

	// Normalize to lowercase to match storage format (graph builder lowercases all entity names)
	for i, ent := range entities {
		entities[i] = strings.ToLower(ent)
	}

	// Try exact match first
	found, err := e.store.GetEntitiesByNames(ctx, entities)
	if err != nil {
		return nil, err
	}

	// Also do substring match to find multi-word entities containing query terms
	fuzzyFound, err := e.store.SearchEntitiesByTerms(ctx, entities, 50)
	if err != nil {
		slog.Warn("retrieval: fuzzy entity search failed", "error", err)
	}

	// Also search by English canonical name for cross-language entity matching
	enFound, err := e.store.SearchEntitiesByNameEN(ctx, entities, 50)
	if err != nil {
		slog.Warn("retrieval: name_en entity search failed", "error", err)
	}

	// Merge results (deduplicate by ID)
	seen := make(map[int64]bool)
	var allEntities []store.Entity
	for _, e := range found {
		if !seen[e.ID] {
			seen[e.ID] = true
			allEntities = append(allEntities, e)
		}
	}
	for _, e := range fuzzyFound {
		if !seen[e.ID] {
			seen[e.ID] = true
			allEntities = append(allEntities, e)
		}
	}
	for _, e := range enFound {
		if !seen[e.ID] {
			seen[e.ID] = true
			allEntities = append(allEntities, e)
		}
	}

	if len(allEntities) == 0 {
		return nil, nil
	}

	slog.Debug("retrieval: graph entity lookup",
		"exact_matches", len(found), "fuzzy_matches", len(fuzzyFound),
		"name_en_matches", len(enFound), "total_unique", len(allEntities))

	entityIDs := make([]int64, len(allEntities))
	for i, e := range allEntities {
		entityIDs[i] = e.ID
	}

	// 1-hop relationship expansion for synthesis queries: discover entities
	// connected to the seed set (e.g., "seguridad y normativa" → "ip54").
	if synthesisMode {
		neighborEntities, err := e.store.GetRelatedEntities(ctx, entityIDs, 100)
		if err != nil {
			slog.Warn("retrieval: 1-hop entity expansion failed", "error", err)
		} else if len(neighborEntities) > 0 {
			added := 0
			for _, ne := range neighborEntities {
				if !seen[ne.ID] {
					seen[ne.ID] = true
					allEntities = append(allEntities, ne)
					entityIDs = append(entityIDs, ne.ID)
					added++
				}
			}
			slog.Debug("retrieval: 1-hop expansion",
				"returned", len(neighborEntities), "new", added, "total_unique", len(allEntities))
		}
	}

	return e.store.GraphSearch(ctx, entityIDs, limit)
}
