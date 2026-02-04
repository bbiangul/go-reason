package parser

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Registry tests
// ---------------------------------------------------------------------------

func TestRegistryBuiltInParsers(t *testing.T) {
	reg := NewRegistry()

	formats := []struct {
		format     string
		wantParser string
	}{
		{"pdf", "*parser.PDFParser"},
		{"docx", "*parser.DOCXParser"},
		{"xlsx", "*parser.XLSXParser"},
		{"xls", "*parser.XLSXParser"},
		{"pptx", "*parser.PPTXParser"},
	}

	for _, tt := range formats {
		t.Run(tt.format, func(t *testing.T) {
			p, err := reg.Get(tt.format)
			if err != nil {
				t.Fatalf("Get(%q) returned error: %v", tt.format, err)
			}
			if p == nil {
				t.Fatalf("Get(%q) returned nil parser", tt.format)
			}
			// Verify the parser supports the expected format.
			supported := p.SupportedFormats()
			found := false
			for _, f := range supported {
				if f == tt.format {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("parser for %q does not list %q in SupportedFormats(): %v",
					tt.format, tt.format, supported)
			}
		})
	}
}

func TestRegistryUnknown(t *testing.T) {
	reg := NewRegistry()

	unknownFormats := []string{"txt", "csv", "json", "html", "rtf", "odt", ""}
	for _, fmt := range unknownFormats {
		t.Run("format_"+fmt, func(t *testing.T) {
			p, err := reg.Get(fmt)
			if err == nil {
				t.Errorf("Get(%q) expected error for unknown format, got parser: %v", fmt, p)
			}
			if p != nil {
				t.Errorf("Get(%q) expected nil parser for unknown format", fmt)
			}
		})
	}
}

func TestRegistryCustomParser(t *testing.T) {
	reg := NewRegistry()

	// Before registration, "custom" should fail.
	_, err := reg.Get("custom")
	if err == nil {
		t.Fatal("expected error for unregistered format")
	}

	// Register a custom parser and verify retrieval.
	reg.Register("custom", &PDFParser{}) // reuse PDFParser as a stand-in
	p, err := reg.Get("custom")
	if err != nil {
		t.Fatalf("Get(\"custom\") after Register returned error: %v", err)
	}
	if p == nil {
		t.Fatal("Get(\"custom\") returned nil after Register")
	}
}

// ---------------------------------------------------------------------------
// splitPageIntoSections tests
// ---------------------------------------------------------------------------

func TestSplitPageIntoSections(t *testing.T) {
	text := `INTRODUCTION
This is the introduction section with some text.

1.1 Scope
The scope of this document covers requirements.

1.2 Definitions
"Force Majeure" means any event beyond control.`

	sections := splitPageIntoSections(text, 1)

	if len(sections) < 3 {
		t.Fatalf("expected at least 3 sections, got %d", len(sections))
	}

	// First section: "INTRODUCTION" heading
	if sections[0].Heading != "INTRODUCTION" {
		t.Errorf("section[0].Heading = %q, want %q", sections[0].Heading, "INTRODUCTION")
	}
	if sections[0].PageNumber != 1 {
		t.Errorf("section[0].PageNumber = %d, want 1", sections[0].PageNumber)
	}
	if sections[0].Content == "" {
		t.Error("section[0].Content should not be empty")
	}

	// Second section: "1.1 Scope"
	if sections[1].Heading != "1.1 Scope" {
		t.Errorf("section[1].Heading = %q, want %q", sections[1].Heading, "1.1 Scope")
	}
	if sections[1].Content == "" {
		t.Error("section[1].Content should contain scope text")
	}

	// Third section: "1.2 Definitions"
	if sections[2].Heading != "1.2 Definitions" {
		t.Errorf("section[2].Heading = %q, want %q", sections[2].Heading, "1.2 Definitions")
	}
	if sections[2].Type != "definition" {
		t.Errorf("section[2].Type = %q, want %q", sections[2].Type, "definition")
	}
}

func TestSplitPageIntoSectionsEmptyText(t *testing.T) {
	sections := splitPageIntoSections("", 1)
	if len(sections) != 0 {
		t.Errorf("expected 0 sections for empty text, got %d", len(sections))
	}
}

func TestSplitPageIntoSectionsNoHeadings(t *testing.T) {
	text := "This is just a regular paragraph with no headings at all."
	sections := splitPageIntoSections(text, 5)

	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0].PageNumber != 5 {
		t.Errorf("section[0].PageNumber = %d, want 5", sections[0].PageNumber)
	}
	// When no headings are detected, the whole page is returned as a single
	// section.  The fallback path at the end of splitPageIntoSections sets
	// Type = "paragraph".  However, if text went through the main loop
	// without a heading, classifySectionType determines the type.
	// For generic text without keywords, it returns "section".
	if sections[0].Type != "section" {
		t.Errorf("section[0].Type = %q, want %q", sections[0].Type, "section")
	}
}

func TestSplitPageIntoSectionsWhitespaceOnly(t *testing.T) {
	sections := splitPageIntoSections("   \n\n   \n  ", 1)
	if len(sections) != 0 {
		t.Errorf("expected 0 sections for whitespace-only text, got %d", len(sections))
	}
}

// ---------------------------------------------------------------------------
// isLikelyHeading tests
// ---------------------------------------------------------------------------

func TestIsLikelyHeading(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		// All-caps headings
		{"all_caps_short", "INTRODUCTION", true},
		{"all_caps_multi_word", "TERMS AND CONDITIONS", true},
		{"all_caps_too_short", "AB", false},

		// Numbered sections
		{"numbered_1.1", "1.1 Scope", true},
		{"numbered_1.2.3", "1.2.3 Detailed Requirements", true},
		{"numbered_single_dot", "3. Overview", true},

		// Keyword prefixes
		{"section_prefix", "Section 5 General", true},
		{"article_prefix", "Article III Obligations", true},
		{"chapter_prefix", "Chapter 2 Architecture", true},
		{"part_prefix", "Part A Summary", true},

		// Not headings
		{"regular_sentence", "This is a regular sentence.", false},
		{"lowercase_text", "some regular content here", false},
		{"mixed_case", "The Contractor shall provide...", false},
		{"long_all_caps", string(make([]byte, 101)), false}, // >100 chars
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the "long_all_caps" test, build a proper all-caps string >100 chars.
			line := tt.line
			if tt.name == "long_all_caps" {
				buf := make([]byte, 101)
				for i := range buf {
					buf[i] = 'A'
				}
				line = string(buf)
			}
			got := isLikelyHeading(line)
			if got != tt.want {
				t.Errorf("isLikelyHeading(%q) = %v, want %v", line, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// classifySectionType tests
// ---------------------------------------------------------------------------

func TestClassifySectionType(t *testing.T) {
	tests := []struct {
		name    string
		heading string
		content string
		want    string
	}{
		{"definition_heading", "Definitions", "These terms are defined below.", "definition"},
		{"definition_content", "Glossary", "The definition of X is...", "definition"},
		{"requirement_shall", "Requirements", "The system shall perform...", "requirement"},
		{"requirement_must", "Obligations", "The contractor must deliver...", "requirement"},
		{"requirement_keyword", "Scope", "Each requirement listed here.", "requirement"},
		{"table_pipes", "Data", "Col1 | Col2 | Col3 | Col4 | Col5", "table"},
		{"table_tabs", "Data", "A\tB\tC\tD\tE", "table"},
		{"table_heading", "Table 1", "Some content", "table"},
		{"regular_section", "Introduction", "This is an overview of the project.", "section"},
		{"empty_heading", "", "Just some text without keywords.", "section"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySectionType(tt.heading, tt.content)
			if got != tt.want {
				t.Errorf("classifySectionType(%q, %q) = %q, want %q",
					tt.heading, tt.content, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// detectHeadingLevel tests
// ---------------------------------------------------------------------------

func TestDetectHeadingLevel(t *testing.T) {
	tests := []struct {
		name    string
		heading string
		want    int
	}{
		{"single_number_dot", "1. Introduction", 1},
		{"two_levels", "1.2 Scope", 1},
		{"three_levels", "1.2.3 Detailed", 2},
		{"four_levels", "1.2.3.4 Deep", 3},
		{"all_caps", "INTRODUCTION", 1},
		{"mixed_case_no_number", "Summary", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectHeadingLevel(tt.heading)
			if got != tt.want {
				t.Errorf("detectHeadingLevel(%q) = %d, want %d", tt.heading, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ParseResult / Section structure tests
// ---------------------------------------------------------------------------

func TestSectionFieldsPopulated(t *testing.T) {
	text := `SCOPE
The scope of this document shall cover all requirements.

1.1 System Requirements
The system must operate under the following conditions.`

	sections := splitPageIntoSections(text, 3)

	for i, sec := range sections {
		if sec.PageNumber != 3 {
			t.Errorf("section[%d].PageNumber = %d, want 3", i, sec.PageNumber)
		}
		if sec.Content == "" {
			t.Errorf("section[%d].Content is empty", i)
		}
		if sec.Type == "" {
			t.Errorf("section[%d].Type is empty", i)
		}
	}

	// First section should be a requirement type (contains "shall")
	if sections[0].Type != "requirement" {
		t.Errorf("section[0].Type = %q, want %q (content has 'shall')",
			sections[0].Type, "requirement")
	}

	// Check heading levels
	if sections[0].Level != 1 {
		t.Errorf("section[0].Level = %d, want 1 (all-caps heading)", sections[0].Level)
	}
	if sections[1].Level < 1 {
		t.Errorf("section[1].Level = %d, want >= 1", sections[1].Level)
	}
}
