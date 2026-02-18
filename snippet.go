package goreason

import (
	"strings"
	"unicode"
)

// snippetMaxLen is the approximate maximum character length for a snippet.
const snippetMaxLen = 300

// extractSnippet returns the 1-2 most relevant sentences from content based on
// word overlap with answerWords. Returns empty string if no good match found.
func extractSnippet(content string, answerWords map[string]bool) string {
	if len(answerWords) == 0 || content == "" {
		return ""
	}

	sentences := snippetSplitSentences(content)
	if len(sentences) == 0 {
		return ""
	}

	// Score each sentence by overlap with answer words.
	type scored struct {
		text  string
		score int
		index int
	}
	scoredSentences := make([]scored, len(sentences))
	for i, s := range sentences {
		words := significantWords(s)
		overlap := 0
		for w := range words {
			if answerWords[w] {
				overlap++
			}
		}
		scoredSentences[i] = scored{text: s, score: overlap, index: i}
	}

	// Find the best sentence.
	bestIdx := 0
	bestScore := scoredSentences[0].score
	for i, s := range scoredSentences {
		if s.score > bestScore {
			bestScore = s.score
			bestIdx = i
		}
	}

	if bestScore == 0 {
		return ""
	}

	result := scoredSentences[bestIdx].text

	// Try to add the next-best adjacent sentence if it fits within the limit.
	if len(result) < snippetMaxLen && len(scoredSentences) > 1 {
		// Prefer the adjacent sentence (next or previous) with the highest score.
		candidateIdx := -1
		candidateScore := 0
		for _, delta := range []int{1, -1} {
			adj := bestIdx + delta
			if adj >= 0 && adj < len(scoredSentences) && scoredSentences[adj].score > candidateScore {
				candidateScore = scoredSentences[adj].score
				candidateIdx = adj
			}
		}
		if candidateIdx >= 0 && candidateScore > 0 {
			combined := result + " " + scoredSentences[candidateIdx].text
			if candidateIdx < bestIdx {
				combined = scoredSentences[candidateIdx].text + " " + result
			}
			if len(combined) <= snippetMaxLen {
				result = combined
			}
		}
	}

	return result
}

// significantWords returns the set of lowercased words >= 4 characters,
// excluding common stop words.
func significantWords(text string) map[string]bool {
	words := make(map[string]bool)
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if len(w) >= 4 && !stopWords[w] {
			words[w] = true
		}
	}
	return words
}

// snippetSplitSentences splits text into sentences at period/question/exclamation
// boundaries followed by whitespace or end of string.
func snippetSplitSentences(text string) []string {
	var sentences []string
	var cur strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		cur.WriteRune(runes[i])
		if runes[i] == '.' || runes[i] == '?' || runes[i] == '!' {
			if i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n' || runes[i+1] == '\t' {
				s := strings.TrimSpace(cur.String())
				if s != "" {
					sentences = append(sentences, s)
				}
				cur.Reset()
			}
		}
	}
	if cur.Len() > 0 {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			sentences = append(sentences, s)
		}
	}
	return sentences
}

// stopWords is a set of common English stop words to exclude from matching.
var stopWords = map[string]bool{
	"that": true, "this": true, "with": true, "from": true,
	"have": true, "been": true, "were": true, "they": true,
	"their": true, "will": true, "would": true, "could": true,
	"should": true, "about": true, "which": true, "there": true,
	"these": true, "those": true, "then": true, "than": true,
	"them": true, "what": true, "when": true, "where": true,
	"your": true, "more": true, "some": true, "such": true,
	"only": true, "also": true, "very": true, "just": true,
	"into": true, "over": true, "each": true, "does": true,
	"most": true, "after": true, "before": true, "other": true,
	"being": true, "same": true, "both": true, "between": true,
}
