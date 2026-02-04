package chunker

import (
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Heading pattern detection
// ---------------------------------------------------------------------------

// headingPatterns are compiled regular expressions for common heading
// styles found in structured documents.
var headingPatterns = []*regexp.Regexp{
	// Numbered: "1.", "1.2", "1.2.3", optionally followed by a title
	regexp.MustCompile(`^\s*(\d+\.)+(\d+)?\s+\S`),
	// Uppercase line (e.g. "INTRODUCTION")
	regexp.MustCompile(`^[A-Z][A-Z\s]{4,}$`),
	// Markdown-style: "# Heading", "## Sub-heading"
	regexp.MustCompile(`^#{1,6}\s+\S`),
	// Appendix / Annex: "Appendix A", "Annex 1"
	regexp.MustCompile(`(?i)^(appendix|annex|schedule|exhibit)\s+[A-Z0-9]`),
	// Article: "Article 1", "Article II"
	regexp.MustCompile(`(?i)^article\s+[IVXLCDM\d]+`),
}

// IsHeading reports whether a line of text looks like a heading.
func IsHeading(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	for _, re := range headingPatterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Section numbering
// ---------------------------------------------------------------------------

// numberingPattern matches hierarchical numbering such as "1.", "1.2",
// "1.2.3", etc.
var numberingPattern = regexp.MustCompile(`^(\d+(?:\.\d+)*)\.\s`)

// DetectNumbering extracts the hierarchical number prefix from a line.
// It returns the matched number string (e.g. "1.2.3") and true, or
// an empty string and false if none was found.
func DetectNumbering(line string) (string, bool) {
	line = strings.TrimSpace(line)
	m := numberingPattern.FindStringSubmatch(line)
	if len(m) < 2 {
		return "", false
	}
	return m[1], true
}

// NumberingLevel returns the depth implied by a hierarchical number
// string.  "1" is level 1, "1.2" is level 2, "1.2.3" is level 3, etc.
func NumberingLevel(numbering string) int {
	if numbering == "" {
		return 0
	}
	return strings.Count(numbering, ".") + 1
}

// ---------------------------------------------------------------------------
// Content type classification
// ---------------------------------------------------------------------------

// ContentType classifies a block of text into one of the canonical
// section types: "table", "definition", "requirement", "paragraph",
// or "section".  The heuristics look at structural cues rather than
// semantic meaning.
func ContentType(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "paragraph"
	}

	if looksLikeTable(trimmed) {
		return "table"
	}
	if looksLikeDefinition(trimmed) {
		return "definition"
	}
	if looksLikeRequirement(trimmed) {
		return "requirement"
	}
	if IsHeading(firstLine(trimmed)) {
		return "section"
	}
	return "paragraph"
}

// ---------------------------------------------------------------------------
// Detection helpers
// ---------------------------------------------------------------------------

// looksLikeTable returns true when text appears to contain a table.
func looksLikeTable(text string) bool {
	lines := strings.Split(text, "\n")

	// Markdown-style tables: at least 3 lines, pipe characters in most.
	if len(lines) >= 3 {
		pipeCount := 0
		for _, l := range lines {
			if strings.Contains(l, "|") {
				pipeCount++
			}
		}
		if pipeCount >= len(lines)/2 {
			return true
		}
	}

	// Tab-delimited columns: at least 2 lines with multiple tabs.
	tabLines := 0
	for _, l := range lines {
		if strings.Count(l, "\t") >= 2 {
			tabLines++
		}
	}
	if tabLines >= 2 {
		return true
	}

	// Separator rows.
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if len(trimmed) > 3 && (allChar(trimmed, '-') || allChar(trimmed, '=')) {
			return true
		}
	}

	return false
}

// definitionPattern matches lines like:
//
//	"Term" means ...
//	"Term" shall mean ...
//	Term: definition text
var definitionPattern = regexp.MustCompile(
	`(?i)(?:^"[^"]+"\s+(?:means|shall\s+mean))|(?:^\S+.*?:\s+\S)`,
)

// looksLikeDefinition reports whether text looks like a definition
// block (glossary entries, defined terms, etc.).
func looksLikeDefinition(text string) bool {
	lines := strings.Split(text, "\n")
	defCount := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if definitionPattern.MatchString(l) {
			defCount++
		}
	}
	// At least one definition-style line in a short block, or multiple
	// in a longer one.
	if len(lines) <= 3 {
		return defCount >= 1
	}
	return defCount >= 2
}

// requirementKeywords are words that typically mark normative
// requirements in standards and contracts.
var requirementKeywords = []string{
	"SHALL", "MUST", "REQUIRED", "SHALL NOT", "MUST NOT",
}

// looksLikeRequirement reports whether text contains normative
// requirement language.
func looksLikeRequirement(text string) bool {
	upper := strings.ToUpper(text)
	for _, kw := range requirementKeywords {
		if strings.Contains(upper, kw) {
			return true
		}
	}
	return false
}

// firstLine returns the first non-empty line of text.
func firstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// allChar reports whether every character in s is c.
func allChar(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != c {
			return false
		}
	}
	return len(s) > 0
}
