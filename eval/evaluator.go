package eval

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/bbiangul/go-reason"
	"github.com/bbiangul/go-reason/llm"
	"github.com/bbiangul/go-reason/retrieval"
)

// Evaluator runs evaluation test sets against a GoReason engine.
type Evaluator struct {
	engine      goreason.Engine
	groundTruth map[string][]GroundTruthSpan // query -> spans (for retrieval P@k/R@k)
	judgeLLM    llm.Provider
	judgeModel  string
}

// NewEvaluator creates a new evaluator.
func NewEvaluator(engine goreason.Engine) *Evaluator {
	return &Evaluator{engine: engine}
}

// SetGroundTruth sets ground-truth spans for retrieval P@k/R@k computation.
// The map key is the query string.
func (e *Evaluator) SetGroundTruth(gt map[string][]GroundTruthSpan) {
	e.groundTruth = gt
}

// SetJudge configures an LLM judge for semantic accuracy evaluation.
// When set, accuracy is computed via LLM instead of verbatim substring matching.
func (e *Evaluator) SetJudge(provider llm.Provider, model string) {
	e.judgeLLM = provider
	e.judgeModel = model
}

// Report holds the results of an evaluation run.
type Report struct {
	Dataset         string                      `json:"dataset"`
	Difficulty      string                      `json:"difficulty,omitempty"`
	TotalTests      int                         `json:"total_tests"`
	Passed          int                         `json:"passed"`
	Failed          int                         `json:"failed"`
	Metrics         AggregateMetrics            `json:"metrics"`
	CategoryMetrics map[string]AggregateMetrics `json:"category_metrics,omitempty"`
	Results         []TestResult                `json:"results"`
	RunTime         time.Duration               `json:"run_time"`
	TokenUsage      TokenUsage                  `json:"token_usage"`
}

// TokenUsage aggregates LLM token consumption across an evaluation run.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AggregateMetrics holds averaged metrics across all tests.
type AggregateMetrics struct {
	AvgFaithfulness        float64 `json:"avg_faithfulness"`
	AvgRelevance           float64 `json:"avg_relevance"`
	AvgAccuracy            float64 `json:"avg_accuracy"`
	AvgStrictAccuracy      float64 `json:"avg_strict_accuracy"`
	AvgContextRecall       float64 `json:"avg_context_recall"`
	AvgCitationQuality     float64 `json:"avg_citation_quality"`
	AvgConfidence          float64 `json:"avg_confidence"`
	AvgClaimGrounding      float64 `json:"avg_claim_grounding"`
	AvgHallucinationScore  float64 `json:"avg_hallucination_score"`

	// Retrieval metrics (populated when ground-truth spans are available)
	AvgRetrievalPrecision map[int]float64 `json:"avg_retrieval_precision,omitempty"` // k -> P@k
	AvgRetrievalRecall    map[int]float64 `json:"avg_retrieval_recall,omitempty"`    // k -> R@k
}

// TestResult holds the result of a single test case with full diagnostics.
type TestResult struct {
	Question         string   `json:"question"`
	ExpectedFacts    []string `json:"expected_facts"`
	Category         string   `json:"category,omitempty"`
	Explanation      string   `json:"explanation,omitempty"`
	Answer           string   `json:"answer"`
	Confidence       float64  `json:"confidence"`
	Faithfulness     float64  `json:"faithfulness"`
	Relevance        float64  `json:"relevance"`
	Accuracy         float64  `json:"accuracy"`
	StrictAccuracy   float64  `json:"strict_accuracy"`
	ContextRecall    float64  `json:"context_recall"`
	CitationQuality    float64  `json:"citation_quality"`
	ClaimGrounding     float64  `json:"claim_grounding"`
	HallucinationScore float64  `json:"hallucination_score"`
	Passed             bool     `json:"passed"`
	Error            string   `json:"error,omitempty"`
	PromptTokens     int      `json:"prompt_tokens"`
	CompletionTokens int      `json:"completion_tokens"`
	TotalTokens      int      `json:"total_tokens"`

	// Timing
	ElapsedMs int64 `json:"elapsed_ms"`

	// Sources (the chunks the model actually saw)
	Sources []SourceTrace `json:"sources,omitempty"`

	// Retrieval breakdown
	Retrieval *RetrievalTrace `json:"retrieval,omitempty"`

	// Reasoning trace
	ReasoningSteps []ReasoningStep `json:"reasoning_steps,omitempty"`

	// Ground truth diagnosis
	GroundTruth *GroundTruthCheck `json:"ground_truth,omitempty"`

	// Retrieval metrics (populated when ground-truth spans are available)
	RetrievalPrecision map[int]float64 `json:"retrieval_precision,omitempty"` // k -> P@k
	RetrievalRecall    map[int]float64 `json:"retrieval_recall,omitempty"`    // k -> R@k
}

// SourceTrace records a single retrieved chunk with its retrieval metadata.
type SourceTrace struct {
	ChunkID    int64    `json:"chunk_id"`
	Heading    string   `json:"heading"`
	Content    string   `json:"content"`
	PageNumber int      `json:"page_number"`
	Score      float64  `json:"score"`
	Methods    []string `json:"methods,omitempty"`
	VecRank    int      `json:"vec_rank,omitempty"`
	FTSRank    int      `json:"fts_rank,omitempty"`
	GraphRank  int      `json:"graph_rank,omitempty"`
}

// RetrievalTrace holds the full retrieval breakdown for a query.
type RetrievalTrace struct {
	VecResults          int      `json:"vec_results"`
	FTSResults          int      `json:"fts_results"`
	GraphResults        int      `json:"graph_results"`
	FusedResults        int      `json:"fused_results"`
	VecWeight           float64  `json:"vec_weight"`
	FTSWeight           float64  `json:"fts_weight"`
	GraphWeight         float64  `json:"graph_weight"`
	IdentifiersDetected bool     `json:"identifiers_detected"`
	FTSQuery            string   `json:"fts_query"`
	GraphEntities       []string `json:"graph_entities"`
	ElapsedMs           int64    `json:"elapsed_ms"`
}

// ReasoningStep records a single round of reasoning with full context for replay.
type ReasoningStep struct {
	Round     int      `json:"round"`
	Action    string   `json:"action"`
	Prompt    string   `json:"prompt,omitempty"`
	Response  string   `json:"response,omitempty"`
	Tokens    int      `json:"tokens,omitempty"`
	ElapsedMs int64    `json:"elapsed_ms,omitempty"`
	Issues    []string `json:"issues,omitempty"`
}

// GroundTruthCheck diagnoses where each expected fact was lost in the pipeline.
type GroundTruthCheck struct {
	FactsInDB      []FactCheck `json:"facts_in_db"`
	FactsEmbedded  []FactCheck `json:"facts_embedded"`
	FactsRetrieved []FactCheck `json:"facts_retrieved"`
	FactsInAnswer  []FactCheck `json:"facts_in_answer"`
	Diagnosis      string      `json:"diagnosis"`
}

// FactCheck records whether a single expected fact was found at a pipeline stage.
type FactCheck struct {
	Fact      string `json:"fact"`
	Found     bool   `json:"found"`
	ChunkID   int64  `json:"chunk_id,omitempty"`
	ChunkRank int    `json:"chunk_rank,omitempty"`
	Details   string `json:"details,omitempty"`
}

// Run executes an evaluation dataset against the engine.
func (e *Evaluator) Run(ctx context.Context, dataset Dataset, opts ...goreason.QueryOption) (*Report, error) {
	start := time.Now()
	report := &Report{
		Dataset:         dataset.Name,
		Difficulty:      dataset.Difficulty,
		TotalTests:      len(dataset.Tests),
		CategoryMetrics: make(map[string]AggregateMetrics),
	}

	// Track per-category accumulators
	catCounts := make(map[string]int)
	catSums := make(map[string]AggregateMetrics)
	metricsCount := 0

	// Retrieval metric accumulators
	retPrecisionSums := make(map[int]float64)
	retRecallSums := make(map[int]float64)
	retMetricsCount := 0

	for i, test := range dataset.Tests {
		result := e.runTest(ctx, test, opts...)
		report.Results = append(report.Results, result)

		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		if result.Error != "" {
			status = "ERROR"
		}

		diagStr := ""
		if result.GroundTruth != nil {
			diagStr = result.GroundTruth.Diagnosis
		}

		slog.Info("eval: test complete",
			"progress", fmt.Sprintf("%d/%d", i+1, len(dataset.Tests)),
			"status", status,
			"diagnosis", diagStr,
			"confidence", fmt.Sprintf("%.2f", result.Confidence),
			"accuracy", fmt.Sprintf("%.2f", result.Accuracy),
			"tokens", result.TotalTokens,
			"elapsed_ms", result.ElapsedMs,
			"question", truncate(test.Question, 80))

		// Accumulate token usage regardless of pass/fail/error
		report.TokenUsage.PromptTokens += result.PromptTokens
		report.TokenUsage.CompletionTokens += result.CompletionTokens
		report.TokenUsage.TotalTokens += result.TotalTokens

		if result.Passed {
			report.Passed++
		} else {
			report.Failed++
		}

		// Exclude error results from metric averages — they contribute
		// all zeros which would artificially depress the scores.
		if result.Error != "" {
			continue
		}

		metricsCount++
		report.Metrics.AvgFaithfulness += result.Faithfulness
		report.Metrics.AvgRelevance += result.Relevance
		report.Metrics.AvgAccuracy += result.Accuracy
		report.Metrics.AvgStrictAccuracy += result.StrictAccuracy
		report.Metrics.AvgContextRecall += result.ContextRecall
		report.Metrics.AvgCitationQuality += result.CitationQuality
		report.Metrics.AvgConfidence += result.Confidence
		report.Metrics.AvgClaimGrounding += result.ClaimGrounding
		report.Metrics.AvgHallucinationScore += result.HallucinationScore

		// Accumulate retrieval metrics
		if result.RetrievalPrecision != nil {
			retMetricsCount++
			for _, k := range RetrievalKValues {
				retPrecisionSums[k] += result.RetrievalPrecision[k]
				retRecallSums[k] += result.RetrievalRecall[k]
			}
		}

		// Per-category accumulation
		if test.Category != "" {
			catCounts[test.Category]++
			sum := catSums[test.Category]
			sum.AvgFaithfulness += result.Faithfulness
			sum.AvgRelevance += result.Relevance
			sum.AvgAccuracy += result.Accuracy
			sum.AvgStrictAccuracy += result.StrictAccuracy
			sum.AvgContextRecall += result.ContextRecall
			sum.AvgCitationQuality += result.CitationQuality
			sum.AvgConfidence += result.Confidence
			sum.AvgClaimGrounding += result.ClaimGrounding
			sum.AvgHallucinationScore += result.HallucinationScore
			catSums[test.Category] = sum
		}
	}

	n := float64(metricsCount)
	if n > 0 {
		report.Metrics.AvgFaithfulness /= n
		report.Metrics.AvgRelevance /= n
		report.Metrics.AvgAccuracy /= n
		report.Metrics.AvgStrictAccuracy /= n
		report.Metrics.AvgContextRecall /= n
		report.Metrics.AvgCitationQuality /= n
		report.Metrics.AvgConfidence /= n
		report.Metrics.AvgClaimGrounding /= n
		report.Metrics.AvgHallucinationScore /= n
	}

	// Compute retrieval metric averages
	if retMetricsCount > 0 {
		rn := float64(retMetricsCount)
		report.Metrics.AvgRetrievalPrecision = make(map[int]float64)
		report.Metrics.AvgRetrievalRecall = make(map[int]float64)
		for _, k := range RetrievalKValues {
			report.Metrics.AvgRetrievalPrecision[k] = retPrecisionSums[k] / rn
			report.Metrics.AvgRetrievalRecall[k] = retRecallSums[k] / rn
		}
	}

	// Compute per-category averages
	for cat, count := range catCounts {
		cn := float64(count)
		sum := catSums[cat]
		report.CategoryMetrics[cat] = AggregateMetrics{
			AvgFaithfulness:       sum.AvgFaithfulness / cn,
			AvgRelevance:          sum.AvgRelevance / cn,
			AvgAccuracy:           sum.AvgAccuracy / cn,
			AvgStrictAccuracy:     sum.AvgStrictAccuracy / cn,
			AvgContextRecall:      sum.AvgContextRecall / cn,
			AvgCitationQuality:    sum.AvgCitationQuality / cn,
			AvgConfidence:         sum.AvgConfidence / cn,
			AvgClaimGrounding:     sum.AvgClaimGrounding / cn,
			AvgHallucinationScore: sum.AvgHallucinationScore / cn,
		}
	}

	report.RunTime = time.Since(start)
	return report, nil
}

func (e *Evaluator) runTest(ctx context.Context, test TestCase, opts ...goreason.QueryOption) TestResult {
	testStart := time.Now()
	result := TestResult{
		Question:      test.Question,
		ExpectedFacts: test.ExpectedFacts,
		Category:      test.Category,
		Explanation:   test.Explanation,
	}

	answer, err := e.engine.Query(ctx, test.Question, opts...)
	if err != nil {
		result.Error = err.Error()
		result.ElapsedMs = time.Since(testStart).Milliseconds()
		return result
	}

	result.Answer = answer.Text
	result.Confidence = answer.Confidence
	result.PromptTokens = answer.PromptTokens
	result.CompletionTokens = answer.CompletionTokens
	result.TotalTokens = answer.TotalTokens

	// Compute metrics
	result.Faithfulness = computeFaithfulness(answer)
	result.Relevance = computeRelevance(answer, test.Question)

	// Always compute strict (verbatim) accuracy
	strictAcc := computeAccuracy(answer, test.ExpectedFacts)
	result.StrictAccuracy = strictAcc
	result.Accuracy = strictAcc

	// If judge is configured, use LLM-based accuracy instead
	if e.judgeLLM != nil {
		llmAcc, err := computeAccuracyLLM(ctx, e.judgeLLM, e.judgeModel, answer, test.ExpectedFacts)
		if err != nil {
			slog.Warn("judge LLM failed, falling back to strict accuracy",
				"error", err,
				"question", truncate(test.Question, 60))
		} else {
			result.Accuracy = llmAcc
		}
	}

	result.ContextRecall = computeContextRecall(answer, test.ExpectedFacts)
	result.CitationQuality = computeCitationQuality(answer)
	result.ClaimGrounding = computeClaimGrounding(answer)
	result.HallucinationScore = computeHallucinationScore(answer)

	// A test passes if:
	// 1. The engine retrieved chunks containing the evidence (ContextRecall >= 0.5)
	// 2. The model extracted the facts from those chunks (Accuracy >= 0.5)
	// This replaces the weak Faithfulness gate with direct retrieval quality measurement.
	result.Passed = result.Accuracy >= 0.5 && result.ContextRecall >= 0.5

	// Build source traces from answer
	result.Sources = buildSourceTraces(answer)

	// Build retrieval trace from answer
	if answer.RetrievalTrace != nil {
		result.Retrieval = buildRetrievalTrace(answer.RetrievalTrace)
	}

	// Build reasoning steps from answer
	result.ReasoningSteps = buildReasoningSteps(answer)

	// Run ground truth diagnosis
	result.GroundTruth = e.runGroundTruthCheck(ctx, test, answer)

	// Compute retrieval P@k/R@k if ground-truth spans are available
	if spans, ok := e.groundTruth[test.Question]; ok && len(spans) > 0 {
		result.RetrievalPrecision = make(map[int]float64)
		result.RetrievalRecall = make(map[int]float64)
		for _, k := range RetrievalKValues {
			result.RetrievalPrecision[k] = computeRetrievalPrecisionAtK(answer, spans, k)
			result.RetrievalRecall[k] = computeRetrievalRecallAtK(answer, spans, k)
		}
	}

	result.ElapsedMs = time.Since(testStart).Milliseconds()

	return result
}

func buildSourceTraces(answer *goreason.Answer) []SourceTrace {
	if answer == nil {
		return nil
	}
	traces := make([]SourceTrace, len(answer.Sources))
	for i, src := range answer.Sources {
		st := SourceTrace{
			ChunkID:    src.ChunkID,
			Heading:    src.Heading,
			Content:    src.Content,
			PageNumber: src.PageNumber,
			Score:      src.Score,
		}
		// Attach per-result method info from the retrieval trace
		if answer.RetrievalTrace != nil && answer.RetrievalTrace.PerResult != nil {
			if info, ok := answer.RetrievalTrace.PerResult[src.ChunkID]; ok {
				st.Methods = info.Methods
				st.VecRank = info.VecRank
				st.FTSRank = info.FTSRank
				st.GraphRank = info.GraphRank
			}
		}
		traces[i] = st
	}
	return traces
}

func buildRetrievalTrace(st *retrieval.SearchTrace) *RetrievalTrace {
	return &RetrievalTrace{
		VecResults:          st.VecResults,
		FTSResults:          st.FTSResults,
		GraphResults:        st.GraphResults,
		FusedResults:        st.FusedResults,
		VecWeight:           st.VecWeight,
		FTSWeight:           st.FTSWeight,
		GraphWeight:         st.GraphWeight,
		IdentifiersDetected: st.IdentifiersDetected,
		FTSQuery:            st.FTSQuery,
		GraphEntities:       st.GraphEntities,
		ElapsedMs:           st.ElapsedMs,
	}
}

func buildReasoningSteps(answer *goreason.Answer) []ReasoningStep {
	if answer == nil {
		return nil
	}
	steps := make([]ReasoningStep, len(answer.Reasoning))
	for i, s := range answer.Reasoning {
		steps[i] = ReasoningStep{
			Round:     s.Round,
			Action:    s.Action,
			Prompt:    s.Prompt,
			Response:  s.Response,
			Tokens:    s.Tokens,
			ElapsedMs: s.ElapsedMs,
			Issues:    s.Issues,
		}
	}
	return steps
}

// runGroundTruthCheck diagnoses why each expected fact was or wasn't in the final answer.
// Pipeline stages: chunk DB → embedding → retrieval top-N → model answer
func (e *Evaluator) runGroundTruthCheck(ctx context.Context, test TestCase, answer *goreason.Answer) *GroundTruthCheck {
	s := e.engine.Store()
	if s == nil {
		return nil
	}

	gt := &GroundTruthCheck{}

	// Collect chunk IDs that were retrieved (for rank lookup)
	retrievedChunks := make(map[int64]int) // chunkID -> rank (1-based)
	if answer != nil {
		for i, src := range answer.Sources {
			retrievedChunks[src.ChunkID] = i + 1
		}
	}

	answerLower := ""
	answerSpaceless := ""
	answerHyphenless := ""
	if answer != nil {
		answerLower = normalizeLLMText(strings.ToLower(answer.Text))
		answerSpaceless = strings.ReplaceAll(answerLower, " ", "")
		answerHyphenless = strings.ReplaceAll(strings.ReplaceAll(answerLower, "-", ""), " ", "")
	}

	// Track the worst failure stage across all facts
	worstStage := "PASS"

	for _, fact := range test.ExpectedFacts {
		// Each fact may have pipe-separated alternatives
		alternatives := strings.Split(fact, "|")

		// Stage 1: Is the fact in any chunk in the DB?
		dbCheck := FactCheck{Fact: fact}
		for _, alt := range alternatives {
			alt = strings.TrimSpace(alt)
			if alt == "" {
				continue
			}
			matches, err := s.SearchChunksByContent(ctx, alt)
			if err != nil {
				dbCheck.Details = fmt.Sprintf("search error: %v", err)
				continue
			}
			if len(matches) > 0 {
				dbCheck.Found = true
				dbCheck.ChunkID = matches[0].ChunkID
				dbCheck.Details = fmt.Sprintf("found in %d chunk(s), first: heading=%q page=%d",
					len(matches), matches[0].Heading, matches[0].PageNumber)
				break
			}
		}
		gt.FactsInDB = append(gt.FactsInDB, dbCheck)

		// Stage 2: Does that chunk have an embedding?
		embCheck := FactCheck{Fact: fact}
		if dbCheck.Found {
			hasEmb, err := s.ChunkHasEmbedding(ctx, dbCheck.ChunkID)
			if err != nil {
				embCheck.Details = fmt.Sprintf("check error: %v", err)
			} else {
				embCheck.Found = hasEmb
				embCheck.ChunkID = dbCheck.ChunkID
				if !hasEmb {
					embCheck.Details = "chunk exists but has no embedding"
				}
			}
		}
		gt.FactsEmbedded = append(gt.FactsEmbedded, embCheck)

		// Stage 3: Was the chunk in the retrieved results?
		retCheck := FactCheck{Fact: fact}
		if dbCheck.Found {
			if rank, ok := retrievedChunks[dbCheck.ChunkID]; ok {
				retCheck.Found = true
				retCheck.ChunkID = dbCheck.ChunkID
				retCheck.ChunkRank = rank
			} else {
				retCheck.Details = "chunk not in top retrieval results"
			}
		}
		gt.FactsRetrieved = append(gt.FactsRetrieved, retCheck)

		// Stage 4: Does the fact appear in the model's answer?
		ansCheck := FactCheck{Fact: fact}
		for _, alt := range alternatives {
			alt = strings.TrimSpace(alt)
			if alt == "" {
				continue
			}
			normAlt := normalizeLLMText(strings.ToLower(alt))
			normAltNoSpace := strings.ReplaceAll(normAlt, " ", "")
			normAltNoHyphen := strings.ReplaceAll(strings.ReplaceAll(normAlt, "-", ""), " ", "")
			if strings.Contains(answerLower, normAlt) ||
				strings.Contains(answerSpaceless, normAltNoSpace) ||
				strings.Contains(answerHyphenless, normAltNoHyphen) {
				ansCheck.Found = true
				break
			}
		}
		gt.FactsInAnswer = append(gt.FactsInAnswer, ansCheck)

		// Determine worst failure stage for this fact
		if !ansCheck.Found {
			if !dbCheck.Found {
				worstStage = worseStage(worstStage, "CHUNK_MISS")
			} else if !embCheck.Found {
				worstStage = worseStage(worstStage, "EMBEDDING_MISS")
			} else if !retCheck.Found {
				worstStage = worseStage(worstStage, "RETRIEVAL_MISS")
			} else {
				worstStage = worseStage(worstStage, "MODEL_MISS")
			}
		}
	}

	gt.Diagnosis = worstStage
	return gt
}

// stageSeverity returns a severity rank for diagnosis stages (higher = worse).
func stageSeverity(stage string) int {
	switch stage {
	case "PASS":
		return 0
	case "MODEL_MISS":
		return 1
	case "RETRIEVAL_MISS":
		return 2
	case "EMBEDDING_MISS":
		return 3
	case "CHUNK_MISS":
		return 4
	default:
		return 0
	}
}

func worseStage(a, b string) string {
	if stageSeverity(b) > stageSeverity(a) {
		return b
	}
	return a
}

// FormatReport produces a human-readable report string.
func FormatReport(r *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== Evaluation Report: %s ===\n", r.Dataset)
	if r.Difficulty != "" {
		fmt.Fprintf(&b, "Difficulty: %s\n", r.Difficulty)
	}
	fmt.Fprintf(&b, "Total: %d | Passed: %d (%.1f%%) | Failed: %d\n",
		r.TotalTests, r.Passed, passRate(r.Passed, r.TotalTests), r.Failed)
	fmt.Fprintf(&b, "Run time: %s\n\n", r.RunTime.Round(time.Millisecond))

	fmt.Fprintf(&b, "Aggregate Metrics:\n")
	fmt.Fprintf(&b, "  Faithfulness:         %.2f\n", r.Metrics.AvgFaithfulness)
	fmt.Fprintf(&b, "  Relevance:            %.2f\n", r.Metrics.AvgRelevance)
	fmt.Fprintf(&b, "  Accuracy:             %.2f\n", r.Metrics.AvgAccuracy)
	if r.Metrics.AvgStrictAccuracy != r.Metrics.AvgAccuracy {
		fmt.Fprintf(&b, "  Strict Accuracy:      %.2f\n", r.Metrics.AvgStrictAccuracy)
	}
	fmt.Fprintf(&b, "  Context Recall:       %.2f\n", r.Metrics.AvgContextRecall)
	fmt.Fprintf(&b, "  Citation Quality:     %.2f\n", r.Metrics.AvgCitationQuality)
	fmt.Fprintf(&b, "  Claim Grounding:      %.2f\n", r.Metrics.AvgClaimGrounding)
	fmt.Fprintf(&b, "  Hallucination Score:  %.2f\n", r.Metrics.AvgHallucinationScore)
	fmt.Fprintf(&b, "  Confidence:           %.2f\n\n", r.Metrics.AvgConfidence)

	// Retrieval metrics (if available)
	if len(r.Metrics.AvgRetrievalPrecision) > 0 {
		fmt.Fprintf(&b, "Retrieval Metrics:\n")
		for _, k := range RetrievalKValues {
			if p, ok := r.Metrics.AvgRetrievalPrecision[k]; ok {
				fmt.Fprintf(&b, "  P@%-3d  %.1f%%\n", k, p*100)
			}
		}
		for _, k := range RetrievalKValues {
			if recall, ok := r.Metrics.AvgRetrievalRecall[k]; ok {
				fmt.Fprintf(&b, "  R@%-3d  %.1f%%\n", k, recall*100)
			}
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "Token Usage:\n")
	fmt.Fprintf(&b, "  Prompt:     %d\n", r.TokenUsage.PromptTokens)
	fmt.Fprintf(&b, "  Completion: %d\n", r.TokenUsage.CompletionTokens)
	fmt.Fprintf(&b, "  Total:      %d\n\n", r.TokenUsage.TotalTokens)

	// Per-category breakdown (sorted for deterministic output)
	if len(r.CategoryMetrics) > 0 {
		cats := make([]string, 0, len(r.CategoryMetrics))
		for cat := range r.CategoryMetrics {
			cats = append(cats, cat)
		}
		sort.Strings(cats)

		fmt.Fprintf(&b, "Per-Category Metrics:\n")
		for _, cat := range cats {
			m := r.CategoryMetrics[cat]
			fmt.Fprintf(&b, "  [%s]\n", cat)
			fmt.Fprintf(&b, "    Faith=%.2f Rel=%.2f Acc=%.2f CtxR=%.2f Cite=%.2f Grnd=%.2f Hall=%.2f Conf=%.2f\n",
				m.AvgFaithfulness, m.AvgRelevance, m.AvgAccuracy, m.AvgContextRecall, m.AvgCitationQuality,
				m.AvgClaimGrounding, m.AvgHallucinationScore, m.AvgConfidence)
		}
		fmt.Fprintln(&b)
	}

	for i, res := range r.Results {
		status := "PASS"
		if !res.Passed {
			status = "FAIL"
		}
		diag := ""
		if res.GroundTruth != nil && res.GroundTruth.Diagnosis != "PASS" {
			diag = fmt.Sprintf(" [%s]", res.GroundTruth.Diagnosis)
		}
		fmt.Fprintf(&b, "[%s]%s %d. %s\n", status, diag, i+1, res.Question)
		if res.Error != "" {
			fmt.Fprintf(&b, "  Error: %s\n", res.Error)
		} else {
			fmt.Fprintf(&b, "  Faith=%.2f Rel=%.2f Acc=%.2f CtxR=%.2f Cite=%.2f Grnd=%.2f Hall=%.2f Conf=%.2f  (%dms)\n",
				res.Faithfulness, res.Relevance, res.Accuracy, res.ContextRecall, res.CitationQuality,
				res.ClaimGrounding, res.HallucinationScore, res.Confidence, res.ElapsedMs)
			if res.StrictAccuracy != res.Accuracy {
				fmt.Fprintf(&b, "  StrictAcc=%.2f\n", res.StrictAccuracy)
			}
		}
	}

	return b.String()
}

func passRate(passed, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(passed) / float64(total) * 100
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
