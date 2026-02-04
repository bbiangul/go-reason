package retrieval

import (
	"strings"
	"unicode"
)

// extractSignificantTerms returns the meaningful words from a query,
// filtering out short words and stop words. Used to collect terms for
// cross-language translation before building FTS/graph queries.
func extractSignificantTerms(query string) []string {
	replacer := strings.NewReplacer(
		"\"", "", "*", "", "(", "", ")", "",
		"+", "", "-", "", "^", "", ":", "",
		"?", "", "[", "", "]", "", "{", "",
		"}", "", "!", "", ".", "", ",", "",
		";", "",
	)
	cleaned := replacer.Replace(query)
	words := strings.Fields(cleaned)

	seen := make(map[string]bool)
	var terms []string
	for _, w := range words {
		lower := strings.ToLower(w)
		if len(lower) > 2 && !isStopWord(lower) && !seen[lower] {
			seen[lower] = true
			terms = append(terms, lower)
		}
	}
	return terms
}

// sanitizeFTSQuery escapes special FTS5 syntax characters and builds
// a basic OR query from the input terms. translated contains additional
// terms from cross-language expansion (may be nil).
func sanitizeFTSQuery(query string, translated []string) string {
	// Remove FTS5 special characters
	replacer := strings.NewReplacer(
		"\"", "",
		"*", "",
		"(", "",
		")", "",
		"+", "",
		"-", "",
		"^", "",
		":", "",
		"?", "",
		"[", "",
		"]", "",
		"{", "",
		"}", "",
		"!", "",
		".", "",
		",", "",
		";", "",
	)
	cleaned := replacer.Replace(query)

	// Split into words and join with OR for broader matching
	words := strings.Fields(cleaned)
	if len(words) == 0 {
		return query
	}

	// Use quoted phrase for exact matches plus individual terms
	var parts []string
	if len(words) > 1 {
		// Add the full phrase
		parts = append(parts, "\""+strings.Join(words, " ")+"\"")
	}
	// Add individual significant words (skip short common words)
	for _, w := range words {
		if len(w) > 2 && !isStopWord(w) {
			parts = append(parts, w)
		}
	}

	// Append cross-language translated terms
	parts = append(parts, translated...)

	if len(parts) == 0 {
		return strings.Join(words, " OR ")
	}
	return strings.Join(parts, " OR ")
}

// extractQueryEntities does simple entity extraction from a query string.
// Extracts capitalized phrases, quoted terms, and domain-specific patterns.
// translated contains additional terms from cross-language expansion (may be nil).
func extractQueryEntities(query string, translated []string) []string {
	var entities []string
	seen := make(map[string]bool)

	add := func(s string) {
		s = strings.TrimSpace(s)
		lower := strings.ToLower(s)
		if s != "" && !seen[lower] && len(s) > 1 {
			seen[lower] = true
			entities = append(entities, s)
		}
	}

	// Extract quoted terms
	inQuote := false
	var quoted strings.Builder
	for _, r := range query {
		if r == '"' || r == '\'' {
			if inQuote {
				add(quoted.String())
				quoted.Reset()
			}
			inQuote = !inQuote
			continue
		}
		if inQuote {
			quoted.WriteRune(r)
		}
	}

	// Extract capitalized multi-word phrases
	words := strings.Fields(query)
	var phrase []string
	for _, w := range words {
		clean := strings.Trim(w, ".,;:!?\"'()[]")
		if clean == "" {
			continue
		}

		firstRune := []rune(clean)[0]
		if unicode.IsUpper(firstRune) && !isStopWord(strings.ToLower(clean)) {
			phrase = append(phrase, clean)
		} else {
			if len(phrase) > 0 {
				add(strings.Join(phrase, " "))
				phrase = nil
			}
		}
	}
	if len(phrase) > 0 {
		add(strings.Join(phrase, " "))
	}

	// Extract domain patterns: ISO/IEC numbers, section references
	for _, w := range words {
		clean := strings.Trim(w, ".,;:!?\"'()[]")
		lower := strings.ToLower(clean)
		if strings.HasPrefix(lower, "iso") || strings.HasPrefix(lower, "iec") ||
			strings.HasPrefix(lower, "astm") || strings.HasPrefix(lower, "ieee") {
			add(clean)
		}
		// Section references like "3.2", "4.1.1"
		if len(clean) >= 3 && clean[0] >= '0' && clean[0] <= '9' && strings.Contains(clean, ".") {
			allDigitsAndDots := true
			for _, r := range clean {
				if !unicode.IsDigit(r) && r != '.' {
					allDigitsAndDots = false
					break
				}
			}
			if allDigitsAndDots {
				add("Section " + clean)
			}
		}
	}

	// Also add significant individual words as potential entity names
	// Include both capitalized words AND lowercase words (len > 3, non-stop)
	// to match entities extracted from foreign-language documents.
	for _, w := range words {
		clean := strings.Trim(w, ".,;:!?\"'()[]")
		if len(clean) > 3 && !isStopWord(strings.ToLower(clean)) {
			add(clean)
		}
	}

	// Append cross-language translated terms as additional entity candidates
	for _, t := range translated {
		add(t)
	}

	return entities
}

// isSynthesisQuery returns true if the query has exhaustive intent —
// asking for ALL items, every reference, complete lists, etc.
// These queries benefit from a wider retrieval window because relevant
// facts are scattered across many topically distant chunks.
func isSynthesisQuery(query string) bool {
	lower := strings.ToLower(query)

	// Explicit exhaustive-intent phrases (English only — queries are
	// expected in the user's language; the cross-language translator
	// handles mapping to document content).
	exhaustivePatterns := []string{
		"all the", "all of the", "every ", "each of",
		"complete list", "comprehensive", "list all",
		"all references", "what are all", "name all",
		"list every", "list each", "enumerate",
		"full list", "entire list",
		"every single",
	}
	for _, p := range exhaustivePatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	// Long queries (15+ words) with multiple question keywords suggest
	// broad synthesis questions rather than point lookups.
	words := strings.Fields(lower)
	if len(words) >= 15 {
		qWords := 0
		for _, w := range words {
			switch w {
			case "what", "which", "how", "where", "when", "why", "list", "describe", "name":
				qWords++
			}
		}
		if qWords >= 2 {
			return true
		}
	}

	return false
}

var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true,
	"but": true, "in": true, "on": true, "at": true, "to": true,
	"for": true, "of": true, "with": true, "by": true, "from": true,
	"is": true, "are": true, "was": true, "were": true, "be": true,
	"been": true, "being": true, "have": true, "has": true, "had": true,
	"do": true, "does": true, "did": true, "will": true, "would": true,
	"could": true, "should": true, "may": true, "might": true, "must": true,
	"shall": true, "can": true, "this": true, "that": true, "these": true,
	"those": true, "what": true, "which": true, "who": true, "whom": true,
	"where": true, "when": true, "how": true, "why": true, "not": true,
	"no": true, "nor": true, "if": true, "then": true, "than": true,
	"so": true, "as": true, "about": true, "into": true, "between": true,
}

func isStopWord(w string) bool {
	return stopWords[strings.ToLower(w)]
}
