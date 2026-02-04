package parser

import (
	"strings"

	"github.com/ledongthuc/pdf"
)

// ComplexityScore represents the structural complexity of a PDF page.
type ComplexityScore struct {
	HasTables   bool
	HasImages   bool
	IsMultiCol  bool
	FontVariety int     // number of distinct fonts
	Score       float64 // 0.0 = simple text, 1.0 = highly complex
}

// DetectComplexity analyzes a PDF file for structural complexity.
func DetectComplexity(path string) (*ComplexityScore, error) {
	f, reader, err := pdf.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	score := &ComplexityScore{}
	totalPages := reader.NumPage()

	for i := 1; i <= totalPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		analyzePageComplexity(text, score)
	}

	// Compute overall score
	s := 0.0
	if score.HasTables {
		s += 0.3
	}
	if score.HasImages {
		s += 0.3
	}
	if score.IsMultiCol {
		s += 0.2
	}
	if score.FontVariety > 3 {
		s += 0.2
	}
	score.Score = s

	return score, nil
}

// IsComplex returns true if the PDF should be routed to vision processing.
func (cs *ComplexityScore) IsComplex() bool {
	return cs.Score >= 0.5
}

func analyzePageComplexity(text string, score *ComplexityScore) {
	lines := strings.Split(text, "\n")

	// Table detection: look for grid-like patterns
	tabCount := 0
	pipeCount := 0
	dashLineCount := 0
	for _, line := range lines {
		tabCount += strings.Count(line, "\t")
		pipeCount += strings.Count(line, "|")
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 3 && (strings.Count(trimmed, "-") > len(trimmed)/2 || strings.Count(trimmed, "_") > len(trimmed)/2) {
			dashLineCount++
		}
	}
	if tabCount > 5 || pipeCount > 5 || dashLineCount > 2 {
		score.HasTables = true
	}

	// Multi-column detection: look for large horizontal whitespace gaps mid-line
	multiColIndicators := 0
	for _, line := range lines {
		if len(line) > 40 && strings.Contains(line, "    ") {
			// Check if there's a significant gap in the middle
			mid := len(line) / 2
			start := mid - 10
			end := mid + 10
			if start < 0 {
				start = 0
			}
			if end > len(line) {
				end = len(line)
			}
			midSection := line[start:end]
			if strings.Count(midSection, " ") > 8 {
				multiColIndicators++
			}
		}
	}
	if multiColIndicators > 3 {
		score.IsMultiCol = true
	}
}
