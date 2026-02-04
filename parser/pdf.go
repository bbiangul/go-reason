package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

type PDFParser struct{}

func (p *PDFParser) SupportedFormats() []string { return []string{"pdf"} }

func (p *PDFParser) Parse(ctx context.Context, path string) (*ParseResult, error) {
	f, reader, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening PDF: %w", err)
	}
	defer f.Close()

	totalPages := reader.NumPage()
	sections := make([]Section, 0)

	for i := 1; i <= totalPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			// Skip pages that fail to extract
			continue
		}

		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		// Split page text into sections by detecting heading patterns
		pageSections := splitPageIntoSections(text, i)
		sections = append(sections, pageSections...)
	}

	if len(sections) == 0 {
		return &ParseResult{
			Method: "native",
			Sections: []Section{{
				Content:    "Unable to extract text from PDF",
				Type:       "paragraph",
				PageNumber: 1,
			}},
		}, nil
	}

	return &ParseResult{
		Sections: sections,
		Method:   "native",
	}, nil
}

// splitPageIntoSections breaks page text into logical sections.
func splitPageIntoSections(text string, pageNum int) []Section {
	lines := strings.Split(text, "\n")
	var sections []Section
	var currentContent strings.Builder
	var currentHeading string
	currentLevel := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			continue
		}

		// Detect headings: all-caps lines, numbered sections, short bold-like lines
		if isLikelyHeading(trimmed) {
			// Save previous section
			if currentContent.Len() > 0 {
				sections = append(sections, Section{
					Heading:    currentHeading,
					Content:    strings.TrimSpace(currentContent.String()),
					Level:      currentLevel,
					PageNumber: pageNum,
					Type:       classifySectionType(currentHeading, currentContent.String()),
				})
				currentContent.Reset()
			}
			currentHeading = trimmed
			currentLevel = detectHeadingLevel(trimmed)
		} else {
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			currentContent.WriteString(trimmed)
		}
	}

	// Final section
	if currentContent.Len() > 0 {
		sections = append(sections, Section{
			Heading:    currentHeading,
			Content:    strings.TrimSpace(currentContent.String()),
			Level:      currentLevel,
			PageNumber: pageNum,
			Type:       classifySectionType(currentHeading, currentContent.String()),
		})
	}

	// If no sections were created, return the whole page as one section
	if len(sections) == 0 && strings.TrimSpace(text) != "" {
		sections = append(sections, Section{
			Content:    text,
			PageNumber: pageNum,
			Type:       "paragraph",
		})
	}

	return sections
}

func isLikelyHeading(line string) bool {
	// All caps and short
	if len(line) < 100 && line == strings.ToUpper(line) && len(line) > 2 {
		return true
	}
	// Numbered section like "1.", "1.1", "1.1.1", "3.9.1", "7.3.1.2"
	if len(line) < 120 {
		if len(line) > 0 && line[0] >= '0' && line[0] <= '9' && strings.Contains(line[:min(10, len(line))], ".") {
			return true
		}
		lower := strings.ToLower(line)
		// English heading prefixes
		if strings.HasPrefix(lower, "section ") || strings.HasPrefix(lower, "article ") ||
			strings.HasPrefix(lower, "chapter ") || strings.HasPrefix(lower, "part ") {
			return true
		}
		// Spanish heading prefixes
		if strings.HasPrefix(lower, "sección ") || strings.HasPrefix(lower, "seccion ") ||
			strings.HasPrefix(lower, "capítulo ") || strings.HasPrefix(lower, "capitulo ") ||
			strings.HasPrefix(lower, "anexo ") {
			return true
		}
		// "Tabla N..." / "Figura N..." — only when followed by a digit to avoid
		// matching mid-paragraph text like "tabla siguiente muestra..."
		if strings.HasPrefix(lower, "tabla ") && len(lower) > 6 && lower[6] >= '0' && lower[6] <= '9' {
			return true
		}
		if strings.HasPrefix(lower, "figura ") && len(lower) > 7 && lower[7] >= '0' && lower[7] <= '9' {
			return true
		}
	}
	return false
}

func detectHeadingLevel(heading string) int {
	// Count dots in numbering to determine depth
	parts := strings.SplitN(heading, " ", 2)
	if len(parts) > 0 {
		dots := strings.Count(parts[0], ".")
		if dots > 0 {
			return dots
		}
	}
	// All-caps = top level
	if heading == strings.ToUpper(heading) {
		return 1
	}
	return 2
}

func classifySectionType(heading, content string) string {
	headingLower := strings.ToLower(heading)
	contentLower := strings.ToLower(content)

	// Definition: check heading and content for definition-related keywords
	if strings.Contains(headingLower, "definition") || strings.Contains(headingLower, "definición") ||
		strings.Contains(headingLower, "glosario") || strings.Contains(headingLower, "glossary") ||
		strings.Contains(contentLower, "definition") || strings.Contains(contentLower, "definición") {
		return "definition"
	}
	// Requirement: check heading and content for requirement-related keywords
	if strings.Contains(headingLower, "shall") || strings.Contains(headingLower, "must") || strings.Contains(headingLower, "requirement") ||
		strings.Contains(headingLower, "requisito") || strings.Contains(headingLower, "especificación") ||
		strings.Contains(contentLower, "shall") || strings.Contains(contentLower, "must") || strings.Contains(contentLower, "requirement") ||
		strings.Contains(contentLower, "requisito") || strings.Contains(contentLower, "especificación") {
		return "requirement"
	}
	// Table: check heading for table keywords
	if strings.Contains(headingLower, "table") || strings.Contains(headingLower, "tabla") {
		return "table"
	}
	// Structural table detection via content: tabs/pipes indicate actual table formatting
	if strings.Count(content, "\t") > 3 || strings.Count(content, "|") > 3 {
		return "table"
	}
	if strings.Contains(headingLower, "anexo") || strings.Contains(headingLower, "annex") {
		return "annex"
	}
	return "section"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
