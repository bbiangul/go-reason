package eval

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/bbiangul/go-reason"
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
