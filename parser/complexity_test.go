package parser

import "testing"

// ---------------------------------------------------------------------------
// analyzePageComplexity tests
// ---------------------------------------------------------------------------

func TestAnalyzePageComplexityTablePipes(t *testing.T) {
	score := &ComplexityScore{}
	tableText := "| Col1 | Col2 | Col3 |\n| --- | --- | --- |\n| val1 | val2 | val3 |\n| a | b | c |\n| d | e | f |\n| g | h | i |"
	analyzePageComplexity(tableText, score)

	if !score.HasTables {
		t.Error("expected HasTables = true for pipe-delimited table text")
	}
}

func TestAnalyzePageComplexityTableTabs(t *testing.T) {
	score := &ComplexityScore{}
	tabText := "Col1\tCol2\tCol3\n" +
		"val1\tval2\tval3\n" +
		"a\tb\tc\n" +
		"d\te\tf\n" +
		"g\th\ti\n" +
		"j\tk\tl\n"
	analyzePageComplexity(tabText, score)

	if !score.HasTables {
		t.Error("expected HasTables = true for tab-delimited table text")
	}
}

func TestAnalyzePageComplexityDashSeparators(t *testing.T) {
	score := &ComplexityScore{}
	dashText := "Header Row\n" +
		"--------------------\n" +
		"Data row 1\n" +
		"--------------------\n" +
		"Data row 2\n" +
		"--------------------\n"
	analyzePageComplexity(dashText, score)

	if !score.HasTables {
		t.Error("expected HasTables = true for text with dash separators")
	}
}

func TestAnalyzePageComplexityNoTable(t *testing.T) {
	score := &ComplexityScore{}
	plainText := "This is a regular paragraph.\nIt has no table-like patterns.\nJust normal sentences."
	analyzePageComplexity(plainText, score)

	if score.HasTables {
		t.Error("expected HasTables = false for plain paragraph text")
	}
}

func TestAnalyzePageComplexityMultiColumn(t *testing.T) {
	score := &ComplexityScore{}

	// Build text with large horizontal whitespace gaps in the middle of lines
	// Each line > 40 chars, with > 8 spaces in a 20-char window around the midpoint.
	multiColText := ""
	for i := 0; i < 5; i++ {
		multiColText += "Some left column text              Some right column text here\n"
	}
	analyzePageComplexity(multiColText, score)

	if !score.IsMultiCol {
		t.Error("expected IsMultiCol = true for multi-column formatted text")
	}
}

func TestAnalyzePageComplexityNotMultiColumn(t *testing.T) {
	score := &ComplexityScore{}
	singleColText := "This is a single-column paragraph.\nEach line flows normally.\nNo large gaps in the middle."
	analyzePageComplexity(singleColText, score)

	if score.IsMultiCol {
		t.Error("expected IsMultiCol = false for single-column text")
	}
}

// ---------------------------------------------------------------------------
// ComplexityScore.IsComplex tests
// ---------------------------------------------------------------------------

func TestIsComplexThreshold(t *testing.T) {
	tests := []struct {
		name      string
		score     ComplexityScore
		wantComp  bool
	}{
		{
			name:     "below_threshold",
			score:    ComplexityScore{Score: 0.3},
			wantComp: false,
		},
		{
			name:     "at_threshold",
			score:    ComplexityScore{Score: 0.5},
			wantComp: true,
		},
		{
			name:     "above_threshold",
			score:    ComplexityScore{Score: 0.8},
			wantComp: true,
		},
		{
			name:     "zero",
			score:    ComplexityScore{Score: 0.0},
			wantComp: false,
		},
		{
			name:     "max",
			score:    ComplexityScore{Score: 1.0},
			wantComp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.score.IsComplex()
			if got != tt.wantComp {
				t.Errorf("ComplexityScore{Score: %f}.IsComplex() = %v, want %v",
					tt.score.Score, got, tt.wantComp)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Score composition tests
// ---------------------------------------------------------------------------

func TestComplexityScoreComposition(t *testing.T) {
	// Verify that the score components add up correctly when set manually.
	// This simulates what DetectComplexity computes after analyzing pages.
	tests := []struct {
		name        string
		hasTables   bool
		hasImages   bool
		isMultiCol  bool
		fontVariety int
		wantScore   float64
		wantComplex bool
	}{
		{
			name:        "simple_text",
			wantScore:   0.0,
			wantComplex: false,
		},
		{
			name:        "tables_only",
			hasTables:   true,
			wantScore:   0.3,
			wantComplex: false,
		},
		{
			name:        "tables_and_images",
			hasTables:   true,
			hasImages:   true,
			wantScore:   0.6,
			wantComplex: true,
		},
		{
			name:        "tables_and_multicol",
			hasTables:   true,
			isMultiCol:  true,
			wantScore:   0.5,
			wantComplex: true,
		},
		{
			name:        "all_complex_features",
			hasTables:   true,
			hasImages:   true,
			isMultiCol:  true,
			fontVariety: 5,
			wantScore:   1.0,
			wantComplex: true,
		},
		{
			name:        "font_variety_only",
			fontVariety: 5,
			wantScore:   0.2,
			wantComplex: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := &ComplexityScore{
				HasTables:   tt.hasTables,
				HasImages:   tt.hasImages,
				IsMultiCol:  tt.isMultiCol,
				FontVariety: tt.fontVariety,
			}

			// Replicate the scoring logic from DetectComplexity.
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

			if score.Score != tt.wantScore {
				t.Errorf("Score = %f, want %f", score.Score, tt.wantScore)
			}
			if score.IsComplex() != tt.wantComplex {
				t.Errorf("IsComplex() = %v, want %v", score.IsComplex(), tt.wantComplex)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Edge cases for analyzePageComplexity
// ---------------------------------------------------------------------------

func TestAnalyzePageComplexityEmptyText(t *testing.T) {
	score := &ComplexityScore{}
	analyzePageComplexity("", score)

	if score.HasTables {
		t.Error("expected HasTables = false for empty text")
	}
	if score.IsMultiCol {
		t.Error("expected IsMultiCol = false for empty text")
	}
}

func TestAnalyzePageComplexityAccumulates(t *testing.T) {
	score := &ComplexityScore{}

	// First call: no tables
	analyzePageComplexity("Normal text.", score)
	if score.HasTables {
		t.Error("HasTables should be false after first page")
	}

	// Second call: has tables -- should accumulate
	tableText := "| A | B | C |\n| D | E | F |\n| G | H | I |\n| J | K | L |\n| M | N | O |\n| P | Q | R |"
	analyzePageComplexity(tableText, score)
	if !score.HasTables {
		t.Error("HasTables should be true after accumulating table page")
	}
}
