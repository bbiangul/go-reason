package eval

import (
	"fmt"
	"strings"

	"github.com/bbiangul/go-reason/parser"
)

// PDFComplexityResult holds the evaluation of PDF complexity detection.
type PDFComplexityResult struct {
	Path           string  `json:"path"`
	ExpectedComplex bool   `json:"expected_complex"`
	DetectedComplex bool   `json:"detected_complex"`
	Score          float64 `json:"score"`
	Correct        bool    `json:"correct"`
	Details        string  `json:"details"`
}

// PDFComplexityTestCase defines a test for the complexity detector.
type PDFComplexityTestCase struct {
	Path            string `json:"path"`
	ExpectedComplex bool   `json:"expected_complex"`
	Description     string `json:"description"`
}

// EvaluatePDFComplexity tests the PDF complexity detector against known files.
func EvaluatePDFComplexity(testCases []PDFComplexityTestCase) []PDFComplexityResult {
	var results []PDFComplexityResult

	for _, tc := range testCases {
		result := PDFComplexityResult{
			Path:            tc.Path,
			ExpectedComplex: tc.ExpectedComplex,
		}

		score, err := parser.DetectComplexity(tc.Path)
		if err != nil {
			result.Details = fmt.Sprintf("error: %v", err)
			results = append(results, result)
			continue
		}

		result.DetectedComplex = score.IsComplex()
		result.Score = score.Score
		result.Correct = result.ExpectedComplex == result.DetectedComplex

		var details []string
		if score.HasTables {
			details = append(details, "tables detected")
		}
		if score.HasImages {
			details = append(details, "images detected")
		}
		if score.IsMultiCol {
			details = append(details, "multi-column detected")
		}
		result.Details = strings.Join(details, ", ")

		results = append(results, result)
	}

	return results
}

// PDFComplexityReport summarizes PDF complexity evaluation results.
func PDFComplexityReport(results []PDFComplexityResult) string {
	var b strings.Builder
	correct := 0
	total := len(results)

	fmt.Fprintf(&b, "=== PDF Complexity Detection Evaluation ===\n")
	for i, r := range results {
		status := "CORRECT"
		if !r.Correct {
			status = "WRONG"
		}
		if r.Correct {
			correct++
		}
		fmt.Fprintf(&b, "[%s] %d. %s (score=%.2f, expected_complex=%v, detected_complex=%v)\n",
			status, i+1, r.Path, r.Score, r.ExpectedComplex, r.DetectedComplex)
		if r.Details != "" {
			fmt.Fprintf(&b, "  Details: %s\n", r.Details)
		}
	}

	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correct) / float64(total) * 100
	}
	fmt.Fprintf(&b, "\nAccuracy: %d/%d (%.1f%%)\n", correct, total, accuracy)

	return b.String()
}
