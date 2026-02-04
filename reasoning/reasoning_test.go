package reasoning

import (
	"strings"
	"testing"

	"github.com/bbiangul/go-reason/store"
)

// testChunks returns a slice of RetrievalResult for use in tests.
func testChunks() []store.RetrievalResult {
	return []store.RetrievalResult{
		{
			ChunkID:    1,
			DocumentID: 100,
			Content:    "The tensile strength shall be at least 500 MPa as specified in section 3.2.",
			Heading:    "Material Requirements",
			ChunkType:  "text",
			PageNumber: 5,
			Filename:   "spec-doc.pdf",
			Path:       "/docs/spec-doc.pdf",
			Score:      0.95,
		},
		{
			ChunkID:    2,
			DocumentID: 100,
			Content:    "All materials must comply with ISO 9001 quality management standards.",
			Heading:    "Quality Standards",
			ChunkType:  "text",
			PageNumber: 8,
			Filename:   "spec-doc.pdf",
			Path:       "/docs/spec-doc.pdf",
			Score:      0.88,
		},
		{
			ChunkID:    3,
			DocumentID: 101,
			Content:    "The contractor shall perform risk assessment per ISO 31000 guidelines.",
			Heading:    "Risk Management",
			ChunkType:  "text",
			PageNumber: 12,
			Filename:   "contract.pdf",
			Path:       "/docs/contract.pdf",
			Score:      0.75,
		},
	}
}

func TestValidation(t *testing.T) {
	chunks := testChunks()

	tests := []struct {
		name              string
		answer            string
		wantCitationValid bool
		wantConsistValid  bool
	}{
		{
			name:              "answer referencing a source",
			answer:            "According to spec-doc.pdf, the tensile strength must be at least 500 MPa.",
			wantCitationValid: true,
			wantConsistValid:  true,
		},
		{
			name:              "answer referencing heading",
			answer:            "The Material Requirements section specifies 500 MPa tensile strength.",
			wantCitationValid: true,
			wantConsistValid:  true,
		},
		{
			name:              "answer with no source references",
			answer:            "The tensile strength is 500 MPa.",
			wantCitationValid: false,
			wantConsistValid:  true,
		},
		{
			name:              "answer with fabricated reference",
			answer:            "According to some unknown document, the value is 500 MPa. As stated in a report that does not exist.",
			wantCitationValid: false,
			wantConsistValid:  true,
		},
		{
			name:              "answer using external knowledge",
			answer:            "Based on my knowledge, the standard requirement is 500 MPa.",
			wantCitationValid: false,
			wantConsistValid:  false,
		},
		{
			name:              "answer with contradiction language",
			answer:            "The document states 500 MPa. However, the document says the opposite about this requirement.",
			wantCitationValid: false,
			wantConsistValid:  false,
		},
		{
			name:              "answer with commonly known pattern",
			answer:            "It is commonly known that tensile strength should be 500 MPa as in spec-doc.pdf.",
			wantCitationValid: true,
			wantConsistValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validate(tt.answer, chunks)

			if result.citationValid != tt.wantCitationValid {
				t.Errorf("citationValid: got %v, want %v (issues: %v)",
					result.citationValid, tt.wantCitationValid, result.citationIssues)
			}
			if result.consistencyValid != tt.wantConsistValid {
				t.Errorf("consistencyValid: got %v, want %v (issues: %v)",
					result.consistencyValid, tt.wantConsistValid, result.consistencyIssues)
			}
		})
	}
}

func TestCitationExtraction(t *testing.T) {
	chunks := testChunks()

	tests := []struct {
		name      string
		answer    string
		wantCount int
		wantRefs  []string
	}{
		{
			name:      "document filename citation",
			answer:    "As noted in (spec-doc.pdf, section 3.2), the value is 500 MPa.",
			wantCount: 1, // the (spec-doc.pdf...) captures the whole parenthetical
			wantRefs:  []string{"spec-doc.pdf"},
		},
		{
			name:      "section reference",
			answer:    "Section 3.2 specifies the tensile requirements.",
			wantCount: 1,
			wantRefs:  []string{"3.2"},
		},
		{
			name:      "article reference",
			answer:    "Article 5 of the contract outlines obligations.",
			wantCount: 1,
			wantRefs:  []string{"5"},
		},
		{
			name:      "clause reference",
			answer:    "Clause 7.1 requires annual reviews.",
			wantCount: 1,
			wantRefs:  []string{"7.1"},
		},
		{
			name:      "page reference",
			answer:    "See Page 5 for material specifications.",
			wantCount: 1,
			wantRefs:  []string{"5"},
		},
		{
			name:      "source number reference",
			answer:    "The answer is found in [Source 1] which states the requirement.",
			wantCount: 1,
			wantRefs:  []string{"1"},
		},
		{
			name:      "no citations",
			answer:    "The requirement is 500 MPa for all materials.",
			wantCount: 0,
			wantRefs:  nil,
		},
		{
			name:      "multiple citation types",
			answer:    "Per (spec-doc.pdf), Section 3.2, Clause 7.1 outlines the requirements.",
			wantCount: 3,
			wantRefs:  []string{"spec-doc.pdf", "3.2", "7.1"},
		},
		{
			name:      "section abbreviation",
			answer:    "Sec. 4.1 provides additional details.",
			wantCount: 1,
			wantRefs:  []string{"4.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			citations := ExtractCitations(tt.answer, chunks)

			if len(citations) != tt.wantCount {
				t.Errorf("citation count: got %d, want %d; citations: %+v",
					len(citations), tt.wantCount, citations)
			}

			for _, ref := range tt.wantRefs {
				found := false
				for _, c := range citations {
					if strings.Contains(c.SourceRef, ref) || strings.Contains(c.Text, ref) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected citation referencing %q in %+v", ref, citations)
				}
			}
		})
	}
}

func TestCitationVerification(t *testing.T) {
	chunks := testChunks()

	// Citation referencing a real filename should be verified.
	citations := ExtractCitations("See (spec-doc.pdf) for details.", chunks)
	if len(citations) == 0 {
		t.Fatal("expected at least one citation")
	}
	if !citations[0].Verified {
		t.Error("expected citation to spec-doc.pdf to be verified")
	}

	// Citation with page number matching a chunk should be verified.
	pageCitations := ExtractCitations("See Page 5 for details.", chunks)
	if len(pageCitations) == 0 {
		t.Fatal("expected page citation")
	}
	if !pageCitations[0].Verified {
		t.Error("expected Page 5 citation to be verified (matches chunk page_number)")
	}
}

func TestConfidenceScoring(t *testing.T) {
	chunks := testChunks()
	weights := DefaultConfidenceWeights()

	tests := []struct {
		name    string
		answer  string
		minConf float64
		maxConf float64
	}{
		{
			name:    "well-cited answer",
			answer:  "According to spec-doc.pdf, Section 3.2 under Material Requirements, the tensile strength is at least 500 MPa. This is also confirmed by the Quality Standards section which references ISO 9001 compliance.",
			minConf: 0.4,
			maxConf: 1.0,
		},
		{
			name:    "uncertain answer",
			answer:  "I'm not sure about this. It's unclear from the provided documents. Cannot determine the exact requirement.",
			minConf: 0.0,
			maxConf: 0.5,
		},
		{
			name:    "contradictory answer",
			answer:  "The requirement is 500 MPa. However, it also states the opposite requirement of 300 MPa, which contradicts the earlier statement.",
			minConf: 0.0,
			maxConf: 0.7,
		},
		{
			name:    "empty answer",
			answer:  "",
			minConf: 0.0,
			maxConf: 0.5,
		},
		{
			name:    "very short answer",
			answer:  "500 MPa",
			minConf: 0.0,
			maxConf: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := ComputeConfidence(tt.answer, chunks, weights)

			if conf < tt.minConf || conf > tt.maxConf {
				t.Errorf("confidence: got %f, want between %f and %f",
					conf, tt.minConf, tt.maxConf)
			}
		})
	}
}

func TestConfidenceWeightsDefault(t *testing.T) {
	w := DefaultConfidenceWeights()

	sum := w.SourceCoverage + w.CitationAccuracy + w.SelfConsistency + w.AnswerLength
	if diff := sum - 1.0; diff < -0.01 || diff > 0.01 {
		t.Errorf("default weights should sum to 1.0, got %f", sum)
	}
}

func TestComputeConfidenceEmptyChunks(t *testing.T) {
	weights := DefaultConfidenceWeights()
	conf := ComputeConfidence("Some answer text here for testing purposes.", nil, weights)

	// With no chunks, source coverage and citation accuracy are 0/0.5.
	// Should still produce a valid score.
	if conf < 0 || conf > 1 {
		t.Errorf("confidence should be between 0 and 1, got %f", conf)
	}
}

func TestEstimateConfidence(t *testing.T) {
	chunks := testChunks()

	tests := []struct {
		name    string
		answer  string
		chunks  []store.RetrievalResult
		minConf float64
		maxConf float64
	}{
		{
			name:    "answer with source references",
			answer:  "According to spec-doc.pdf, the value is 500 MPa.",
			chunks:  chunks,
			minConf: 0.5,
			maxConf: 1.0,
		},
		{
			name:    "answer with heading reference",
			answer:  "The Material Requirements section states the requirement.",
			chunks:  chunks,
			minConf: 0.5,
			maxConf: 1.0,
		},
		{
			name:    "answer with hedging language",
			answer:  "This might possibly be the case, though it is unclear.",
			chunks:  chunks,
			minConf: 0.0,
			maxConf: 0.5,
		},
		{
			name:    "empty answer",
			answer:  "",
			chunks:  chunks,
			minConf: 0.0,
			maxConf: 0.01,
		},
		{
			name:    "empty chunks",
			answer:  "Some answer text.",
			chunks:  nil,
			minConf: 0.0,
			maxConf: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := estimateConfidence(tt.answer, tt.chunks)

			if conf < tt.minConf || conf > tt.maxConf {
				t.Errorf("estimateConfidence: got %f, want between %f and %f",
					conf, tt.minConf, tt.maxConf)
			}
		})
	}
}

func TestValidationResultConfidence(t *testing.T) {
	tests := []struct {
		name     string
		result   validationResult
		minConf  float64
		maxConf  float64
	}{
		{
			name: "all valid",
			result: validationResult{
				citationValid:     true,
				consistencyValid:  true,
				completenessValid: true,
			},
			minConf: 1.0,
			maxConf: 1.0,
		},
		{
			name: "citation issues",
			result: validationResult{
				citationValid:     false,
				citationIssues:    []string{"missing references"},
				consistencyValid:  true,
				completenessValid: true,
			},
			minConf: 0.8,
			maxConf: 0.9,
		},
		{
			name: "consistency issues",
			result: validationResult{
				citationValid:    true,
				consistencyValid: false,
				consistencyIssues: []string{"contradiction found"},
				completenessValid: true,
			},
			minConf: 0.7,
			maxConf: 0.9,
		},
		{
			name: "multiple issues",
			result: validationResult{
				citationValid:      false,
				citationIssues:     []string{"no refs", "fabricated ref"},
				consistencyValid:   false,
				consistencyIssues:  []string{"contradiction"},
				completenessValid:  false,
				completenessIssues: []string{"incomplete"},
			},
			minConf: 0.0,
			maxConf: 0.5,
		},
		{
			name: "many issues lower bound clamped",
			result: validationResult{
				citationValid:      false,
				citationIssues:     []string{"a", "b", "c", "d", "e", "f", "g"},
				consistencyValid:   false,
				consistencyIssues:  []string{"x", "y", "z"},
				completenessValid:  false,
				completenessIssues: []string{"1", "2", "3"},
			},
			minConf: 0.0,
			maxConf: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := tt.result.confidence()

			if conf < tt.minConf || conf > tt.maxConf {
				t.Errorf("confidence: got %f, want between %f and %f",
					conf, tt.minConf, tt.maxConf)
			}
		})
	}
}

func TestValidationResultSummary(t *testing.T) {
	t.Run("all passed", func(t *testing.T) {
		v := &validationResult{
			citationValid:     true,
			consistencyValid:  true,
			completenessValid: true,
		}
		summary := v.summary()
		if summary != "All validations passed." {
			t.Errorf("expected 'All validations passed.', got %q", summary)
		}
	})

	t.Run("citation issues", func(t *testing.T) {
		v := &validationResult{
			citationValid:     false,
			citationIssues:    []string{"no source references"},
			consistencyValid:  true,
			completenessValid: true,
		}
		summary := v.summary()
		if !strings.Contains(summary, "Citation issues") {
			t.Errorf("expected summary to contain 'Citation issues', got %q", summary)
		}
		if !strings.Contains(summary, "no source references") {
			t.Errorf("expected summary to contain issue text, got %q", summary)
		}
	})

	t.Run("multiple issue types", func(t *testing.T) {
		v := &validationResult{
			citationValid:      false,
			citationIssues:     []string{"missing refs"},
			consistencyValid:   false,
			consistencyIssues:  []string{"contradiction found"},
			completenessValid:  false,
			completenessIssues: []string{"incomplete analysis"},
		}
		summary := v.summary()
		if !strings.Contains(summary, "Citation issues") {
			t.Errorf("expected Citation issues in summary, got %q", summary)
		}
		if !strings.Contains(summary, "Consistency issues") {
			t.Errorf("expected Consistency issues in summary, got %q", summary)
		}
		if !strings.Contains(summary, "Completeness issues") {
			t.Errorf("expected Completeness issues in summary, got %q", summary)
		}
	})
}

func TestAnswerLengthScore(t *testing.T) {
	tests := []struct {
		name      string
		wordCount int
		expected  float64
	}{
		{"very short", 5, 0.2},
		{"short", 20, 0.5},
		{"medium", 60, 0.8},
		{"long", 200, 1.0},
		{"very long", 600, 0.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words := make([]string, tt.wordCount)
			for i := range words {
				words[i] = "word"
			}
			answer := strings.Join(words, " ")
			score := answerLengthScore(answer)
			if score != tt.expected {
				t.Errorf("answerLengthScore(%d words): got %f, want %f",
					tt.wordCount, score, tt.expected)
			}
		})
	}
}

func TestSelfConsistencyScore(t *testing.T) {
	tests := []struct {
		name    string
		answer  string
		minConf float64
		maxConf float64
	}{
		{
			name:    "consistent answer",
			answer:  "The requirement is clearly stated in the document.",
			minConf: 0.99,
			maxConf: 1.0,
		},
		{
			name:    "contradictory answer",
			answer:  "The value is 500 MPa. On the other hand, it contradicts the earlier specification.",
			minConf: 0.5,
			maxConf: 0.8,
		},
		{
			name:    "uncertain answer",
			answer:  "I'm not sure about this and cannot determine the exact value.",
			minConf: 0.3,
			maxConf: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := selfConsistencyScore(tt.answer)
			if score < tt.minConf || score > tt.maxConf {
				t.Errorf("selfConsistencyScore: got %f, want between %f and %f",
					score, tt.minConf, tt.maxConf)
			}
		})
	}
}
