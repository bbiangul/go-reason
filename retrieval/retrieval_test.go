package retrieval

import (
	"testing"

	"github.com/brunobiangulo/goreason/store"
)

func TestFuseRRF(t *testing.T) {
	vec := []store.RetrievalResult{
		{ChunkID: 1, Content: "a"},
		{ChunkID: 2, Content: "b"},
	}
	fts := []store.RetrievalResult{
		{ChunkID: 2, Content: "b"},
		{ChunkID: 3, Content: "c"},
	}
	graph := []store.RetrievalResult{
		{ChunkID: 1, Content: "a"},
	}

	results, infoMap := fuseRRF(vec, fts, graph, 1.0, 1.0, 0.5, 10)

	if len(results) != 3 {
		t.Fatalf("expected 3 fused results, got %d", len(results))
	}

	// Verify method tracking
	if info, ok := infoMap[1]; !ok || len(info.Methods) != 2 {
		t.Errorf("chunk 1 should have 2 methods (vec+graph), got %v", infoMap[1])
	}
	if info, ok := infoMap[2]; !ok || len(info.Methods) != 2 {
		t.Errorf("chunk 2 should have 2 methods (vec+fts), got %v", infoMap[2])
	}

	// Compute expected scores manually using RRF formula: weight / (k + rank + 1)
	// where k = 60 (rrfK constant).
	//
	// Chunk 1: vec rank 0 -> 1.0/(60+0+1) = 1/61, graph rank 0 -> 0.5/(60+0+1) = 0.5/61
	//          total = 1.5/61 ≈ 0.02459
	// Chunk 2: vec rank 1 -> 1.0/(60+1+1) = 1/62, fts rank 0 -> 1.0/(60+0+1) = 1/61
	//          total = 1/62 + 1/61 ≈ 0.03252
	// Chunk 3: fts rank 1 -> 1.0/(60+1+1) = 1/62
	//          total = 1/62 ≈ 0.01613

	chunk1Score := 1.0/61.0 + 0.5/61.0
	chunk2Score := 1.0/62.0 + 1.0/61.0
	chunk3Score := 1.0 / 62.0

	// Chunk 2 should have the highest score (appears in both vec and fts).
	if results[0].ChunkID != 2 {
		t.Errorf("expected chunk 2 first (highest score), got chunk %d", results[0].ChunkID)
	}
	// Chunk 1 should be second.
	if results[1].ChunkID != 1 {
		t.Errorf("expected chunk 1 second, got chunk %d", results[1].ChunkID)
	}
	// Chunk 3 should be last.
	if results[2].ChunkID != 3 {
		t.Errorf("expected chunk 3 last, got chunk %d", results[2].ChunkID)
	}

	// Verify actual score values with a tolerance.
	const eps = 1e-9
	if diff := results[0].Score - chunk2Score; diff < -eps || diff > eps {
		t.Errorf("chunk 2 score: got %f, want %f", results[0].Score, chunk2Score)
	}
	if diff := results[1].Score - chunk1Score; diff < -eps || diff > eps {
		t.Errorf("chunk 1 score: got %f, want %f", results[1].Score, chunk1Score)
	}
	if diff := results[2].Score - chunk3Score; diff < -eps || diff > eps {
		t.Errorf("chunk 3 score: got %f, want %f", results[2].Score, chunk3Score)
	}
}

func TestFuseRRFMaxResults(t *testing.T) {
	vec := []store.RetrievalResult{
		{ChunkID: 1, Content: "a"},
		{ChunkID: 2, Content: "b"},
		{ChunkID: 3, Content: "c"},
	}

	results, _ := fuseRRF(vec, nil, nil, 1.0, 1.0, 1.0, 2)
	if len(results) != 2 {
		t.Errorf("expected 2 results with maxResults=2, got %d", len(results))
	}
}

func TestFuseRRFEmptyInputs(t *testing.T) {
	results, _ := fuseRRF(nil, nil, nil, 1.0, 1.0, 1.0, 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty inputs, got %d", len(results))
	}
}

func TestFuseRRFWeightZero(t *testing.T) {
	vec := []store.RetrievalResult{
		{ChunkID: 1, Content: "a"},
	}
	fts := []store.RetrievalResult{
		{ChunkID: 2, Content: "b"},
	}

	// Weight for vec is 0, so chunk 1 should have score 0. Only fts contributes.
	results, _ := fuseRRF(vec, fts, nil, 0.0, 1.0, 0.0, 10)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// fts chunk should be ranked first since vec weight is 0.
	if results[0].ChunkID != 2 {
		t.Errorf("expected chunk 2 first when vec weight=0, got chunk %d", results[0].ChunkID)
	}
}

func TestSanitizeFTSQuery(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "plain text",
			input: "quality management system",
		},
		{
			name:  "special characters removed",
			input: `"ISO 9001" + (quality) - management*`,
		},
		{
			name:  "colons and carets",
			input: "title:ISO category:standard ^boost",
		},
		{
			name:  "single word",
			input: "compliance",
		},
		{
			name:  "short words filtered",
			input: "a to be or not",
		},
		{
			name:  "empty after cleaning",
			input: "quality management",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFTSQuery(tt.input, nil)

			// Result should never contain unescaped FTS5 operators.
			for _, ch := range []string{"*", "(", ")", "+", "^", ":"} {
				if contains(result, ch) {
					t.Errorf("sanitized query still contains %q: %s", ch, result)
				}
			}

			// Result should not be empty for non-empty input with real words.
			if tt.name == "plain text" && result == "" {
				t.Error("expected non-empty result for plain text input")
			}
		})
	}
}

func TestSanitizeFTSQueryMultiWord(t *testing.T) {
	result := sanitizeFTSQuery("ISO 9001 quality", nil)

	// Multi-word inputs should produce quoted phrase + individual terms joined with OR.
	if result == "" {
		t.Fatal("expected non-empty result")
	}

	// Should contain OR separators for multi-term queries.
	if !containsStr(result, "OR") {
		t.Errorf("expected OR in multi-word query, got: %s", result)
	}
}

func TestExtractQueryEntities(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string // at least these should be found
	}{
		{
			name:     "capitalized words",
			query:    "What does ISO 9001 say about Quality Management?",
			expected: []string{"ISO 9001", "Quality Management"},
		},
		{
			name:     "quoted terms",
			query:    `Tell me about "risk assessment" and "force majeure"`,
			expected: []string{"risk assessment", "force majeure"},
		},
		{
			name:     "ISO standard references",
			query:    "Does iso27001 apply here?",
			expected: []string{"iso27001"},
		},
		{
			name:     "section references",
			query:    "What does section 3.2 require?",
			expected: []string{"Section 3.2"},
		},
		{
			name:     "ASTM standard",
			query:    "What is ASTM D638?",
			expected: []string{"ASTM"},
		},
		{
			name:     "IEEE standard",
			query:    "According to IEEE 802.11, what is the range?",
			expected: []string{"IEEE"},
		},
		{
			name:     "significant words in simple query",
			query:    "what is the meaning of this?",
			expected: []string{"meaning"}, // significant lowercase words now extracted
		},
		{
			name:     "mixed capitalization",
			query:    "Compare NIST 800-53 with ISO 27001 Security Controls",
			expected: []string{"NIST", "ISO", "Security Controls"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractQueryEntities(tt.query, nil)

			if tt.expected == nil {
				if len(entities) != 0 {
					t.Errorf("expected no entities, got %v", entities)
				}
				return
			}

			for _, exp := range tt.expected {
				found := false
				for _, e := range entities {
					if containsStr(e, exp) || containsStr(exp, e) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected to find entity matching %q in %v", exp, entities)
				}
			}
		})
	}
}

func TestExtractQueryEntitiesSingleQuotes(t *testing.T) {
	entities := extractQueryEntities("What is 'force majeure' in this context?", nil)
	found := false
	for _, e := range entities {
		if containsStr(e, "force majeure") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find 'force majeure' in entities: %v", entities)
	}
}

func TestIsStopWord(t *testing.T) {
	stopWords := []string{"the", "a", "an", "and", "or", "is", "are", "in", "on"}
	for _, w := range stopWords {
		if !isStopWord(w) {
			t.Errorf("expected %q to be a stop word", w)
		}
	}

	nonStopWords := []string{"quality", "management", "standard", "ISO", "compliance"}
	for _, w := range nonStopWords {
		if isStopWord(w) {
			t.Errorf("expected %q not to be a stop word", w)
		}
	}
}

// contains checks whether s contains the substring sub.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func containsStr(haystack, needle string) bool {
	return len(haystack) >= len(needle) && searchStr(haystack, needle)
}
