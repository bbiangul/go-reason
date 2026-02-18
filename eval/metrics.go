package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"unicode"

	"github.com/bbiangul/go-reason"
	"github.com/bbiangul/go-reason/llm"
)

// normalizeLLMText normalizes Unicode characters commonly inserted by LLMs
// so that substring matching works reliably. Handles:
//   - Unicode whitespace → ASCII space (U+202F, U+00A0, etc.)
//   - Unicode hyphens → ASCII hyphen (U+2011, U+2010, U+2012, U+2013, U+2014)
//   - Strips zero-width characters (U+200B, U+200C, U+200D, U+FEFF)
func normalizeLLMText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case unicode.IsSpace(r):
			b.WriteByte(' ')
		case r == '\u2010' || r == '\u2011' || r == '\u2012' || r == '\u2013' || r == '\u2014':
			b.WriteByte('-')
		case r == '\u200B' || r == '\u200C' || r == '\u200D' || r == '\uFEFF':
			// strip zero-width characters
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normalizeSpaces is an alias kept for backward compatibility.
func normalizeSpaces(s string) string { return normalizeLLMText(s) }

// computeFaithfulness checks if the answer only contains information from sources.
func computeFaithfulness(answer *goreason.Answer) float64 {
	if answer == nil || answer.Text == "" {
		return 0
	}

	lower := strings.ToLower(answer.Text)

	// Penalize for external knowledge indicators
	externalIndicators := []string{
		"based on my knowledge",
		"in general",
		"it is commonly known",
		"typically",
		"usually",
		"as everyone knows",
		"from my understanding",
	}

	score := 1.0
	for _, indicator := range externalIndicators {
		if strings.Contains(lower, indicator) {
			score -= 0.2
		}
	}

	// Bonus for source references
	if len(answer.Sources) > 0 {
		referencedSources := 0
		for _, src := range answer.Sources {
			if src.Filename != "" && strings.Contains(lower, strings.ToLower(src.Filename)) {
				referencedSources++
			}
		}
		if referencedSources > 0 {
			score += 0.1 * float64(min(referencedSources, 3))
		}
	}

	return clamp(score)
}

// computeRelevance checks if the retrieved chunks are relevant to the question.
func computeRelevance(answer *goreason.Answer, question string) float64 {
	if answer == nil || len(answer.Sources) == 0 {
		return 0
	}

	questionWords := significantWords(question)
	if len(questionWords) == 0 {
		return 0.5
	}

	relevantSources := 0
	for _, src := range answer.Sources {
		srcLower := strings.ToLower(src.Content + " " + src.Heading)
		matchCount := 0
		for _, w := range questionWords {
			if strings.Contains(srcLower, w) {
				matchCount++
			}
		}
		if float64(matchCount)/float64(len(questionWords)) >= 0.3 {
			relevantSources++
		}
	}

	return clamp(float64(relevantSources) / float64(len(answer.Sources)))
}

// computeAccuracy checks if expected facts appear in the answer.
// Each fact may contain pipe-separated alternatives (e.g. "nivel de llenado|fill level"),
// where matching any alternative counts as a hit for that fact.
func computeAccuracy(answer *goreason.Answer, expectedFacts []string) float64 {
	if answer == nil || answer.Text == "" || len(expectedFacts) == 0 {
		return 0
	}

	normalized := normalizeLLMText(strings.ToLower(answer.Text))
	// Prepare a version with spaces collapsed for matching facts like "5%" against "5 %"
	spaceless := strings.ReplaceAll(normalized, " ", "")
	// Prepare a version with hyphens and spaces stripped so "fill-level" matches "fill level"
	hyphenless := strings.ReplaceAll(strings.ReplaceAll(normalized, "-", ""), " ", "")
	found := 0
	for _, fact := range expectedFacts {
		alternatives := strings.Split(fact, "|")
		for _, alt := range alternatives {
			alt = strings.TrimSpace(alt)
			if alt == "" {
				continue
			}
			normAlt := normalizeLLMText(strings.ToLower(alt))
			normAltNoSpace := strings.ReplaceAll(normAlt, " ", "")
			normAltNoHyphen := strings.ReplaceAll(strings.ReplaceAll(normAlt, "-", ""), " ", "")
			if strings.Contains(normalized, normAlt) ||
				strings.Contains(spaceless, normAltNoSpace) ||
				strings.Contains(hyphenless, normAltNoHyphen) {
				found++
				break
			}
		}
	}

	return float64(found) / float64(len(expectedFacts))
}

// computeAccuracyLLM uses an LLM judge to semantically evaluate whether each
// expected fact is covered by the answer. This handles paraphrasing that
// verbatim substring matching misses. All facts are batched into a single
// LLM call for efficiency.
func computeAccuracyLLM(ctx context.Context, judge llm.Provider, model string, answer *goreason.Answer, expectedFacts []string) (float64, error) {
	if answer == nil || answer.Text == "" || len(expectedFacts) == 0 {
		return 0, nil
	}

	// Build the numbered fact list for the prompt
	var factsBuilder strings.Builder
	for i, fact := range expectedFacts {
		alternatives := strings.Split(fact, "|")
		primary := strings.TrimSpace(alternatives[0])
		fmt.Fprintf(&factsBuilder, "%d. %s", i+1, primary)
		if len(alternatives) > 1 {
			var alts []string
			for _, a := range alternatives[1:] {
				a = strings.TrimSpace(a)
				if a != "" {
					alts = append(alts, a)
				}
			}
			if len(alts) > 0 {
				fmt.Fprintf(&factsBuilder, " (alternatives: %s)", strings.Join(alts, ", "))
			}
		}
		factsBuilder.WriteByte('\n')
	}

	prompt := fmt.Sprintf(`You are an evaluation judge for a RAG system. Determine which expected facts are semantically covered by the answer.

A fact is "covered" if the answer conveys the same core information, even if paraphrased, summarized, or worded differently.
A fact is NOT covered if the answer contradicts it, omits it entirely, or gets key details (numbers, names, dates) wrong.

Answer:
%s

Expected Facts:
%s
Respond with JSON: {"covered": [true, false, ...]} — one boolean per fact, in order.`, answer.Text, factsBuilder.String())

	resp, err := judge.Chat(ctx, llm.ChatRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Temperature:    0,
		ResponseFormat: "json_object",
	})
	if err != nil {
		return 0, fmt.Errorf("judge LLM call failed: %w", err)
	}

	// Parse the JSON response
	var result struct {
		Covered []bool `json:"covered"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return 0, fmt.Errorf("judge response parse error: %w (response: %s)", err, truncateStr(resp.Content, 200))
	}

	if len(result.Covered) != len(expectedFacts) {
		slog.Warn("judge returned wrong number of booleans",
			"expected", len(expectedFacts),
			"got", len(result.Covered))
		// Use what we got, capped to expected length
		if len(result.Covered) > len(expectedFacts) {
			result.Covered = result.Covered[:len(expectedFacts)]
		}
	}

	covered := 0
	for _, c := range result.Covered {
		if c {
			covered++
		}
	}
	return float64(covered) / float64(len(expectedFacts)), nil
}

// truncateStr truncates a string to maxLen characters for logging.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// computeContextRecall checks what fraction of expected facts are present in
// the retrieved chunk content. This measures whether the retrieval system
// surfaced the evidence needed to answer the question correctly.
// Uses the same pipe-separated alternative matching as computeAccuracy.
func computeContextRecall(answer *goreason.Answer, expectedFacts []string) float64 {
	if answer == nil || len(answer.Sources) == 0 || len(expectedFacts) == 0 {
		return 0
	}

	// Build concatenated corpus from all retrieved chunks
	var corpus strings.Builder
	for _, src := range answer.Sources {
		corpus.WriteString(src.Content)
		corpus.WriteByte(' ')
		corpus.WriteString(src.Heading)
		corpus.WriteByte(' ')
	}
	corpusText := strings.ToLower(corpus.String())
	normalized := normalizeLLMText(corpusText)
	spaceless := strings.ReplaceAll(normalized, " ", "")
	hyphenless := strings.ReplaceAll(strings.ReplaceAll(normalized, "-", ""), " ", "")

	found := 0
	for _, fact := range expectedFacts {
		alternatives := strings.Split(fact, "|")
		for _, alt := range alternatives {
			alt = strings.TrimSpace(alt)
			if alt == "" {
				continue
			}
			normAlt := normalizeLLMText(strings.ToLower(alt))
			normAltNoSpace := strings.ReplaceAll(normAlt, " ", "")
			normAltNoHyphen := strings.ReplaceAll(strings.ReplaceAll(normAlt, "-", ""), " ", "")
			if strings.Contains(normalized, normAlt) ||
				strings.Contains(spaceless, normAltNoSpace) ||
				strings.Contains(hyphenless, normAltNoHyphen) {
				found++
				break
			}
		}
	}

	return float64(found) / float64(len(expectedFacts))
}

// computeCitationQuality checks if citations are precise and verifiable.
func computeCitationQuality(answer *goreason.Answer) float64 {
	if answer == nil || answer.Text == "" {
		return 0
	}

	lower := strings.ToLower(answer.Text)
	score := 0.5 // base score

	// Check for specific citation patterns (English and Spanish)
	citationPatterns := []string{
		"section", "article", "clause", "page",
		"paragraph", "table", "figure",
		"sección", "capítulo", "página", "tabla", "figura", "anexo",
	}
	citationCount := 0
	for _, p := range citationPatterns {
		if strings.Contains(lower, p) {
			citationCount++
		}
	}

	if citationCount > 0 {
		score += 0.1 * float64(min(citationCount, 3))
	}

	// Check if filenames are referenced
	for _, src := range answer.Sources {
		if src.Filename != "" && strings.Contains(lower, strings.ToLower(src.Filename)) {
			score += 0.1
			break
		}
	}

	return clamp(score)
}

func significantWords(text string) []string {
	stopWords := map[string]bool{
		// English stop words (3+ bytes only; shorter are filtered by len(w) > 2)
		"the": true, "are": true, "was": true, "were": true,
		"for": true, "with": true,
		"what": true, "which": true, "who": true, "how": true, "where": true,
		"when": true, "that": true, "this": true, "and": true,
		// Spanish stop words (3+ bytes only; 2-byte words like "el","la","de"
		// are already filtered by the len(w) > 2 check below)
		"del": true, "los": true, "las": true, "una": true,
		"que": true, "por": true, "con": true, "para": true,
		"como": true, "más": true, "pero": true,
		"sus": true, "entre": true, "también": true,
		"desde": true, "sobre": true, "tiene": true, "ser": true,
		"son": true, "está": true, "hay": true, "fue": true,
		"cuál": true, "qué": true, "cómo": true, "dónde": true,
	}

	var words []string
	for _, w := range strings.Fields(text) {
		w = strings.Trim(strings.ToLower(w), ".,;:!?\"'()[]")
		if len(w) > 2 && !stopWords[w] {
			words = append(words, w)
		}
	}
	return words
}

// numberPattern matches integers and decimals (e.g. "120", "3.14", "1.2").
var numberPattern = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)

// computeClaimGrounding verifies that key claims in the answer have support
// in the retrieved sources. Returns 1.0 when all significant terms appear in
// at least one source, 0.0 when none do.
func computeClaimGrounding(answer *goreason.Answer) float64 {
	if answer == nil || answer.Text == "" || len(answer.Sources) == 0 {
		return 0
	}

	// Build a single lowercased corpus from all source content.
	var corpus strings.Builder
	for _, src := range answer.Sources {
		corpus.WriteString(strings.ToLower(src.Content))
		corpus.WriteByte(' ')
		corpus.WriteString(strings.ToLower(src.Heading))
		corpus.WriteByte(' ')
	}
	corpusStr := corpus.String()

	// Extract significant terms: technical words (>3 chars, non-stop) + numbers.
	// Use a set to avoid double-counting (numbers can appear in both).
	answerLower := strings.ToLower(answer.Text)
	seen := make(map[string]struct{})
	var terms []string
	for _, w := range significantWords(answerLower) {
		if _, ok := seen[w]; !ok {
			seen[w] = struct{}{}
			terms = append(terms, w)
		}
	}
	for _, num := range numberPattern.FindAllString(answerLower, -1) {
		if _, ok := seen[num]; !ok {
			seen[num] = struct{}{}
			terms = append(terms, num)
		}
	}

	if len(terms) == 0 {
		return 1.0 // nothing to check
	}

	grounded := 0
	for _, term := range terms {
		if strings.Contains(corpusStr, term) {
			grounded++
		}
	}
	return clamp(float64(grounded) / float64(len(terms)))
}

// computeHallucinationScore detects fabricated numbers and entities.
// Returns 1.0 (clean, no hallucination detected) to 0.0 (heavy hallucination).
func computeHallucinationScore(answer *goreason.Answer) float64 {
	if answer == nil || answer.Text == "" {
		return 0
	}
	if len(answer.Sources) == 0 {
		// No sources to check against — can't verify, assume neutral.
		return 0.5
	}

	// Build source corpus.
	var corpus strings.Builder
	for _, src := range answer.Sources {
		corpus.WriteString(strings.ToLower(src.Content))
		corpus.WriteByte(' ')
		corpus.WriteString(strings.ToLower(src.Heading))
		corpus.WriteByte(' ')
	}
	corpusStr := corpus.String()

	answerLower := strings.ToLower(answer.Text)

	// 1. Check numbers in the answer against sources.
	numbers := numberPattern.FindAllString(answerLower, -1)
	trivialNumbers := map[string]bool{
		"0": true, "1": true, "2": true, "3": true, "4": true,
		"5": true, "6": true, "7": true, "8": true, "9": true, "10": true,
	}

	totalChecks := 0
	penalties := 0.0
	maxPenalties := 0.0

	for _, num := range numbers {
		if trivialNumbers[num] {
			continue
		}
		totalChecks++
		maxPenalties += 1.0
		if !strings.Contains(corpusStr, num) {
			penalties += 1.0 // ungrounded number
		}
	}

	// 2. Check technical terms (words >5 chars that look like proper nouns or technical terms).
	words := significantWords(answerLower)
	for _, w := range words {
		if len(w) <= 5 {
			continue
		}
		totalChecks++
		maxPenalties += 0.5
		if !strings.Contains(corpusStr, w) {
			penalties += 0.5 // ungrounded term (less severe than numbers)
		}
	}

	if totalChecks == 0 {
		return 1.0
	}

	// Score: 1.0 when no penalties, 0.0 when all checks penalized at their max weight.
	score := 1.0 - (penalties / maxPenalties)
	return clamp(score)
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RetrievalKValues are the k values at which P@k and R@k are computed.
var RetrievalKValues = []int{1, 4, 8, 16, 32, 64}

// computeRetrievalPrecisionAtK computes what fraction of the top-k retrieved
// chunks contain ground-truth text. A chunk is "relevant" if it contains any
// substring from the ground-truth spans (case-insensitive).
func computeRetrievalPrecisionAtK(answer *goreason.Answer, groundTruth []GroundTruthSpan, k int) float64 {
	if answer == nil || len(answer.Sources) == 0 || len(groundTruth) == 0 {
		return 0
	}

	topK := answer.Sources
	if len(topK) > k {
		topK = topK[:k]
	}

	relevant := 0
	for _, src := range topK {
		if chunkMatchesGroundTruth(src, groundTruth) {
			relevant++
		}
	}

	return float64(relevant) / float64(len(topK))
}

// computeRetrievalRecallAtK computes what fraction of ground-truth spans are
// covered by at least one of the top-k retrieved chunks.
func computeRetrievalRecallAtK(answer *goreason.Answer, groundTruth []GroundTruthSpan, k int) float64 {
	if answer == nil || len(answer.Sources) == 0 || len(groundTruth) == 0 {
		return 0
	}

	topK := answer.Sources
	if len(topK) > k {
		topK = topK[:k]
	}

	found := 0
	for _, gt := range groundTruth {
		gtLower := strings.ToLower(gt.Text)
		for _, src := range topK {
			srcLower := strings.ToLower(src.Content)
			// A ground-truth span is "found" if either:
			// 1. The chunk contains the full snippet text, OR
			// 2. The chunk is from the same file and contains a significant
			//    portion of the snippet (for cases where chunking splits it).
			if strings.Contains(srcLower, gtLower) {
				found++
				break
			}
			// Check file match + substantial overlap (>50% of snippet words).
			if strings.EqualFold(src.Filename, gt.FilePath) && snippetOverlap(srcLower, gtLower) > 0.5 {
				found++
				break
			}
		}
	}

	return float64(found) / float64(len(groundTruth))
}

// chunkMatchesGroundTruth checks if a retrieved chunk contains text from any
// ground-truth span.
func chunkMatchesGroundTruth(src goreason.Source, groundTruth []GroundTruthSpan) bool {
	srcLower := strings.ToLower(src.Content)
	for _, gt := range groundTruth {
		gtLower := strings.ToLower(gt.Text)
		if strings.Contains(srcLower, gtLower) {
			return true
		}
		if strings.EqualFold(src.Filename, gt.FilePath) && snippetOverlap(srcLower, gtLower) > 0.5 {
			return true
		}
	}
	return false
}

// snippetOverlap computes the fraction of words in the ground-truth snippet
// that appear in the chunk content.
func snippetOverlap(chunkLower, snippetLower string) float64 {
	words := strings.Fields(snippetLower)
	if len(words) == 0 {
		return 0
	}
	found := 0
	for _, w := range words {
		if len(w) > 3 && strings.Contains(chunkLower, w) {
			found++
		}
	}
	return float64(found) / float64(len(words))
}
