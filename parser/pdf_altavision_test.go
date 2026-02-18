package parser

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ledongthuc/pdf"
)

// TestALTAVisionModelHeadings verifies that the PDF parser extracts section
// headings for equipment models on pages 32-38. These headings ("3.9.1 Modelo A:
// AV Cabezal Standard", etc.) are critical for associating weight/spec data with
// specific models.
func TestALTAVisionModelHeadings(t *testing.T) {
	pdfPath := os.Getenv("ALTAVISION_PDF")
	if pdfPath == "" {
		pdfPath = os.ExpandEnv("$HOME/Downloads/Manual Técnico ALTAVision AV-FM RevG02.1 (1).pdf")
	}
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skipf("PDF not found at %s — set ALTAVISION_PDF env var", pdfPath)
	}

	// --- Step 1: Check raw text extraction per page ---

	f, reader, err := pdf.Open(pdfPath)
	if err != nil {
		t.Fatalf("failed to open PDF: %v", err)
	}
	defer f.Close()

	// The model headings appear on these PDF pages (1-indexed):
	modelPages := []struct {
		page    int
		heading string // expected heading substring
		weight  string // expected weight substring
	}{
		{32, "Modelo A", "153"},
		{34, "ModeloB", "80 kg"}, // no space in actual PDF
		{36, "Modelo C", "300"},
	}

	t.Run("RawTextExtraction", func(t *testing.T) {
		for _, mp := range modelPages {
			t.Run(fmt.Sprintf("Page%d", mp.page), func(t *testing.T) {
				if mp.page > reader.NumPage() {
					t.Skipf("PDF has only %d pages", reader.NumPage())
				}

				page := reader.Page(mp.page)
				if page.V.IsNull() {
					t.Fatalf("page %d is null", mp.page)
				}

				text, err := page.GetPlainText(nil)
				if err != nil {
					t.Fatalf("GetPlainText failed for page %d: %v", mp.page, err)
				}

				t.Logf("=== Page %d raw text (%d chars) ===\n%s\n===",
					mp.page, len(text), text)

				if !strings.Contains(text, mp.weight) {
					t.Errorf("page %d: weight %q NOT found in raw text", mp.page, mp.weight)
				}

				if !strings.Contains(text, mp.heading) {
					t.Errorf("page %d: heading %q NOT found in raw text — THIS IS THE BUG",
						mp.page, mp.heading)
				}
			})
		}
	})

	// --- Step 2: Check section splitting on raw text (baseline — shows GetPlainText bug) ---

	t.Run("SectionSplitting_RawBaseline", func(t *testing.T) {
		for _, mp := range modelPages {
			t.Run(fmt.Sprintf("Page%d", mp.page), func(t *testing.T) {
				if mp.page > reader.NumPage() {
					t.Skipf("PDF has only %d pages", reader.NumPage())
				}

				page := reader.Page(mp.page)
				text, err := page.GetPlainText(nil)
				if err != nil {
					t.Fatalf("GetPlainText failed: %v", err)
				}

				sections := splitPageIntoSections(strings.TrimSpace(text), mp.page)

				t.Logf("Page %d produced %d sections:", mp.page, len(sections))
				for i, s := range sections {
					t.Logf("  [%d] heading=%q level=%d type=%s content=%.100s...",
						i, s.Heading, s.Level, s.Type, s.Content)
				}

				// Log (not assert) — raw GetPlainText has wrong ordering
				foundHeading := false
				for _, s := range sections {
					if strings.Contains(s.Heading, mp.heading) {
						foundHeading = true
						break
					}
				}
				if !foundHeading {
					t.Logf("page %d: (expected) heading %q missing in raw text sections — fixed by extractPageTextOrdered",
						mp.page, mp.heading)
				}
			})
		}
	})

	// --- Step 2b: Check extractPageTextOrdered output ---

	t.Run("OrderedTextExtraction", func(t *testing.T) {
		for _, mp := range modelPages {
			t.Run(fmt.Sprintf("Page%d", mp.page), func(t *testing.T) {
				if mp.page > reader.NumPage() {
					t.Skipf("PDF has only %d pages", reader.NumPage())
				}

				page := reader.Page(mp.page)
				text, err := extractPageTextOrdered(page)
				if err != nil {
					t.Fatalf("extractPageTextOrdered failed: %v", err)
				}

				t.Logf("=== Page %d ordered text (%d chars) ===\n%s\n===",
					mp.page, len(text), text)

				if !strings.Contains(text, mp.heading) {
					t.Errorf("page %d: heading %q NOT found in ordered text", mp.page, mp.heading)
				}
				if !strings.Contains(text, mp.weight) {
					t.Errorf("page %d: weight %q NOT found in ordered text", mp.page, mp.weight)
				}
			})
		}
	})

	// --- Step 3: Full parser pipeline ---

	t.Run("FullParserPipeline", func(t *testing.T) {
		p := &PDFParser{}
		result, err := p.Parse(context.Background(), pdfPath)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		// Search all sections for the model headings
		for _, mp := range modelPages {
			t.Run(fmt.Sprintf("Model_%s", mp.heading), func(t *testing.T) {
				foundHeading := false
				foundWeightWithModel := false

				for _, s := range result.Sections {
					if strings.Contains(s.Heading, mp.heading) {
						foundHeading = true
						t.Logf("Found heading %q in section: heading=%q page=%d",
							mp.heading, s.Heading, s.PageNumber)

						if strings.Contains(s.Content, mp.weight) {
							foundWeightWithModel = true
							t.Logf("Weight %q found in same section content", mp.weight)
						}
					}
				}

				if !foundHeading {
					t.Errorf("heading containing %q not found in any parsed section — model names are being lost by the parser",
						mp.heading)

					// Try to find the weight orphaned somewhere
					for _, s := range result.Sections {
						if strings.Contains(s.Content, mp.weight) && s.PageNumber == mp.page {
							t.Logf("  Weight %q found ORPHANED in section: heading=%q page=%d",
								mp.weight, s.Heading, s.PageNumber)
						}
					}
				}

				if !foundWeightWithModel {
					t.Errorf("weight %q not co-located with model heading %q — even if both are parsed, they're separated",
						mp.weight, mp.heading)
				}
			})
		}
	})
}
