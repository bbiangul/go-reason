package reasoning

import (
	"strings"

	"github.com/brunobiangulo/goreason/store"
)

// ConfidenceWeights controls the relative importance of confidence factors.
type ConfidenceWeights struct {
	SourceCoverage   float64 // How many sources are referenced
	CitationAccuracy float64 // How accurate citations are
	SelfConsistency  float64 // Internal consistency of the answer
	AnswerLength     float64 // Whether the answer is substantive
}

// DefaultConfidenceWeights returns balanced weights.
func DefaultConfidenceWeights() ConfidenceWeights {
	return ConfidenceWeights{
		SourceCoverage:   0.3,
		CitationAccuracy: 0.3,
		SelfConsistency:  0.25,
		AnswerLength:     0.15,
	}
}

// ComputeConfidence calculates a confidence score for an answer.
func ComputeConfidence(answer string, chunks []store.RetrievalResult, weights ConfidenceWeights) float64 {
	sc := sourceCoverageScore(answer, chunks)
	ca := citationAccuracyScore(answer, chunks)
	si := selfConsistencyScore(answer)
	al := answerLengthScore(answer)

	confidence := sc*weights.SourceCoverage +
		ca*weights.CitationAccuracy +
		si*weights.SelfConsistency +
		al*weights.AnswerLength

	if confidence < 0 {
		return 0
	}
	if confidence > 1 {
		return 1
	}
	return confidence
}

// sourceCoverageScore measures what fraction of top sources are referenced.
func sourceCoverageScore(answer string, chunks []store.RetrievalResult) float64 {
	if len(chunks) == 0 {
		return 0
	}

	lower := strings.ToLower(answer)
	referenced := 0
	// Check top 5 sources
	checkCount := len(chunks)
	if checkCount > 5 {
		checkCount = 5
	}

	for _, c := range chunks[:checkCount] {
		if c.Filename != "" && strings.Contains(lower, strings.ToLower(c.Filename)) {
			referenced++
			continue
		}
		if c.Heading != "" && strings.Contains(lower, strings.ToLower(c.Heading)) {
			referenced++
			continue
		}
		// Check if any significant phrase from the chunk appears in the answer
		words := strings.Fields(c.Content)
		if len(words) > 5 {
			phrase := strings.Join(words[:5], " ")
			if strings.Contains(lower, strings.ToLower(phrase)) {
				referenced++
			}
		}
	}

	return float64(referenced) / float64(checkCount)
}

// citationAccuracyScore measures how many citations can be verified.
func citationAccuracyScore(answer string, chunks []store.RetrievalResult) float64 {
	citations := ExtractCitations(answer, chunks)
	if len(citations) == 0 {
		return 0.5 // neutral if no citations found
	}

	verified := 0
	for _, c := range citations {
		if c.Verified {
			verified++
		}
	}

	return float64(verified) / float64(len(citations))
}

// selfConsistencyScore checks for internal consistency.
func selfConsistencyScore(answer string) float64 {
	lower := strings.ToLower(answer)
	score := 1.0

	// Penalize contradictory language
	contradictions := []string{
		"on the other hand",
		"however, it also",
		"contradicts",
		"inconsistent",
	}
	for _, c := range contradictions {
		if strings.Contains(lower, c) {
			score -= 0.15
		}
	}

	// Penalize uncertainty markers
	uncertainties := []string{
		"i'm not sure",
		"it's unclear",
		"cannot determine",
		"insufficient information",
		"not enough context",
	}
	for _, u := range uncertainties {
		if strings.Contains(lower, u) {
			score -= 0.2
		}
	}

	if score < 0 {
		return 0
	}
	return score
}

// answerLengthScore gives higher scores to substantive answers.
func answerLengthScore(answer string) float64 {
	words := len(strings.Fields(answer))
	switch {
	case words < 10:
		return 0.2
	case words < 30:
		return 0.5
	case words < 100:
		return 0.8
	case words < 500:
		return 1.0
	default:
		return 0.9 // slightly lower for very long answers
	}
}
