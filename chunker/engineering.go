package chunker

import (
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Requirement detection
// ---------------------------------------------------------------------------

// requirementPattern matches normative requirement keywords as defined
// by RFC 2119 and ISO directive language.  The keywords must appear as
// whole words (typically uppercase in standards documents, but this
// pattern is case-insensitive for robustness).
var requirementPattern = regexp.MustCompile(
	`(?i)\b(SHALL\s+NOT|MUST\s+NOT|SHALL|MUST|SHOULD\s+NOT|SHOULD|REQUIRED|RECOMMENDED|MAY|OPTIONAL)\b`,
)

// Requirement holds a detected normative statement.
type Requirement struct {
	Text       string // The full sentence or clause containing the keyword.
	Keyword    string // The matched keyword (e.g. "SHALL", "MUST NOT").
	Level      string // "mandatory", "recommended", or "optional".
	LineNumber int    // Zero-based line index within the input text.
}

// DetectRequirements scans text line by line and returns every line
// that contains a normative requirement keyword.
func DetectRequirements(text string) []Requirement {
	lines := strings.Split(text, "\n")
	var reqs []Requirement

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		matches := requirementPattern.FindAllString(trimmed, -1)
		if len(matches) == 0 {
			continue
		}
		// Use the first (strongest) keyword found on the line.
		kw := strings.ToUpper(matches[0])
		reqs = append(reqs, Requirement{
			Text:       trimmed,
			Keyword:    kw,
			Level:      requirementLevel(kw),
			LineNumber: i,
		})
	}
	return reqs
}

// IsRequirement reports whether text contains at least one normative
// requirement keyword.
func IsRequirement(text string) bool {
	return requirementPattern.MatchString(text)
}

// requirementLevel maps a keyword to its normative level.
func requirementLevel(keyword string) string {
	switch strings.ToUpper(strings.TrimSpace(keyword)) {
	case "SHALL", "SHALL NOT", "MUST", "MUST NOT", "REQUIRED":
		return "mandatory"
	case "SHOULD", "SHOULD NOT", "RECOMMENDED":
		return "recommended"
	case "MAY", "OPTIONAL":
		return "optional"
	default:
		return "mandatory"
	}
}

// ---------------------------------------------------------------------------
// Standards reference detection
// ---------------------------------------------------------------------------

// standardsPatterns match references to well-known standards bodies
// and their document numbering schemes.
var standardsPatterns = []*regexp.Regexp{
	// ISO standards: "ISO 9001", "ISO/IEC 27001:2022", "ISO 9001-1"
	regexp.MustCompile(`\bISO(?:/IEC)?\s+\d[\d\-]+(?::\d{4})?`),
	// IEC standards: "IEC 61508", "IEC 62443-3-3"
	regexp.MustCompile(`\bIEC\s+\d[\d\-]+(?::\d{4})?`),
	// ASTM standards: "ASTM D1234", "ASTM E1234-56"
	regexp.MustCompile(`\bASTM\s+[A-Z]\d+(?:-\d+)?(?::\d{4})?`),
	// IEEE standards: "IEEE 802.11", "IEEE Std 1547"
	regexp.MustCompile(`\bIEEE\s+(?:Std\s+)?\d[\d\.]+`),
	// ANSI standards: "ANSI Z359.1", "ANSI/NFPA 70"
	regexp.MustCompile(`\bANSI(?:/\w+)?\s+[A-Z]?[\d\.]+`),
	// BS (British Standards): "BS EN 1090", "BS 7671"
	regexp.MustCompile(`\bBS\s+(?:EN\s+)?\d[\d\-]+`),
	// EN (European Norm): "EN 1090-2"
	regexp.MustCompile(`\bEN\s+\d[\d\-]+`),
	// DIN (German standards): "DIN EN 1090"
	regexp.MustCompile(`\bDIN\s+(?:EN\s+)?\d[\d\-]+`),
	// NFPA: "NFPA 70", "NFPA 101"
	regexp.MustCompile(`\bNFPA\s+\d+`),
	// ASME: "ASME B31.3", "ASME BPVC"
	regexp.MustCompile(`\bASME\s+[A-Z][\d\.]+`),
	// AWS: "AWS D1.1"
	regexp.MustCompile(`\bAWS\s+[A-Z][\d\.]+`),
	// MIL-STD: "MIL-STD-810G"
	regexp.MustCompile(`\bMIL-STD-\d+[A-Z]?`),
	// SAE: "SAE J1939", "SAE AMS 2759"
	regexp.MustCompile(`\bSAE\s+[A-Z]+\s*\d+`),
	// API: "API 650", "API Std 520"
	regexp.MustCompile(`\bAPI\s+(?:Std\s+)?\d+`),
}

// StandardsReference holds a detected standards reference.
type StandardsReference struct {
	Standard string // The matched standard identifier (e.g. "ISO 9001:2015").
	Body     string // The standards body (e.g. "ISO", "ASTM").
	Offset   int    // Byte offset of the match within the input text.
}

// DetectStandardsReferences scans text and returns all standards
// references found.
func DetectStandardsReferences(text string) []StandardsReference {
	bodyNames := []string{
		"ISO", "IEC", "ASTM", "IEEE", "ANSI", "BS", "EN", "DIN",
		"NFPA", "ASME", "AWS", "MIL", "SAE", "API",
	}

	var refs []StandardsReference
	for i, re := range standardsPatterns {
		matches := re.FindAllStringIndex(text, -1)
		for _, loc := range matches {
			refs = append(refs, StandardsReference{
				Standard: text[loc[0]:loc[1]],
				Body:     bodyNames[i],
				Offset:   loc[0],
			})
		}
	}
	return refs
}

// HasStandardsReference reports whether text contains any standards
// reference.
func HasStandardsReference(text string) bool {
	for _, re := range standardsPatterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Table preservation
// ---------------------------------------------------------------------------

// TableChunk holds a detected table block and its surrounding context.
type TableChunk struct {
	Content    string // The full table text, preserved as-is.
	StartLine  int    // Zero-based line index where the table begins.
	EndLine    int    // Zero-based line index where the table ends (exclusive).
	HasHeaders bool   // Whether a header separator row was detected.
}

// DetectTables scans text and identifies contiguous blocks that appear
// to be tabular data.  Tables are preserved as atomic units so that
// the chunker does not split them across chunk boundaries.
func DetectTables(text string) []TableChunk {
	lines := strings.Split(text, "\n")
	var tables []TableChunk

	i := 0
	for i < len(lines) {
		// Look for the start of a table.
		if isTableLine(lines[i]) {
			start := i
			hasHeaders := false
			for i < len(lines) && isTableLine(lines[i]) {
				if isHeaderSeparator(lines[i]) {
					hasHeaders = true
				}
				i++
			}
			// Require at least 2 table-like lines.
			if i-start >= 2 {
				content := strings.Join(lines[start:i], "\n")
				tables = append(tables, TableChunk{
					Content:    content,
					StartLine:  start,
					EndLine:    i,
					HasHeaders: hasHeaders,
				})
			}
			continue
		}
		i++
	}
	return tables
}

// PreserveTableChunks examines text and returns a list of text
// fragments where tables are kept as single atomic pieces and the
// remaining prose is split normally.  The returned fragments are in
// document order.
func PreserveTableChunks(text string) []string {
	tables := DetectTables(text)
	if len(tables) == 0 {
		return []string{text}
	}

	lines := strings.Split(text, "\n")
	var fragments []string
	cursor := 0

	for _, tbl := range tables {
		// Prose before this table.
		if cursor < tbl.StartLine {
			prose := strings.TrimSpace(strings.Join(lines[cursor:tbl.StartLine], "\n"))
			if prose != "" {
				fragments = append(fragments, prose)
			}
		}
		// The table itself (atomic).
		fragments = append(fragments, tbl.Content)
		cursor = tbl.EndLine
	}

	// Remaining prose after the last table.
	if cursor < len(lines) {
		prose := strings.TrimSpace(strings.Join(lines[cursor:], "\n"))
		if prose != "" {
			fragments = append(fragments, prose)
		}
	}

	return fragments
}

// ---------------------------------------------------------------------------
// Table detection helpers
// ---------------------------------------------------------------------------

// isTableLine reports whether a line looks like part of a table.
func isTableLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	// Markdown-style pipe tables.
	if strings.Contains(trimmed, "|") {
		return true
	}
	// Tab-delimited columns (at least two tabs).
	if strings.Count(trimmed, "\t") >= 2 {
		return true
	}
	// Separator rows.
	if isHeaderSeparator(trimmed) {
		return true
	}
	return false
}

// isHeaderSeparator detects markdown-style header separators like
// "|---|---|" or "------".
func isHeaderSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	// Remove pipe characters and spaces, see if the rest is all dashes.
	cleaned := strings.ReplaceAll(trimmed, "|", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "") // alignment markers
	if len(cleaned) < 3 {
		return false
	}
	for _, r := range cleaned {
		if r != '-' {
			return false
		}
	}
	return true
}
