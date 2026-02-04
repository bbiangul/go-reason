package reasoning

import (
	"strings"

	"github.com/brunobiangulo/goreason/store"
)

// validationResult holds the outcome of answer validation.
type validationResult struct {
	citationValid    bool
	citationIssues   []string
	consistencyValid bool
	consistencyIssues []string
	completenessValid bool
	completenessIssues []string
}

func (v *validationResult) summary() string {
	var parts []string

	if !v.citationValid {
		parts = append(parts, "Citation issues: "+strings.Join(v.citationIssues, "; "))
	}
	if !v.consistencyValid {
		parts = append(parts, "Consistency issues: "+strings.Join(v.consistencyIssues, "; "))
	}
	if !v.completenessValid {
		parts = append(parts, "Completeness issues: "+strings.Join(v.completenessIssues, "; "))
	}

	if len(parts) == 0 {
		return "All validations passed."
	}
	return strings.Join(parts, "\n")
}

func (v *validationResult) confidence() float64 {
	score := 1.0

	if !v.citationValid {
		score -= 0.15 * float64(len(v.citationIssues))
	}
	if !v.consistencyValid {
		score -= 0.2 * float64(len(v.consistencyIssues))
	}
	if !v.completenessValid {
		score -= 0.1 * float64(len(v.completenessIssues))
	}

	if score < 0 {
		score = 0
	}
	return score
}

// validate runs all validators on an answer.
func validate(answer string, chunks []store.RetrievalResult) *validationResult {
	result := &validationResult{
		citationValid:     true,
		consistencyValid:  true,
		completenessValid: true,
	}

	validateCitations(answer, chunks, result)
	validateConsistency(answer, chunks, result)

	return result
}

// validateCitations checks that the answer references sources that exist.
func validateCitations(answer string, chunks []store.RetrievalResult, result *validationResult) {
	lowerAnswer := strings.ToLower(answer)

	// Check if the answer makes any reference to sources at all
	hasAnyRef := false
	for _, c := range chunks {
		if c.Filename != "" && strings.Contains(lowerAnswer, strings.ToLower(c.Filename)) {
			hasAnyRef = true
			break
		}
		if c.Heading != "" && strings.Contains(lowerAnswer, strings.ToLower(c.Heading)) {
			hasAnyRef = true
			break
		}
	}

	if !hasAnyRef && len(chunks) > 0 {
		result.citationValid = false
		result.citationIssues = append(result.citationIssues,
			"Answer does not reference any of the provided sources")
	}

	// Check for fabricated references (mentions of documents not in sources)
	// Simple heuristic: look for "Section X.Y" or "Article X" patterns
	sentences := strings.Split(answer, ".")
	for _, sent := range sentences {
		lower := strings.ToLower(strings.TrimSpace(sent))
		if strings.Contains(lower, "according to") || strings.Contains(lower, "as stated in") {
			// Check if the referenced document exists in our sources
			found := false
			for _, c := range chunks {
				if c.Filename != "" && strings.Contains(lower, strings.ToLower(c.Filename)) {
					found = true
					break
				}
			}
			if !found && (strings.Contains(lower, "document") || strings.Contains(lower, "report")) {
				result.citationValid = false
				result.citationIssues = append(result.citationIssues,
					"Possible fabricated reference in: "+strings.TrimSpace(sent))
			}
		}
	}
}

// validateConsistency checks that the answer doesn't contradict the sources.
func validateConsistency(answer string, chunks []store.RetrievalResult, result *validationResult) {
	lowerAnswer := strings.ToLower(answer)

	// Check for negation patterns that might indicate contradictions
	// This is a heuristic - in production you'd use the LLM for this
	negations := []string{
		"however, the document states otherwise",
		"contrary to",
		"this contradicts",
		"the document says the opposite",
	}

	for _, neg := range negations {
		if strings.Contains(lowerAnswer, neg) {
			result.consistencyValid = false
			result.consistencyIssues = append(result.consistencyIssues,
				"Answer contains potential self-contradiction")
		}
	}

	// Check if the answer claims information not found in sources
	if strings.Contains(lowerAnswer, "based on my knowledge") ||
		strings.Contains(lowerAnswer, "in general") ||
		strings.Contains(lowerAnswer, "it is commonly known") {
		result.consistencyValid = false
		result.consistencyIssues = append(result.consistencyIssues,
			"Answer appears to use external knowledge instead of provided sources")
	}
}
