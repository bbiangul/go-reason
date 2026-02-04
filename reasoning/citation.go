package reasoning

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/brunobiangulo/goreason/store"
)

// Citation represents an extracted citation from an answer.
type Citation struct {
	Text       string `json:"text"`       // The cited text
	SourceRef  string `json:"source_ref"` // Reference string (e.g., "doc.pdf, Section 3.2")
	ChunkID    int64  `json:"chunk_id"`   // Matched chunk ID, 0 if unmatched
	Verified   bool   `json:"verified"`   // Whether the citation was verified against sources
}

var (
	// Patterns for common citation styles
	citationPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\(([^)]+\.(?:pdf|docx|xlsx|pptx))[^)]*\)`),          // (document.pdf, ...)
		regexp.MustCompile(`(?:Section|Sec\.|§)\s*(\d+(?:\.\d+)*)`),              // Section 3.2
		regexp.MustCompile(`(?:Article|Art\.)\s*(\d+(?:\.\d+)*)`),                // Article 5
		regexp.MustCompile(`(?:Clause|Cl\.)\s*(\d+(?:\.\d+)*)`),                  // Clause 7.1
		regexp.MustCompile(`(?:Page|p\.)\s*(\d+)`),                                // Page 12
		regexp.MustCompile(`\[Source\s*(\d+)\]`),                                  // [Source 1]
	}
)

// ExtractCitations finds citation references in an answer text.
func ExtractCitations(answer string, chunks []store.RetrievalResult) []Citation {
	var citations []Citation
	seen := make(map[string]bool)

	for _, pattern := range citationPatterns {
		matches := pattern.FindAllStringSubmatch(answer, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			ref := strings.TrimSpace(match[0])
			if seen[ref] {
				continue
			}
			seen[ref] = true

			citation := Citation{
				Text:      ref,
				SourceRef: match[1],
			}

			// Try to match to a chunk
			citation.ChunkID, citation.Verified = matchCitationToChunk(match[1], chunks)
			citations = append(citations, citation)
		}
	}

	return citations
}

// matchCitationToChunk tries to find the chunk that a citation refers to.
func matchCitationToChunk(ref string, chunks []store.RetrievalResult) (int64, bool) {
	lowerRef := strings.ToLower(ref)

	// Try filename match
	for _, c := range chunks {
		if strings.Contains(strings.ToLower(c.Filename), lowerRef) {
			return c.ChunkID, true
		}
	}

	// Try heading match
	for _, c := range chunks {
		if c.Heading != "" && strings.Contains(strings.ToLower(c.Heading), lowerRef) {
			return c.ChunkID, true
		}
	}

	// Try page number match
	var pageNum int
	if _, err := fmt.Sscanf(ref, "%d", &pageNum); err == nil && pageNum > 0 {
		for _, c := range chunks {
			if c.PageNumber == pageNum {
				return c.ChunkID, true
			}
		}
	}

	// Try source number match (e.g., "1" from "[Source 1]")
	var srcNum int
	if _, err := fmt.Sscanf(ref, "%d", &srcNum); err == nil && srcNum > 0 && srcNum <= len(chunks) {
		return chunks[srcNum-1].ChunkID, true
	}

	return 0, false
}
