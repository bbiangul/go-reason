package eval

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bbiangul/go-reason"
	"github.com/bbiangul/go-reason/llm"
)

// FullContextEvaluator sends the entire document text + question directly to an
// LLM provider, bypassing RAG entirely. This serves as a baseline to compare
// against Graph RAG and Basic RAG approaches.
type FullContextEvaluator struct {
	provider llm.Provider
	docText  string // entire PDF text preloaded
}

// NewFullContextEvaluator creates a full-context evaluator.
// The docText should contain the entire document content (e.g. extracted PDF text).
func NewFullContextEvaluator(provider llm.Provider, docText string) *FullContextEvaluator {
	return &FullContextEvaluator{
		provider: provider,
		docText:  docText,
	}
}

// Run executes an evaluation dataset by sending the full document text + each
// question to the LLM. It produces a Report with the same metric structure as
// the engine-based evaluator so results are directly comparable.
func (e *FullContextEvaluator) Run(ctx context.Context, dataset Dataset) (*Report, error) {
	start := time.Now()
	report := &Report{
		Dataset:         dataset.Name + " (full-context)",
		Difficulty:      dataset.Difficulty,
		TotalTests:      len(dataset.Tests),
		CategoryMetrics: make(map[string]AggregateMetrics),
	}

	catCounts := make(map[string]int)
	catSums := make(map[string]AggregateMetrics)
	metricsCount := 0

	for i, test := range dataset.Tests {
		result := e.runTest(ctx, test)
		report.Results = append(report.Results, result)

		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		if result.Error != "" {
			status = "ERROR"
		}

		slog.Info("eval[full-context]: test complete",
			"progress", fmt.Sprintf("%d/%d", i+1, len(dataset.Tests)),
			"status", status,
			"accuracy", fmt.Sprintf("%.2f", result.Accuracy),
			"tokens", result.TotalTokens,
			"elapsed_ms", result.ElapsedMs,
			"question", truncate(test.Question, 80))

		report.TokenUsage.PromptTokens += result.PromptTokens
		report.TokenUsage.CompletionTokens += result.CompletionTokens
		report.TokenUsage.TotalTokens += result.TotalTokens

		if result.Passed {
			report.Passed++
		} else {
			report.Failed++
		}

		if result.Error != "" {
			continue
		}

		metricsCount++
		report.Metrics.AvgFaithfulness += result.Faithfulness
		report.Metrics.AvgRelevance += result.Relevance
		report.Metrics.AvgAccuracy += result.Accuracy
		report.Metrics.AvgContextRecall += result.ContextRecall
		report.Metrics.AvgCitationQuality += result.CitationQuality
		report.Metrics.AvgConfidence += result.Confidence
		report.Metrics.AvgClaimGrounding += result.ClaimGrounding
		report.Metrics.AvgHallucinationScore += result.HallucinationScore

		if test.Category != "" {
			catCounts[test.Category]++
			sum := catSums[test.Category]
			sum.AvgFaithfulness += result.Faithfulness
			sum.AvgRelevance += result.Relevance
			sum.AvgAccuracy += result.Accuracy
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
		report.Metrics.AvgContextRecall /= n
		report.Metrics.AvgCitationQuality /= n
		report.Metrics.AvgConfidence /= n
		report.Metrics.AvgClaimGrounding /= n
		report.Metrics.AvgHallucinationScore /= n
	}

	for cat, count := range catCounts {
		cn := float64(count)
		sum := catSums[cat]
		report.CategoryMetrics[cat] = AggregateMetrics{
			AvgFaithfulness:       sum.AvgFaithfulness / cn,
			AvgRelevance:          sum.AvgRelevance / cn,
			AvgAccuracy:           sum.AvgAccuracy / cn,
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

func (e *FullContextEvaluator) runTest(ctx context.Context, test TestCase) TestResult {
	testStart := time.Now()
	result := TestResult{
		Question:      test.Question,
		ExpectedFacts: test.ExpectedFacts,
		Category:      test.Category,
		Explanation:   test.Explanation,
	}

	prompt := fmt.Sprintf(
		"Based on the following document, answer this question thoroughly and accurately. "+
			"Include specific article numbers and relevant details from the document.\n\n"+
			"Question: %s\n\nDocument:\n%s",
		test.Question, e.docText,
	)

	resp, err := e.provider.Chat(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
	})
	if err != nil {
		result.Error = err.Error()
		result.ElapsedMs = time.Since(testStart).Milliseconds()
		return result
	}

	// Construct a synthetic goreason.Answer so existing metric functions work.
	// Sources is empty — this is full-context, no retrieval involved.
	found := resp.Content != ""
	answer := &goreason.Answer{
		Text:             resp.Content,
		Found:            &found,
		Confidence:       0.8, // fixed confidence for full-context (no retrieval signal)
		Sources:          nil, // no retrieval sources
		PromptTokens:     resp.PromptTokens,
		CompletionTokens: resp.CompletionTokens,
		TotalTokens:      resp.TotalTokens,
	}

	result.Answer = answer.Text
	result.Confidence = answer.Confidence
	result.PromptTokens = answer.PromptTokens
	result.CompletionTokens = answer.CompletionTokens
	result.TotalTokens = answer.TotalTokens

	// Compute metrics using existing functions.
	// With no Sources (full-context bypasses RAG):
	//   - Faithfulness: base score (no source penalty/bonus)
	//   - Relevance: 0 (no sources to check)
	//   - Accuracy: fact coverage (primary comparison metric)
	//   - ContextRecall: 0 (no retrieved chunks)
	//   - CitationQuality: base score from citation patterns in text
	//   - ClaimGrounding: 0 (no sources)
	//   - HallucinationScore: 0.5 (neutral, no sources)
	result.Faithfulness = computeFaithfulness(answer)
	result.Relevance = computeRelevance(answer, test.Question)
	result.Accuracy = computeAccuracy(answer, test.ExpectedFacts)
	result.ContextRecall = computeContextRecall(answer, test.ExpectedFacts) // will be 0 (no sources)
	result.CitationQuality = computeCitationQuality(answer)
	result.ClaimGrounding = computeClaimGrounding(answer)
	result.HallucinationScore = computeHallucinationScore(answer)

	// For full-context eval (no RAG), ContextRecall doesn't apply since there
	// are no retrieved chunks. Keep Accuracy + Faithfulness gate for baseline comparison.
	result.Passed = result.Accuracy >= 0.5 && result.Faithfulness >= 0.5
	result.ElapsedMs = time.Since(testStart).Milliseconds()

	return result
}
