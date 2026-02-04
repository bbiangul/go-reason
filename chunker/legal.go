package chunker

import (
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Clause boundary detection
// ---------------------------------------------------------------------------

// clausePattern matches hierarchical numbered clauses such as
// "1.1", "1.1.1", "12.3.4", etc. at the start of a line.
var clausePattern = regexp.MustCompile(`^(\d+(?:\.\d+)+)\s`)

// DetectClauseBoundaries scans text and returns the byte offsets where
// new numbered clauses begin.  Each entry in the returned slice is the
// index of the first byte of a clause number at the start of a line.
func DetectClauseBoundaries(text string) []int {
	lines := strings.Split(text, "\n")
	var boundaries []int
	offset := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if clausePattern.MatchString(trimmed) {
			boundaries = append(boundaries, offset)
		}
		offset += len(line) + 1 // +1 for the newline
	}
	return boundaries
}

// SplitByClauses splits text at clause boundaries so that each
// returned string starts with a clause number.  Text before the
// first clause (preamble) is returned as the first element if
// non-empty.
func SplitByClauses(text string) []string {
	boundaries := DetectClauseBoundaries(text)
	if len(boundaries) == 0 {
		return []string{text}
	}

	var parts []string
	for i, b := range boundaries {
		// Preamble before the first clause.
		if i == 0 && b > 0 {
			preamble := strings.TrimSpace(text[:b])
			if preamble != "" {
				parts = append(parts, preamble)
			}
		}

		var end int
		if i+1 < len(boundaries) {
			end = boundaries[i+1]
		} else {
			end = len(text)
		}
		part := strings.TrimSpace(text[b:end])
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

// ExtractClauseNumber extracts the leading clause number from text.
// For example, given "1.2.3 The contractor shall..." it returns
// "1.2.3" and true.
func ExtractClauseNumber(text string) (string, bool) {
	text = strings.TrimSpace(text)
	m := clausePattern.FindStringSubmatch(text)
	if len(m) < 2 {
		return "", false
	}
	return m[1], true
}

// ClauseDepth returns the nesting depth of a clause number.
// "1.1" returns 2, "1.1.1" returns 3, etc.
func ClauseDepth(clause string) int {
	if clause == "" {
		return 0
	}
	return strings.Count(clause, ".") + 1
}

// ---------------------------------------------------------------------------
// Definition extraction
// ---------------------------------------------------------------------------

// definitionMeansPattern matches lines where a quoted term is being
// defined using "means" or "shall mean".
var definitionMeansPattern = regexp.MustCompile(
	`(?i)^[""\x{201c}]([^"""\x{201d}]+)[""\x{201d}]\s+(?:means|shall\s+mean)\b`,
)

// definitionColonPattern matches "Term: definition" style entries.
var definitionColonPattern = regexp.MustCompile(
	`^([A-Z][A-Za-z\s]+):\s+(.+)`,
)

// Definition holds a single extracted defined term and its definition
// text.
type Definition struct {
	Term       string
	Definition string
	LineNumber int // zero-based line index within the input text
}

// ExtractDefinitions scans text for definition patterns and returns
// all found definitions.  It recognises two styles:
//   - Quoted term followed by "means" / "shall mean"
//   - Capitalised term followed by colon and explanation
func ExtractDefinitions(text string) []Definition {
	lines := strings.Split(text, "\n")
	var defs []Definition

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// "Term" means ...
		if m := definitionMeansPattern.FindStringSubmatch(trimmed); len(m) >= 2 {
			// The definition body is everything after the "means" / "shall mean".
			body := definitionMeansPattern.ReplaceAllString(trimmed, "")
			body = strings.TrimSpace(body)
			// Also collect continuation lines.
			body = collectContinuation(lines, i, body)
			defs = append(defs, Definition{
				Term:       m[1],
				Definition: trimmed, // keep full line as definition text
				LineNumber: i,
			})
			continue
		}

		// Term: definition
		if m := definitionColonPattern.FindStringSubmatch(trimmed); len(m) >= 3 {
			defs = append(defs, Definition{
				Term:       strings.TrimSpace(m[1]),
				Definition: trimmed,
				LineNumber: i,
			})
		}
	}
	return defs
}

// collectContinuation gathers non-empty continuation lines that follow
// the definition start line (lines that are indented or do not start a
// new definition/clause).
func collectContinuation(lines []string, startIdx int, initial string) string {
	var b strings.Builder
	b.WriteString(initial)
	for j := startIdx + 1; j < len(lines); j++ {
		line := lines[j]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}
		// Stop if this line starts a new clause or definition.
		if clausePattern.MatchString(trimmed) ||
			definitionMeansPattern.MatchString(trimmed) ||
			definitionColonPattern.MatchString(trimmed) {
			break
		}
		b.WriteString(" ")
		b.WriteString(trimmed)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Cross-reference detection
// ---------------------------------------------------------------------------

// crossRefPatterns match common cross-reference styles found in legal
// and contractual documents.
var crossRefPatterns = []*regexp.Regexp{
	// "clause 1.2", "Clause 1.2.3"
	regexp.MustCompile(`(?i)\bclause\s+(\d+(?:\.\d+)*)`),
	// "section 1.2", "Section 3"
	regexp.MustCompile(`(?i)\bsection\s+(\d+(?:\.\d+)*)`),
	// "article 5", "Article IV"
	regexp.MustCompile(`(?i)\barticle\s+(\d+|[IVXLCDM]+)`),
	// "schedule 1", "Schedule A"
	regexp.MustCompile(`(?i)\bschedule\s+([A-Z0-9]+)`),
	// "appendix A", "Appendix 3"
	regexp.MustCompile(`(?i)\bappendix\s+([A-Z0-9]+)`),
	// "annex 1", "Annex B"
	regexp.MustCompile(`(?i)\bannex\s+([A-Z0-9]+)`),
	// Parenthetical references: "(see 1.2.3)", "(ref. 4.5)"
	regexp.MustCompile(`\((?:see|ref\.?)\s+(\d+(?:\.\d+)*)\)`),
}

// CrossReference holds a detected cross-reference within text.
type CrossReference struct {
	FullMatch string // The entire matched substring (e.g. "clause 1.2.3")
	Target    string // The reference target (e.g. "1.2.3")
	Type      string // "clause", "section", "article", "schedule", "appendix", "annex", "ref"
	Offset    int    // Byte offset of the match within the input text
}

// DetectCrossReferences scans text and returns all cross-references
// found.
func DetectCrossReferences(text string) []CrossReference {
	typeLabels := []string{
		"clause", "section", "article", "schedule", "appendix", "annex", "ref",
	}

	var refs []CrossReference
	for i, re := range crossRefPatterns {
		matches := re.FindAllStringSubmatchIndex(text, -1)
		for _, loc := range matches {
			if len(loc) < 4 {
				continue
			}
			refs = append(refs, CrossReference{
				FullMatch: text[loc[0]:loc[1]],
				Target:    text[loc[2]:loc[3]],
				Type:      typeLabels[i],
				Offset:    loc[0],
			})
		}
	}
	return refs
}

// HasCrossReferences is a convenience function that reports whether
// text contains any cross-references.
func HasCrossReferences(text string) bool {
	for _, re := range crossRefPatterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}
