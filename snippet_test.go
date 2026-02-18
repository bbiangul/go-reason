package goreason

import (
	"testing"
)

func TestExtractSnippet_BasicOverlap(t *testing.T) {
	content := "The motor operates at 5kW rated power. The voltage supply is 230V AC. Safety requirements follow ISO 13849."
	answerWords := significantWords("The motor has a rated power of 5kW according to the specification.")

	snippet := extractSnippet(content, answerWords)
	if snippet == "" {
		t.Fatal("expected non-empty snippet")
	}
	// Should contain the motor/power sentence as best match
	if !containsSubstring(snippet, "motor") {
		t.Errorf("expected snippet to mention motor, got: %q", snippet)
	}
}

func TestExtractSnippet_NoOverlap(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog."
	answerWords := significantWords("quantum computing uses superconducting qubits")

	snippet := extractSnippet(content, answerWords)
	if snippet != "" {
		t.Errorf("expected empty snippet when no overlap, got: %q", snippet)
	}
}

func TestExtractSnippet_EmptyInputs(t *testing.T) {
	if s := extractSnippet("", map[string]bool{"test": true}); s != "" {
		t.Errorf("expected empty for empty content, got: %q", s)
	}
	if s := extractSnippet("some content here.", nil); s != "" {
		t.Errorf("expected empty for nil answerWords, got: %q", s)
	}
	if s := extractSnippet("some content here.", map[string]bool{}); s != "" {
		t.Errorf("expected empty for empty answerWords, got: %q", s)
	}
}

func TestExtractSnippet_RespectMaxLen(t *testing.T) {
	// Build content with many sentences
	content := "First sentence about motors. Second sentence about voltage ratings. " +
		"Third sentence about safety compliance. Fourth sentence about wiring diagrams. " +
		"Fifth sentence about installation procedures. Sixth sentence about maintenance schedules."
	answerWords := significantWords("motors voltage safety wiring installation maintenance")

	snippet := extractSnippet(content, answerWords)
	if len(snippet) > snippetMaxLen {
		t.Errorf("snippet exceeds max length: %d > %d", len(snippet), snippetMaxLen)
	}
}

func TestSignificantWords(t *testing.T) {
	words := significantWords("The motor operates at 5kW. This is very important for safety.")

	// Should include words >= 4 chars, excluding stop words
	if !words["motor"] {
		t.Error("expected 'motor' in significant words")
	}
	if !words["operates"] {
		t.Error("expected 'operates' in significant words")
	}
	if !words["important"] {
		t.Error("expected 'important' in significant words")
	}
	if !words["safety"] {
		t.Error("expected 'safety' in significant words")
	}

	// Should exclude stop words and short words
	if words["this"] {
		t.Error("'this' should be excluded (stop word)")
	}
	if words["very"] {
		t.Error("'very' should be excluded (stop word)")
	}
	if words["the"] {
		t.Error("'the' should be excluded (< 4 chars)")
	}
	if words["at"] {
		t.Error("'at' should be excluded (< 4 chars)")
	}
}

func TestSnippetSplitSentences(t *testing.T) {
	text := "First sentence. Second sentence? Third sentence! Final text without period"
	sentences := snippetSplitSentences(text)

	if len(sentences) != 4 {
		t.Fatalf("expected 4 sentences, got %d: %v", len(sentences), sentences)
	}
	if sentences[0] != "First sentence." {
		t.Errorf("sentence 0: got %q", sentences[0])
	}
	if sentences[1] != "Second sentence?" {
		t.Errorf("sentence 1: got %q", sentences[1])
	}
	if sentences[2] != "Third sentence!" {
		t.Errorf("sentence 2: got %q", sentences[2])
	}
	if sentences[3] != "Final text without period" {
		t.Errorf("sentence 3: got %q", sentences[3])
	}
}

func TestExtractSnippet_AdjacentSentences(t *testing.T) {
	// When best sentence is short, should include an adjacent one
	content := "Setup is easy. The motor runs at 5kW. The voltage is 230V."
	answerWords := significantWords("motor 5kW voltage 230V")

	snippet := extractSnippet(content, answerWords)
	// Should pick the two best-scoring adjacent sentences
	if !containsSubstring(snippet, "motor") {
		t.Errorf("expected motor mention in snippet: %q", snippet)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
