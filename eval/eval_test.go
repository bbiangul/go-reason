package eval

import (
	"strings"
	"testing"
	"time"

	"github.com/bbiangul/go-reason"
)

func TestEasyDataset(t *testing.T) {
	ds := EasyDataset()
	if ds.Name == "" {
		t.Error("EasyDataset name is empty")
	}
	if ds.Difficulty != "easy" {
		t.Errorf("EasyDataset difficulty: got %q, want %q", ds.Difficulty, "easy")
	}
	if len(ds.Tests) == 0 {
		t.Fatal("EasyDataset has no tests")
	}
	for i, tc := range ds.Tests {
		if tc.Question == "" {
			t.Errorf("EasyDataset test %d has empty question", i)
		}
		if len(tc.ExpectedFacts) == 0 {
			t.Errorf("EasyDataset test %d has no expected facts", i)
		}
		if tc.Category == "" {
			t.Errorf("EasyDataset test %d has empty category", i)
		}
	}
}

func TestMediumDataset(t *testing.T) {
	ds := MediumDataset()
	if ds.Name == "" {
		t.Error("MediumDataset name is empty")
	}
	if ds.Difficulty != "medium" {
		t.Errorf("MediumDataset difficulty: got %q, want %q", ds.Difficulty, "medium")
	}
	if len(ds.Tests) == 0 {
		t.Fatal("MediumDataset has no tests")
	}
	for i, tc := range ds.Tests {
		if tc.Question == "" {
			t.Errorf("MediumDataset test %d has empty question", i)
		}
		if len(tc.ExpectedFacts) == 0 {
			t.Errorf("MediumDataset test %d has no expected facts", i)
		}
	}
}

func TestComplexDataset(t *testing.T) {
	ds := ComplexDataset()
	if ds.Name == "" {
		t.Error("ComplexDataset name is empty")
	}
	if ds.Difficulty != "complex" {
		t.Errorf("ComplexDataset difficulty: got %q, want %q", ds.Difficulty, "complex")
	}
	if len(ds.Tests) == 0 {
		t.Fatal("ComplexDataset has no tests")
	}
	for i, tc := range ds.Tests {
		if tc.Question == "" {
			t.Errorf("ComplexDataset test %d has empty question", i)
		}
		if len(tc.ExpectedFacts) == 0 {
			t.Errorf("ComplexDataset test %d has no expected facts", i)
		}
	}
}

func TestComputeAccuracy(t *testing.T) {
	tests := []struct {
		name          string
		answerText    string
		expectedFacts []string
		wantAccuracy  float64
	}{
		{
			name:          "all facts found",
			answerText:    "The tensile strength in section 3.2 is 500 MPa.",
			expectedFacts: []string{"tensile strength", "section 3.2"},
			wantAccuracy:  1.0,
		},
		{
			name:          "some facts found",
			answerText:    "The tensile strength is defined in the document.",
			expectedFacts: []string{"tensile strength", "section 3.2", "500 MPa"},
			wantAccuracy:  1.0 / 3.0,
		},
		{
			name:          "no facts found",
			answerText:    "The document does not contain relevant information.",
			expectedFacts: []string{"tensile strength", "section 3.2"},
			wantAccuracy:  0.0,
		},
		{
			name:          "case insensitive match",
			answerText:    "The TENSILE STRENGTH is specified.",
			expectedFacts: []string{"tensile strength"},
			wantAccuracy:  1.0,
		},
		{
			name:          "empty facts",
			answerText:    "Some answer.",
			expectedFacts: nil,
			wantAccuracy:  0.0,
		},
		{
			name:          "nil answer",
			answerText:    "",
			expectedFacts: []string{"fact"},
			wantAccuracy:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var answer *goreason.Answer
			if tt.answerText != "" || tt.name == "nil answer" {
				answer = &goreason.Answer{Text: tt.answerText}
			}
			if tt.name == "nil answer" {
				answer = &goreason.Answer{Text: ""}
			}

			accuracy := computeAccuracy(answer, tt.expectedFacts)

			const eps = 0.01
			if diff := accuracy - tt.wantAccuracy; diff < -eps || diff > eps {
				t.Errorf("accuracy: got %f, want %f", accuracy, tt.wantAccuracy)
			}
		})
	}
}

func TestComputeAccuracyNilAnswer(t *testing.T) {
	accuracy := computeAccuracy(nil, []string{"fact"})
	if accuracy != 0.0 {
		t.Errorf("expected 0 for nil answer, got %f", accuracy)
	}
}

func TestComputeFaithfulness(t *testing.T) {
	tests := []struct {
		name     string
		answer   *goreason.Answer
		minScore float64
		maxScore float64
	}{
		{
			name:     "nil answer",
			answer:   nil,
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name:     "empty text",
			answer:   &goreason.Answer{Text: ""},
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name: "faithful answer with source reference",
			answer: &goreason.Answer{
				Text: "According to spec-doc.pdf, the requirement is 500 MPa.",
				Sources: []goreason.Source{
					{Filename: "spec-doc.pdf", Content: "500 MPa requirement"},
				},
			},
			minScore: 1.0,
			maxScore: 1.0,
		},
		{
			name: "unfaithful answer with external knowledge",
			answer: &goreason.Answer{
				Text: "Based on my knowledge, typically the value is 500 MPa. As everyone knows this is standard.",
			},
			minScore: 0.0,
			maxScore: 0.7,
		},
		{
			name: "answer with in general phrase",
			answer: &goreason.Answer{
				Text: "In general, the tensile strength should be 500 MPa.",
			},
			minScore: 0.5,
			maxScore: 0.9,
		},
		{
			name: "multiple external indicators",
			answer: &goreason.Answer{
				Text: "Based on my knowledge, in general, it is commonly known that from my understanding the value is 500.",
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeFaithfulness(tt.answer)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("faithfulness: got %f, want between %f and %f",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestComputeRelevance(t *testing.T) {
	tests := []struct {
		name     string
		answer   *goreason.Answer
		question string
		minScore float64
		maxScore float64
	}{
		{
			name:     "nil answer",
			answer:   nil,
			question: "What is the tensile strength?",
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name: "no sources",
			answer: &goreason.Answer{
				Text:    "The answer is 500 MPa.",
				Sources: nil,
			},
			question: "What is the tensile strength?",
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name: "relevant sources",
			answer: &goreason.Answer{
				Text: "500 MPa",
				Sources: []goreason.Source{
					{Content: "The tensile strength requirement is 500 MPa.", Heading: "Material Specs"},
				},
			},
			question: "What is the tensile strength requirement?",
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name: "irrelevant sources",
			answer: &goreason.Answer{
				Text: "Something",
				Sources: []goreason.Source{
					{Content: "The weather is nice today.", Heading: "Weather Report"},
				},
			},
			question: "What is the tensile strength?",
			minScore: 0.0,
			maxScore: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeRelevance(tt.answer, tt.question)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("relevance: got %f, want between %f and %f",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestFormatReport(t *testing.T) {
	report := &Report{
		Dataset:    "Test Dataset",
		TotalTests: 3,
		Passed:     2,
		Failed:     1,
		Metrics: AggregateMetrics{
			AvgFaithfulness:    0.85,
			AvgRelevance:       0.75,
			AvgAccuracy:        0.90,
			AvgCitationQuality: 0.70,
			AvgConfidence:      0.80,
		},
		Results: []TestResult{
			{
				Question:      "What is the tensile strength?",
				ExpectedFacts: []string{"tensile strength"},
				Answer:        "500 MPa",
				Confidence:    0.9,
				Faithfulness:  0.95,
				Relevance:     0.85,
				Accuracy:      1.0,
				CitationQuality: 0.8,
				Passed:        true,
			},
			{
				Question:      "Who signed the contract?",
				ExpectedFacts: []string{"John Smith"},
				Answer:        "John Smith signed the contract.",
				Confidence:    0.8,
				Faithfulness:  0.9,
				Relevance:     0.7,
				Accuracy:      1.0,
				CitationQuality: 0.7,
				Passed:        true,
			},
			{
				Question:      "What is the effective date?",
				ExpectedFacts: []string{"January 1, 2025"},
				Answer:        "",
				Confidence:    0.1,
				Faithfulness:  0.0,
				Relevance:     0.0,
				Accuracy:      0.0,
				CitationQuality: 0.0,
				Passed:        false,
				Error:         "no results found",
			},
		},
		RunTime: 5 * time.Second,
	}

	output := FormatReport(report)

	// Verify key elements are present in the report.
	checks := []string{
		"Test Dataset",
		"Total: 3",
		"Passed: 2",
		"Failed: 1",
		"Faithfulness",
		"Relevance",
		"Accuracy",
		"Citation Quality",
		"Claim Grounding",
		"Hallucination Score",
		"Confidence",
		"[PASS]",
		"[FAIL]",
		"no results found",
		"Grnd=",
		"Hall=",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("report missing %q in output:\n%s", check, output)
		}
	}
}

func TestFormatReportEmpty(t *testing.T) {
	report := &Report{
		Dataset:    "Empty",
		TotalTests: 0,
	}

	output := FormatReport(report)
	if !strings.Contains(output, "Empty") {
		t.Error("expected dataset name in empty report")
	}
	if !strings.Contains(output, "Total: 0") {
		t.Error("expected Total: 0 in empty report")
	}
}

func TestPDFComplexityReport(t *testing.T) {
	results := []PDFComplexityResult{
		{
			Path:            "/docs/simple.pdf",
			ExpectedComplex: false,
			DetectedComplex: false,
			Score:           0.1,
			Correct:         true,
			Details:         "",
		},
		{
			Path:            "/docs/complex.pdf",
			ExpectedComplex: true,
			DetectedComplex: true,
			Score:           0.8,
			Correct:         true,
			Details:         "tables detected, images detected",
		},
		{
			Path:            "/docs/misdetected.pdf",
			ExpectedComplex: true,
			DetectedComplex: false,
			Score:           0.3,
			Correct:         false,
			Details:         "multi-column detected",
		},
	}

	report := PDFComplexityReport(results)

	// Verify structure.
	if !strings.Contains(report, "PDF Complexity Detection Evaluation") {
		t.Error("report missing title")
	}
	if !strings.Contains(report, "[CORRECT]") {
		t.Error("report missing CORRECT status")
	}
	if !strings.Contains(report, "[WRONG]") {
		t.Error("report missing WRONG status")
	}
	if !strings.Contains(report, "Accuracy: 2/3") {
		t.Errorf("report missing or incorrect accuracy, got:\n%s", report)
	}
	if !strings.Contains(report, "66.7%") {
		t.Errorf("report missing percentage, got:\n%s", report)
	}
	if !strings.Contains(report, "tables detected, images detected") {
		t.Error("report missing details for complex PDF")
	}
}

func TestPDFComplexityReportEmpty(t *testing.T) {
	report := PDFComplexityReport(nil)
	if !strings.Contains(report, "Accuracy: 0/0") {
		t.Errorf("expected 0/0 accuracy for empty results, got:\n%s", report)
	}
}

func TestPDFComplexityReportAllCorrect(t *testing.T) {
	results := []PDFComplexityResult{
		{Path: "a.pdf", Correct: true},
		{Path: "b.pdf", Correct: true},
	}

	report := PDFComplexityReport(results)
	if !strings.Contains(report, "100.0%") {
		t.Errorf("expected 100%% accuracy, got:\n%s", report)
	}
}

func TestSignificantWords(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "filters stop words",
			text:     "What is the tensile strength of the material?",
			expected: []string{"tensile", "strength", "material"},
		},
		{
			name:     "short words removed",
			text:     "a to be or",
			expected: nil,
		},
		{
			name:     "preserves significant words",
			text:     "ISO 9001 quality management system compliance",
			expected: []string{"iso", "9001", "quality", "management", "system", "compliance"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words := significantWords(tt.text)

			if tt.expected == nil {
				if len(words) != 0 {
					t.Errorf("expected no words, got %v", words)
				}
				return
			}

			for _, exp := range tt.expected {
				found := false
				for _, w := range words {
					if w == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected word %q in result %v", exp, words)
				}
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{-0.5, 0.0},
		{0.0, 0.0},
		{0.5, 0.5},
		{1.0, 1.0},
		{1.5, 1.0},
	}

	for _, tt := range tests {
		result := clamp(tt.input)
		if result != tt.expected {
			t.Errorf("clamp(%f): got %f, want %f", tt.input, result, tt.expected)
		}
	}
}

func TestComputeClaimGrounding(t *testing.T) {
	tests := []struct {
		name     string
		answer   *goreason.Answer
		minScore float64
		maxScore float64
	}{
		{
			name:     "nil answer",
			answer:   nil,
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name:     "empty text",
			answer:   &goreason.Answer{Text: ""},
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name: "no sources",
			answer: &goreason.Answer{
				Text: "The pressure is 120 PSI.",
			},
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name: "all terms grounded",
			answer: &goreason.Answer{
				Text: "The pressure requirement is 120 PSI at section 3.2.",
				Sources: []goreason.Source{
					{Content: "The pressure requirement is 120 PSI as specified in section 3.2."},
				},
			},
			minScore: 0.8,
			maxScore: 1.0,
		},
		{
			name: "no terms grounded",
			answer: &goreason.Answer{
				Text: "The temperature is 450 degrees Celsius.",
				Sources: []goreason.Source{
					{Content: "The humidity level should be maintained at 60%."},
				},
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
		{
			name: "partial grounding",
			answer: &goreason.Answer{
				Text: "The pressure is 120 PSI and the voltage is 999 volts.",
				Sources: []goreason.Source{
					{Content: "pressure requirement: 120 PSI"},
				},
			},
			minScore: 0.2,
			maxScore: 0.7,
		},
		{
			name: "numbers in sources counted",
			answer: &goreason.Answer{
				Text: "Values are 153 and 300.",
				Sources: []goreason.Source{
					{Content: "model weighs 153 kg"},
					{Content: "XL model weighs 300 kg"},
				},
			},
			// terms: "values" (ungrounded), "153" (grounded), "300" (grounded) → 2/3
			minScore: 0.6,
			maxScore: 0.7,
		},
		{
			name: "heading content also searched",
			answer: &goreason.Answer{
				Text: "The calibration process uses pixels.",
				Sources: []goreason.Source{
					{Content: "some other content", Heading: "Calibration process with pixels"},
				},
			},
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name: "no significant terms returns 1.0",
			answer: &goreason.Answer{
				Text: "It is so.",
				Sources: []goreason.Source{
					{Content: "something"},
				},
			},
			minScore: 1.0,
			maxScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeClaimGrounding(tt.answer)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("claimGrounding: got %f, want between %f and %f",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestComputeHallucinationScore(t *testing.T) {
	tests := []struct {
		name     string
		answer   *goreason.Answer
		minScore float64
		maxScore float64
	}{
		{
			name:     "nil answer",
			answer:   nil,
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name:     "empty text",
			answer:   &goreason.Answer{Text: ""},
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name: "no sources returns neutral 0.5",
			answer: &goreason.Answer{
				Text: "The pressure is 120 PSI.",
			},
			minScore: 0.49,
			maxScore: 0.51,
		},
		{
			name: "all numbers grounded - clean",
			answer: &goreason.Answer{
				Text: "The weight is 153 kg and the pressure is 120 PSI.",
				Sources: []goreason.Source{
					{Content: "weight: 153 kg, pressure: 120 PSI"},
				},
			},
			minScore: 0.8,
			maxScore: 1.0,
		},
		{
			name: "fabricated numbers - hallucinated",
			answer: &goreason.Answer{
				Text: "The weight is 999 kg and the voltage is 4500 volts.",
				Sources: []goreason.Source{
					{Content: "The system operates within normal parameters."},
				},
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
		{
			name: "trivial numbers ignored",
			answer: &goreason.Answer{
				Text: "There are 3 items and 5 units.",
				Sources: []goreason.Source{
					{Content: "The process has multiple stages."},
				},
			},
			// Numbers 3 and 5 are trivial (skipped). Words "items" and "units"
			// are <=5 chars so also skipped. No checkable terms → 1.0.
			minScore: 1.0,
			maxScore: 1.0,
		},
		{
			name: "mixed grounded and ungrounded",
			answer: &goreason.Answer{
				Text: "The pressure is 120 PSI but the temperature is 9999 degrees.",
				Sources: []goreason.Source{
					{Content: "pressure: 120 PSI"},
				},
			},
			minScore: 0.2,
			maxScore: 0.7,
		},
		{
			name: "long words checked against sources",
			answer: &goreason.Answer{
				Text: "The calibration process requires specific parameters.",
				Sources: []goreason.Source{
					{Content: "calibration process parameters specification"},
				},
			},
			minScore: 0.7,
			maxScore: 1.0,
		},
		{
			name: "ungrounded long words penalized less than numbers",
			answer: &goreason.Answer{
				Text: "The fabricated specification requires attention.",
				Sources: []goreason.Source{
					{Content: "normal system operation"},
				},
			},
			// Words have 0.5 penalty weight vs 1.0 for numbers,
			// so even fully ungrounded words give some residual score.
			minScore: 0.0,
			maxScore: 0.1,
		},
		{
			name: "no checkable terms returns 1.0",
			answer: &goreason.Answer{
				Text: "Yes it is.",
				Sources: []goreason.Source{
					{Content: "something"},
				},
			},
			minScore: 1.0,
			maxScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeHallucinationScore(tt.answer)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("hallucinationScore: got %f, want between %f and %f",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestComputeCitationQuality(t *testing.T) {
	tests := []struct {
		name     string
		answer   *goreason.Answer
		minScore float64
		maxScore float64
	}{
		{
			name:     "nil answer",
			answer:   nil,
			minScore: 0.0,
			maxScore: 0.01,
		},
		{
			name: "answer with section and page citations",
			answer: &goreason.Answer{
				Text: "See section 3.2 on page 5 for details.",
				Sources: []goreason.Source{
					{Filename: "doc.pdf"},
				},
			},
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name: "answer with filename reference",
			answer: &goreason.Answer{
				Text: "According to doc.pdf, the requirement is stated clearly.",
				Sources: []goreason.Source{
					{Filename: "doc.pdf"},
				},
			},
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name: "answer with no citation patterns",
			answer: &goreason.Answer{
				Text: "The requirement is 500 MPa.",
			},
			minScore: 0.4,
			maxScore: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeCitationQuality(tt.answer)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("citationQuality: got %f, want between %f and %f",
					score, tt.minScore, tt.maxScore)
			}
		})
	}
}
