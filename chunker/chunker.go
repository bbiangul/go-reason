package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"

	"github.com/bbiangul/go-reason/parser"
	"github.com/bbiangul/go-reason/store"
)

// Config controls the chunking behaviour.
type Config struct {
	MaxTokens int // Maximum estimated tokens per chunk.
	Overlap   int // Token overlap between consecutive child chunks.
}

// Chunker converts parsed document sections into store-ready chunks.
type Chunker struct {
	cfg Config
}

// New returns a Chunker with the given configuration.
// Zero-value fields are replaced with sensible defaults.
func New(cfg Config) *Chunker {
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 1024
	}
	if cfg.Overlap == 0 {
		cfg.Overlap = 128
	}
	return &Chunker{cfg: cfg}
}

// Chunk converts parsed sections into store chunks with hierarchical
// relationships.  It returns a flat slice where parent-child
// relationships are tracked via ParentChunkID.  The returned chunks use
// position indices as temporary IDs; real database IDs are assigned on
// insert.
func (c *Chunker) Chunk(sections []parser.Section) []store.Chunk {
	var chunks []store.Chunk
	pos := 0
	for _, sec := range sections {
		c.processSection(sec, nil, &chunks, &pos, -1, nil)
	}
	return chunks
}

// ChunkWithSectionMap converts parsed sections into store chunks and returns
// a parallel slice mapping each chunk index to its originating top-level
// section index. This enables callers to associate per-section data (e.g.
// images) with the correct chunk IDs after insertion.
func (c *Chunker) ChunkWithSectionMap(sections []parser.Section) ([]store.Chunk, []int) {
	var chunks []store.Chunk
	var sectionMap []int
	pos := 0
	for i, sec := range sections {
		c.processSection(sec, nil, &chunks, &pos, i, &sectionMap)
	}
	return chunks, sectionMap
}

// processSection recursively converts a parser.Section (and its children)
// into one parent chunk plus zero or more child chunks.
// When sectionIdx >= 0 and sectionMap is non-nil, each chunk's originating
// top-level section index is recorded.
func (c *Chunker) processSection(sec parser.Section, parentPos *int64, chunks *[]store.Chunk, pos *int, sectionIdx int, sectionMap *[]int) {
	// --- parent chunk ---
	parentContent := buildParentContent(sec)
	parentMeta := marshalMeta(sec.Metadata)
	parentHash := contentHash(parentContent)
	parentIndex := int64(*pos)

	parent := store.Chunk{
		ID:            parentIndex, // temporary, replaced on DB insert
		ParentChunkID: parentPos,
		Content:       parentContent,
		ChunkType:     chunkTypeFromSection(sec),
		Heading:       sec.Heading,
		PageNumber:    sec.PageNumber,
		PositionInDoc: *pos,
		TokenCount:    estimateTokens(parentContent),
		Metadata:      parentMeta,
		ContentHash:   parentHash,
	}
	*chunks = append(*chunks, parent)
	if sectionMap != nil {
		*sectionMap = append(*sectionMap, sectionIdx)
	}
	*pos++

	// --- child chunks from content ---
	if sec.Content != "" {
		fragments := c.splitContent(sec.Content)
		for _, frag := range fragments {
			childHash := contentHash(frag)
			child := store.Chunk{
				ID:            int64(*pos),
				ParentChunkID: &parentIndex,
				Content:       frag,
				ChunkType:     childChunkType(sec),
				Heading:       sec.Heading,
				PageNumber:    sec.PageNumber,
				PositionInDoc: *pos,
				TokenCount:    estimateTokens(frag),
				Metadata:      parentMeta,
				ContentHash:   childHash,
			}
			*chunks = append(*chunks, child)
			if sectionMap != nil {
				*sectionMap = append(*sectionMap, sectionIdx)
			}
			*pos++
		}
	}

	// --- recurse into child sections ---
	for _, child := range sec.Children {
		c.processSection(child, &parentIndex, chunks, pos, sectionIdx, sectionMap)
	}
}

// splitContent breaks a long text into fragments that each fit within
// MaxTokens, splitting at paragraph and then sentence boundaries.
// Consecutive fragments share an overlap of c.cfg.Overlap tokens worth
// of trailing text from the previous fragment.
func (c *Chunker) splitContent(text string) []string {
	if estimateTokens(text) <= c.cfg.MaxTokens {
		return []string{strings.TrimSpace(text)}
	}

	paragraphs := splitParagraphs(text)
	var fragments []string
	var current strings.Builder
	currentTokens := 0
	overlapText := ""

	for _, para := range paragraphs {
		paraTokens := estimateTokens(para)

		// If a single paragraph exceeds MaxTokens, split it by sentences.
		if paraTokens > c.cfg.MaxTokens {
			// Flush current buffer first.
			if current.Len() > 0 {
				fragments = append(fragments, strings.TrimSpace(current.String()))
				overlapText = extractOverlap(current.String(), c.cfg.Overlap)
				current.Reset()
				currentTokens = 0
			}
			sentenceFragments := c.splitBySentences(para, overlapText)
			fragments = append(fragments, sentenceFragments...)
			if len(sentenceFragments) > 0 {
				overlapText = extractOverlap(sentenceFragments[len(sentenceFragments)-1], c.cfg.Overlap)
			}
			continue
		}

		// Would adding this paragraph exceed the limit?
		if currentTokens+paraTokens > c.cfg.MaxTokens && current.Len() > 0 {
			fragments = append(fragments, strings.TrimSpace(current.String()))
			overlapText = extractOverlap(current.String(), c.cfg.Overlap)
			current.Reset()
			currentTokens = 0

			// Start the new fragment with overlap text.
			if overlapText != "" {
				current.WriteString(overlapText)
				current.WriteString("\n\n")
				currentTokens = estimateTokens(overlapText)
			}
		}

		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
		currentTokens += paraTokens
	}

	if current.Len() > 0 {
		fragments = append(fragments, strings.TrimSpace(current.String()))
	}

	return fragments
}

// splitBySentences breaks a paragraph into fragments at sentence
// boundaries, respecting MaxTokens and prepending overlap from the
// previous fragment.
func (c *Chunker) splitBySentences(text string, initialOverlap string) []string {
	sentences := splitSentences(text)
	var fragments []string
	var current strings.Builder
	currentTokens := 0

	if initialOverlap != "" {
		current.WriteString(initialOverlap)
		current.WriteString(" ")
		currentTokens = estimateTokens(initialOverlap)
	}

	for _, sent := range sentences {
		sentTokens := estimateTokens(sent)

		if currentTokens+sentTokens > c.cfg.MaxTokens && current.Len() > 0 {
			fragments = append(fragments, strings.TrimSpace(current.String()))
			overlap := extractOverlap(current.String(), c.cfg.Overlap)
			current.Reset()
			currentTokens = 0
			if overlap != "" {
				current.WriteString(overlap)
				current.WriteString(" ")
				currentTokens = estimateTokens(overlap)
			}
		}

		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(sent)
		currentTokens += sentTokens
	}

	if current.Len() > 0 {
		fragments = append(fragments, strings.TrimSpace(current.String()))
	}

	return fragments
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// estimateTokens approximates the token count of text using a simple
// word-based heuristic: tokens ~ words * 1.3.
func estimateTokens(text string) int {
	words := len(strings.Fields(text))
	return int(math.Ceil(float64(words) * 1.3))
}

// buildParentContent produces the parent chunk body: the heading
// followed by an abbreviated version of the section content (first
// 200 characters).
func buildParentContent(sec parser.Section) string {
	var b strings.Builder
	if sec.Heading != "" {
		b.WriteString(sec.Heading)
		b.WriteString("\n\n")
	}
	content := strings.TrimSpace(sec.Content)
	if len(content) > 200 {
		// Cut at the last space within the first 200 chars to avoid
		// splitting a word.
		idx := strings.LastIndex(content[:200], " ")
		if idx < 0 {
			idx = 200
		}
		content = content[:idx] + "..."
	}
	b.WriteString(content)
	return strings.TrimSpace(b.String())
}

// chunkTypeFromSection maps a section type to a chunk type string.
func chunkTypeFromSection(sec parser.Section) string {
	switch sec.Type {
	case "table":
		return "table"
	case "definition":
		return "definition"
	case "requirement":
		return "requirement"
	case "paragraph":
		return "paragraph"
	default:
		return "section"
	}
}

// childChunkType returns the chunk type to assign to child fragments
// of a section.
func childChunkType(sec parser.Section) string {
	switch sec.Type {
	case "table":
		return "table"
	case "definition":
		return "definition"
	case "requirement":
		return "requirement"
	default:
		return "paragraph"
	}
}

// splitParagraphs splits text on blank-line boundaries.
func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// splitSentences is a simple sentence tokeniser.  It splits on
// period/question-mark/exclamation followed by whitespace or end of
// string, while trying not to split on abbreviations.
func splitSentences(text string) []string {
	var sentences []string
	var cur strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		cur.WriteRune(runes[i])
		if runes[i] == '.' || runes[i] == '?' || runes[i] == '!' {
			// Look ahead: if next char is whitespace or end of string,
			// treat as sentence boundary (simple heuristic).
			if i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n' || runes[i+1] == '\t' {
				s := strings.TrimSpace(cur.String())
				if s != "" {
					sentences = append(sentences, s)
				}
				cur.Reset()
			}
		}
	}
	if cur.Len() > 0 {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			sentences = append(sentences, s)
		}
	}
	return sentences
}

// extractOverlap returns the trailing portion of text whose estimated
// token count is at most maxTokens.  It works at the word level.
func extractOverlap(text string, maxTokens int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	// tokens ~ words * 1.3, so max words ~ maxTokens / 1.3
	maxWords := int(float64(maxTokens) / 1.3)
	if maxWords > len(words) {
		maxWords = len(words)
	}
	if maxWords == 0 {
		return ""
	}
	return strings.Join(words[len(words)-maxWords:], " ")
}

// contentHash returns the SHA-256 hex digest of text.
func contentHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])
}

// marshalMeta serialises a metadata map to a JSON string.
// Returns "{}" for nil or empty maps.
func marshalMeta(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}
