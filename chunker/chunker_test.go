package chunker

import (
	"strings"
	"testing"

	"github.com/bbiangul/go-reason/parser"
)

// ---------------------------------------------------------------------------
// Core chunker tests
// ---------------------------------------------------------------------------

func TestChunkSimple(t *testing.T) {
	c := New(Config{MaxTokens: 512, Overlap: 64})
	sections := []parser.Section{
		{
			Heading:    "Introduction",
			Content:    "This is the introduction to the document.",
			Level:      1,
			PageNumber: 1,
			Type:       "section",
		},
	}

	chunks := c.Chunk(sections)

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	// First chunk is the parent.
	parent := chunks[0]
	if parent.Heading != "Introduction" {
		t.Errorf("parent.Heading = %q, want %q", parent.Heading, "Introduction")
	}
	if parent.PageNumber != 1 {
		t.Errorf("parent.PageNumber = %d, want 1", parent.PageNumber)
	}
	if parent.ChunkType != "section" {
		t.Errorf("parent.ChunkType = %q, want %q", parent.ChunkType, "section")
	}
	if parent.ParentChunkID != nil {
		t.Errorf("parent.ParentChunkID should be nil for top-level, got %v", parent.ParentChunkID)
	}
	if parent.ContentHash == "" {
		t.Error("parent.ContentHash should not be empty")
	}
	if parent.TokenCount <= 0 {
		t.Error("parent.TokenCount should be > 0")
	}

	// Second chunk is the child content chunk.
	if len(chunks) < 2 {
		t.Fatal("expected a child chunk for the section content")
	}
	child := chunks[1]
	if child.ParentChunkID == nil {
		t.Fatal("child.ParentChunkID should not be nil")
	}
	if *child.ParentChunkID != parent.ID {
		t.Errorf("child.ParentChunkID = %d, want %d", *child.ParentChunkID, parent.ID)
	}
}

func TestChunkHierarchical(t *testing.T) {
	c := New(Config{MaxTokens: 512, Overlap: 64})
	sections := []parser.Section{
		{
			Heading:    "Chapter 1",
			Content:    "Chapter overview content.",
			Level:      1,
			PageNumber: 1,
			Type:       "section",
			Children: []parser.Section{
				{
					Heading:    "1.1 Details",
					Content:    "Details about section one point one.",
					Level:      2,
					PageNumber: 1,
					Type:       "section",
				},
				{
					Heading:    "1.2 More Details",
					Content:    "Further information on section one point two.",
					Level:      2,
					PageNumber: 2,
					Type:       "requirement",
				},
			},
		},
	}

	chunks := c.Chunk(sections)

	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 parent chunks (1 parent + 2 children sections), got %d", len(chunks))
	}

	// The first chunk is the top-level parent.
	topParent := chunks[0]
	if topParent.ParentChunkID != nil {
		t.Error("top-level parent should have nil ParentChunkID")
	}

	// Find child section chunks whose parent is the top-level section.
	// Children sections should reference the top-level parent.
	foundChildSections := 0
	for _, ch := range chunks {
		if ch.ParentChunkID != nil && *ch.ParentChunkID == topParent.ID {
			foundChildSections++
		}
	}
	// The top parent produces child content chunks + the child section parents
	// reference it.
	if foundChildSections == 0 {
		t.Error("expected at least one chunk referencing the top-level parent")
	}
}

func TestChunkLongContent(t *testing.T) {
	c := New(Config{MaxTokens: 20, Overlap: 4})

	// Build content that exceeds MaxTokens.
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("This is sentence number. ")
	}

	sections := []parser.Section{
		{
			Heading:    "Long Section",
			Content:    sb.String(),
			Level:      1,
			PageNumber: 1,
			Type:       "section",
		},
	}

	chunks := c.Chunk(sections)

	// With very low MaxTokens, we should get multiple child chunks.
	childCount := 0
	for _, ch := range chunks {
		if ch.ParentChunkID != nil {
			childCount++
		}
	}
	if childCount < 2 {
		t.Errorf("expected multiple child chunks for long content, got %d", childCount)
	}
}

func TestChunkPreservesMetadata(t *testing.T) {
	c := New(Config{MaxTokens: 512, Overlap: 64})
	sections := []parser.Section{
		{
			Heading:    "Data Sheet",
			Content:    "Sheet data content here.",
			Level:      1,
			PageNumber: 5,
			Type:       "table",
			Metadata: map[string]string{
				"sheet_name": "Sheet1",
				"row_count":  "42",
			},
		},
	}

	chunks := c.Chunk(sections)

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	parent := chunks[0]
	if parent.Metadata == "{}" || parent.Metadata == "" {
		t.Error("parent.Metadata should contain serialised metadata")
	}
	if !strings.Contains(parent.Metadata, "Sheet1") {
		t.Errorf("parent.Metadata should contain 'Sheet1', got %q", parent.Metadata)
	}
	if !strings.Contains(parent.Metadata, "42") {
		t.Errorf("parent.Metadata should contain '42', got %q", parent.Metadata)
	}
}

func TestChunkNilMetadata(t *testing.T) {
	c := New(Config{MaxTokens: 512, Overlap: 64})
	sections := []parser.Section{
		{
			Heading: "No Meta",
			Content: "Content without metadata.",
			Type:    "section",
		},
	}

	chunks := c.Chunk(sections)

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0].Metadata != "{}" {
		t.Errorf("expected Metadata = \"{}\" for nil metadata, got %q", chunks[0].Metadata)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{"empty", "", 0},
		{"single_word", "hello", 2},        // ceil(1 * 1.3) = 2
		{"two_words", "hello world", 3},     // ceil(2 * 1.3) = 3
		{"ten_words", "a b c d e f g h i j", 13}, // ceil(10 * 1.3) = 13
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateTokens(tt.text)
			if got != tt.want {
				t.Errorf("estimateTokens(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestChunkTypesFromSection(t *testing.T) {
	tests := []struct {
		sectionType string
		wantParent  string
		wantChild   string
	}{
		{"table", "table", "table"},
		{"definition", "definition", "definition"},
		{"requirement", "requirement", "requirement"},
		{"paragraph", "paragraph", "paragraph"},
		{"section", "section", "paragraph"},
		{"unknown", "section", "paragraph"},
		{"", "section", "paragraph"},
	}

	for _, tt := range tests {
		t.Run("type_"+tt.sectionType, func(t *testing.T) {
			sec := parser.Section{Type: tt.sectionType}
			gotParent := chunkTypeFromSection(sec)
			gotChild := childChunkType(sec)
			if gotParent != tt.wantParent {
				t.Errorf("chunkTypeFromSection(Type=%q) = %q, want %q",
					tt.sectionType, gotParent, tt.wantParent)
			}
			if gotChild != tt.wantChild {
				t.Errorf("childChunkType(Type=%q) = %q, want %q",
					tt.sectionType, gotChild, tt.wantChild)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Content hash tests
// ---------------------------------------------------------------------------

func TestContentHash(t *testing.T) {
	hash1 := contentHash("hello world")
	hash2 := contentHash("hello world")
	hash3 := contentHash("different content")

	if hash1 != hash2 {
		t.Error("identical content should produce identical hashes")
	}
	if hash1 == hash3 {
		t.Error("different content should produce different hashes")
	}
	if len(hash1) != 64 {
		t.Errorf("SHA-256 hex digest should be 64 chars, got %d", len(hash1))
	}
}

// ---------------------------------------------------------------------------
// buildParentContent tests
// ---------------------------------------------------------------------------

func TestBuildParentContent(t *testing.T) {
	// Short content: heading + full content
	sec := parser.Section{
		Heading: "Test Heading",
		Content: "Short content.",
	}
	result := buildParentContent(sec)
	if !strings.Contains(result, "Test Heading") {
		t.Error("result should contain the heading")
	}
	if !strings.Contains(result, "Short content.") {
		t.Error("result should contain the full short content")
	}

	// Long content: should be truncated with "..."
	longContent := strings.Repeat("word ", 100) // 500 chars
	sec2 := parser.Section{
		Heading: "Long Section",
		Content: longContent,
	}
	result2 := buildParentContent(sec2)
	if !strings.HasSuffix(result2, "...") {
		t.Error("long content should be truncated with '...'")
	}
	if len(result2) > 300 {
		t.Errorf("truncated result should be reasonable length, got %d", len(result2))
	}

	// No heading
	sec3 := parser.Section{
		Content: "Content only.",
	}
	result3 := buildParentContent(sec3)
	if result3 != "Content only." {
		t.Errorf("expected just content, got %q", result3)
	}
}

// ---------------------------------------------------------------------------
// Default config tests
// ---------------------------------------------------------------------------

func TestNewDefaults(t *testing.T) {
	c := New(Config{})
	if c.cfg.MaxTokens != 1024 {
		t.Errorf("default MaxTokens = %d, want 1024", c.cfg.MaxTokens)
	}
	if c.cfg.Overlap != 128 {
		t.Errorf("default Overlap = %d, want 128", c.cfg.Overlap)
	}
}

func TestNewCustomConfig(t *testing.T) {
	c := New(Config{MaxTokens: 2048, Overlap: 256})
	if c.cfg.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, want 2048", c.cfg.MaxTokens)
	}
	if c.cfg.Overlap != 256 {
		t.Errorf("Overlap = %d, want 256", c.cfg.Overlap)
	}
}

// ---------------------------------------------------------------------------
// splitContent tests
// ---------------------------------------------------------------------------

func TestSplitContentShort(t *testing.T) {
	c := New(Config{MaxTokens: 512, Overlap: 64})
	fragments := c.splitContent("Short text that fits in one chunk.")
	if len(fragments) != 1 {
		t.Errorf("expected 1 fragment for short text, got %d", len(fragments))
	}
}

func TestSplitContentLong(t *testing.T) {
	c := New(Config{MaxTokens: 10, Overlap: 2})

	// Generate enough text to need splitting.
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		sb.WriteString("This is paragraph number. ")
	}

	fragments := c.splitContent(sb.String())
	if len(fragments) < 2 {
		t.Errorf("expected multiple fragments, got %d", len(fragments))
	}

	// All fragments should be non-empty.
	for i, f := range fragments {
		if strings.TrimSpace(f) == "" {
			t.Errorf("fragment[%d] is empty", i)
		}
	}
}

// ---------------------------------------------------------------------------
// marshalMeta tests
// ---------------------------------------------------------------------------

func TestMarshalMeta(t *testing.T) {
	// Nil map
	result := marshalMeta(nil)
	if result != "{}" {
		t.Errorf("marshalMeta(nil) = %q, want %q", result, "{}")
	}

	// Empty map
	result = marshalMeta(map[string]string{})
	if result != "{}" {
		t.Errorf("marshalMeta(empty) = %q, want %q", result, "{}")
	}

	// Map with values
	result = marshalMeta(map[string]string{"key": "value"})
	if !strings.Contains(result, "key") || !strings.Contains(result, "value") {
		t.Errorf("marshalMeta with data = %q, expected key/value", result)
	}
}

// ---------------------------------------------------------------------------
// Position tracking tests
// ---------------------------------------------------------------------------

func TestChunkPositionInDoc(t *testing.T) {
	c := New(Config{MaxTokens: 512, Overlap: 64})
	sections := []parser.Section{
		{Heading: "A", Content: "Content A.", Type: "section", PageNumber: 1},
		{Heading: "B", Content: "Content B.", Type: "section", PageNumber: 2},
		{Heading: "C", Content: "Content C.", Type: "section", PageNumber: 3},
	}

	chunks := c.Chunk(sections)

	// Verify positions are monotonically increasing.
	prevPos := -1
	for i, ch := range chunks {
		if ch.PositionInDoc <= prevPos {
			t.Errorf("chunk[%d].PositionInDoc = %d, expected > %d", i, ch.PositionInDoc, prevPos)
		}
		prevPos = ch.PositionInDoc
	}
}

// ---------------------------------------------------------------------------
// Empty input tests
// ---------------------------------------------------------------------------

func TestChunkEmptySections(t *testing.T) {
	c := New(Config{MaxTokens: 512, Overlap: 64})
	chunks := c.Chunk(nil)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for nil sections, got %d", len(chunks))
	}

	chunks = c.Chunk([]parser.Section{})
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty sections, got %d", len(chunks))
	}
}

// ---------------------------------------------------------------------------
// Legal helper tests
// ---------------------------------------------------------------------------

func TestDetectClauseBoundaries(t *testing.T) {
	text := `Preamble text here.
1.1 First clause of the agreement.
Some continuation text.
1.2 Second clause of the agreement.
1.2.1 Subclause detail.`

	boundaries := DetectClauseBoundaries(text)

	if len(boundaries) < 3 {
		t.Fatalf("expected at least 3 clause boundaries, got %d", len(boundaries))
	}

	// Verify that each boundary points to a position where a clause number begins.
	for i, b := range boundaries {
		remaining := text[b:]
		if !strings.HasPrefix(strings.TrimSpace(remaining), "1.") {
			t.Errorf("boundary[%d] at offset %d does not start with a clause number: %q",
				i, b, remaining[:min(30, len(remaining))])
		}
	}
}

func TestDetectClauseBoundariesNoClauses(t *testing.T) {
	text := "This text has no numbered clauses at all."
	boundaries := DetectClauseBoundaries(text)
	if len(boundaries) != 0 {
		t.Errorf("expected 0 boundaries, got %d", len(boundaries))
	}
}

func TestExtractDefinitions(t *testing.T) {
	text := `"Force Majeure" means any event beyond the reasonable control of the parties.
"Contractor" shall mean the entity providing services.
Regular text that is not a definition.
Liability: The obligation of a party to compensate for damages.`

	defs := ExtractDefinitions(text)

	if len(defs) < 2 {
		t.Fatalf("expected at least 2 definitions, got %d", len(defs))
	}

	// Check the first definition.
	foundForceMajeure := false
	foundLiability := false
	for _, d := range defs {
		if d.Term == "Force Majeure" {
			foundForceMajeure = true
			if d.LineNumber != 0 {
				t.Errorf("Force Majeure LineNumber = %d, want 0", d.LineNumber)
			}
		}
		if d.Term == "Liability" {
			foundLiability = true
		}
	}

	if !foundForceMajeure {
		t.Error("expected to find definition for 'Force Majeure'")
	}
	if !foundLiability {
		t.Error("expected to find definition for 'Liability'")
	}
}

func TestExtractDefinitionsEmpty(t *testing.T) {
	defs := ExtractDefinitions("No definitions in this text.")
	if len(defs) != 0 {
		t.Errorf("expected 0 definitions, got %d", len(defs))
	}
}

func TestSplitByClauses(t *testing.T) {
	text := `Preamble text.
1.1 First clause.
1.2 Second clause.`

	parts := SplitByClauses(text)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts (preamble + clauses), got %d", len(parts))
	}

	// First part should be the preamble.
	if !strings.Contains(parts[0], "Preamble") {
		t.Errorf("first part should be preamble, got %q", parts[0])
	}
}

func TestExtractClauseNumber(t *testing.T) {
	tests := []struct {
		text     string
		wantNum  string
		wantOK   bool
	}{
		{"1.2.3 The contractor shall...", "1.2.3", true},
		{"1.1 Scope", "1.1", true},
		{"12.3.4 Deep clause", "12.3.4", true},
		{"No clause here", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		num, ok := ExtractClauseNumber(tt.text)
		if ok != tt.wantOK {
			t.Errorf("ExtractClauseNumber(%q) ok = %v, want %v", tt.text, ok, tt.wantOK)
		}
		if num != tt.wantNum {
			t.Errorf("ExtractClauseNumber(%q) = %q, want %q", tt.text, num, tt.wantNum)
		}
	}
}

func TestClauseDepth(t *testing.T) {
	tests := []struct {
		clause string
		want   int
	}{
		{"1.1", 2},
		{"1.1.1", 3},
		{"1.2.3.4", 4},
		{"", 0},
	}

	for _, tt := range tests {
		got := ClauseDepth(tt.clause)
		if got != tt.want {
			t.Errorf("ClauseDepth(%q) = %d, want %d", tt.clause, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Engineering helper tests
// ---------------------------------------------------------------------------

func TestDetectRequirements(t *testing.T) {
	text := `The system shall operate at temperatures from -40C to 85C.
The contractor must provide documentation.
The system should support failover.
Users may optionally configure alerts.
This line has no requirements.`

	reqs := DetectRequirements(text)

	if len(reqs) < 4 {
		t.Fatalf("expected at least 4 requirements, got %d", len(reqs))
	}

	// Verify levels.
	levelMap := map[string]string{
		"SHALL": "mandatory",
		"MUST":  "mandatory",
		"SHOULD": "recommended",
		"MAY":   "optional",
	}

	for _, req := range reqs {
		expectedLevel, ok := levelMap[req.Keyword]
		if ok && req.Level != expectedLevel {
			t.Errorf("requirement keyword %q has level %q, want %q",
				req.Keyword, req.Level, expectedLevel)
		}
	}
}

func TestDetectRequirementsEmpty(t *testing.T) {
	reqs := DetectRequirements("No normative language here.")
	if len(reqs) != 0 {
		t.Errorf("expected 0 requirements, got %d", len(reqs))
	}
}

func TestIsRequirement(t *testing.T) {
	if !IsRequirement("The system shall perform validation.") {
		t.Error("expected IsRequirement = true for 'shall'")
	}
	if !IsRequirement("Users MUST authenticate.") {
		t.Error("expected IsRequirement = true for 'MUST'")
	}
	if IsRequirement("This is a regular sentence.") {
		t.Error("expected IsRequirement = false for regular text")
	}
}

func TestDetectStandardsReferences(t *testing.T) {
	text := `The system complies with ISO 9001:2015 and IEEE 802.11.
Materials per ASTM D1234 and MIL-STD-810G.
Electrical per IEC 61508 and NFPA 70.
Welding per AWS D1.1 and ASME B31.3.`

	refs := DetectStandardsReferences(text)

	if len(refs) < 6 {
		t.Fatalf("expected at least 6 standards references, got %d", len(refs))
	}

	// Check that specific standards were found.
	foundISO := false
	foundIEEE := false
	foundASTM := false
	foundMIL := false
	for _, ref := range refs {
		switch ref.Body {
		case "ISO":
			foundISO = true
			if !strings.Contains(ref.Standard, "ISO") {
				t.Errorf("ISO ref Standard = %q, expected to contain 'ISO'", ref.Standard)
			}
		case "IEEE":
			foundIEEE = true
		case "ASTM":
			foundASTM = true
		case "MIL":
			foundMIL = true
		}
	}

	if !foundISO {
		t.Error("expected to find ISO standard reference")
	}
	if !foundIEEE {
		t.Error("expected to find IEEE standard reference")
	}
	if !foundASTM {
		t.Error("expected to find ASTM standard reference")
	}
	if !foundMIL {
		t.Error("expected to find MIL standard reference")
	}
}

func TestDetectStandardsReferencesEmpty(t *testing.T) {
	refs := DetectStandardsReferences("No standards referenced here.")
	if len(refs) != 0 {
		t.Errorf("expected 0 references, got %d", len(refs))
	}
}

func TestHasStandardsReference(t *testing.T) {
	if !HasStandardsReference("Per ISO 9001 requirements.") {
		t.Error("expected true for ISO reference")
	}
	if HasStandardsReference("No standards here.") {
		t.Error("expected false for no standards")
	}
}

// ---------------------------------------------------------------------------
// Structure helper tests
// ---------------------------------------------------------------------------

func TestIsHeading(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"numbered_single", "1. Introduction", true},
		{"numbered_multi", "1.2. Requirements", true},
		{"numbered_deep", "1.2.3. Details", true},
		{"all_caps", "INTRODUCTION", true},
		{"all_caps_multi", "TERMS AND CONDITIONS", true},
		{"markdown_h1", "# Main Title", true},
		{"markdown_h2", "## Subsection", true},
		{"appendix", "Appendix A Reference Data", true},
		{"annex", "Annex 1 Supporting Documents", true},
		{"article", "Article IV Obligations", true},
		{"regular_text", "This is a normal sentence.", false},
		{"empty", "", false},
		{"short_caps", "AB", false}, // too short for caps pattern
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsHeading(tt.line)
			if got != tt.want {
				t.Errorf("IsHeading(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestContentType(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "table_pipes",
			text: "| Col1 | Col2 | Col3 |\n| --- | --- | --- |\n| a | b | c |",
			want: "table",
		},
		{
			name: "table_tabs",
			text: "A\tB\tC\nD\tE\tF\nG\tH\tI",
			want: "table",
		},
		{
			name: "definition_means",
			text: `"Force Majeure" means any event beyond control.`,
			want: "definition",
		},
		{
			name: "requirement_shall",
			text: "The system SHALL operate continuously.",
			want: "requirement",
		},
		{
			name: "requirement_must",
			text: "The contractor MUST deliver documentation.",
			want: "requirement",
		},
		{
			name: "section_with_heading",
			text: "INTRODUCTION\nSome paragraph text.",
			want: "section",
		},
		{
			name: "plain_paragraph",
			text: "This is just a regular paragraph of text.",
			want: "paragraph",
		},
		{
			name: "empty",
			text: "",
			want: "paragraph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContentType(tt.text)
			if got != tt.want {
				t.Errorf("ContentType(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestDetectNumbering(t *testing.T) {
	tests := []struct {
		line    string
		wantNum string
		wantOK  bool
	}{
		{"1. Introduction", "1", true},
		{"1.2. Details", "1.2", true},
		{"1.2.3. Deep", "1.2.3", true},
		{"Regular text", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		num, ok := DetectNumbering(tt.line)
		if ok != tt.wantOK || num != tt.wantNum {
			t.Errorf("DetectNumbering(%q) = (%q, %v), want (%q, %v)",
				tt.line, num, ok, tt.wantNum, tt.wantOK)
		}
	}
}

func TestNumberingLevel(t *testing.T) {
	tests := []struct {
		numbering string
		want      int
	}{
		{"1", 1},
		{"1.2", 2},
		{"1.2.3", 3},
		{"", 0},
	}

	for _, tt := range tests {
		got := NumberingLevel(tt.numbering)
		if got != tt.want {
			t.Errorf("NumberingLevel(%q) = %d, want %d", tt.numbering, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Cross-reference detection tests
// ---------------------------------------------------------------------------

func TestDetectCrossReferences(t *testing.T) {
	text := "See clause 1.2.3 for details. Refer to section 4.5 and article IV."

	refs := DetectCrossReferences(text)
	if len(refs) < 3 {
		t.Fatalf("expected at least 3 cross-references, got %d", len(refs))
	}

	foundClause := false
	foundSection := false
	foundArticle := false
	for _, ref := range refs {
		switch ref.Type {
		case "clause":
			foundClause = true
			if ref.Target != "1.2.3" {
				t.Errorf("clause target = %q, want %q", ref.Target, "1.2.3")
			}
		case "section":
			foundSection = true
			if ref.Target != "4.5" {
				t.Errorf("section target = %q, want %q", ref.Target, "4.5")
			}
		case "article":
			foundArticle = true
		}
	}
	if !foundClause {
		t.Error("expected to find clause cross-reference")
	}
	if !foundSection {
		t.Error("expected to find section cross-reference")
	}
	if !foundArticle {
		t.Error("expected to find article cross-reference")
	}
}

func TestHasCrossReferences(t *testing.T) {
	if !HasCrossReferences("See clause 1.2 for details.") {
		t.Error("expected true for text with clause reference")
	}
	if HasCrossReferences("No references at all.") {
		t.Error("expected false for text with no references")
	}
}

// ---------------------------------------------------------------------------
// Table detection tests (engineering.go)
// ---------------------------------------------------------------------------

func TestDetectTables(t *testing.T) {
	text := "Some intro text.\n| A | B | C |\n| --- | --- | --- |\n| 1 | 2 | 3 |\nMore text."

	tables := DetectTables(text)
	if len(tables) == 0 {
		t.Fatal("expected at least 1 table detected")
	}
	if !tables[0].HasHeaders {
		t.Error("expected HasHeaders = true for markdown table with separator")
	}
}

func TestPreserveTableChunks(t *testing.T) {
	text := "Before table.\n| A | B |\n| --- | --- |\n| 1 | 2 |\nAfter table."

	fragments := PreserveTableChunks(text)
	if len(fragments) < 2 {
		t.Fatalf("expected at least 2 fragments (prose + table), got %d", len(fragments))
	}

	// Verify the table is preserved as one atomic fragment.
	foundTable := false
	for _, f := range fragments {
		if strings.Contains(f, "| A | B |") && strings.Contains(f, "| 1 | 2 |") {
			foundTable = true
		}
	}
	if !foundTable {
		t.Error("expected to find an atomic table fragment")
	}
}

func TestPreserveTableChunksNoTable(t *testing.T) {
	text := "Plain text with no tables at all."
	fragments := PreserveTableChunks(text)
	if len(fragments) != 1 {
		t.Errorf("expected 1 fragment for text without tables, got %d", len(fragments))
	}
	if fragments[0] != text {
		t.Errorf("fragment should be the original text")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
