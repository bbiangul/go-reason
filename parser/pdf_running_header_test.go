package parser

import (
	"strings"
	"testing"
)

func TestFixRunningHeaders_BasicReplacement(t *testing.T) {
	// Simulates a document where "DOC TITLE" appears on every page as a
	// running header. Section "4.1 Tracker" starts on page 5, continues
	// onto page 6 where the running header displaces it.
	sections := []Section{
		{Heading: "DOC TITLE", Content: "intro", PageNumber: 1, Level: 1},
		{Heading: "1.0 Overview", Content: "overview text", PageNumber: 1, Level: 1},
		{Heading: "DOC TITLE", Content: "more overview", PageNumber: 2, Level: 1},
		{Heading: "2.0 Details", Content: "details text", PageNumber: 2, Level: 1},
		{Heading: "DOC TITLE", Content: "details cont", PageNumber: 3, Level: 1},
		{Heading: "3.0 Specs", Content: "specs text", PageNumber: 3, Level: 1},
		{Heading: "DOC TITLE", Content: "specs cont", PageNumber: 4, Level: 1},
		{Heading: "4.0 Components", Content: "components", PageNumber: 4, Level: 1},
		{Heading: "DOC TITLE", Content: "components cont", PageNumber: 5, Level: 1},
		{Heading: "4.1 Tracker", Content: "tracker overview", PageNumber: 5, Level: 2},
		// Page 6: running header displaces "4.1 Tracker"
		{Heading: "DOC TITLE", Content: "fusibles 6.3A, 16 entradas, 16 salidas", PageNumber: 6, Level: 1},
		{Heading: "DOC TITLE", Content: "more content", PageNumber: 7, Level: 1},
	}

	result := fixRunningHeaders(sections, 7)

	// The running header "DOC TITLE" should be replaced with the last real heading.
	// On page 6, the last real heading was "4.1 Tracker" from page 5.
	for _, s := range result {
		if s.PageNumber == 6 && strings.Contains(s.Content, "fusibles") {
			if s.Heading != "4.1 Tracker" {
				t.Errorf("page 6 (fusibles): expected heading %q, got %q", "4.1 Tracker", s.Heading)
			}
			return
		}
	}
	t.Error("did not find the fusibles section on page 6")
}

func TestFixRunningHeaders_ThresholdDetection(t *testing.T) {
	// A heading appearing on exactly 3 pages out of 12 (25%) should be
	// detected as running if threshold = max(3, 12/4) = 3.
	sections := []Section{
		{Heading: "REPEATED", Content: "a", PageNumber: 1, Level: 1},
		{Heading: "1.0 Real", Content: "b", PageNumber: 2, Level: 1},
		{Heading: "REPEATED", Content: "c", PageNumber: 5, Level: 1},
		{Heading: "REPEATED", Content: "d", PageNumber: 9, Level: 1},
	}

	result := fixRunningHeaders(sections, 12)

	// "REPEATED" appears on 3 pages, threshold is max(3, 3) = 3.
	// Pages 5 and 9 should get "1.0 Real" carried over.
	for _, s := range result {
		if s.Content == "c" && s.Heading != "1.0 Real" {
			t.Errorf("page 5: expected heading %q, got %q", "1.0 Real", s.Heading)
		}
		if s.Content == "d" && s.Heading != "1.0 Real" {
			t.Errorf("page 9: expected heading %q, got %q", "1.0 Real", s.Heading)
		}
	}
}

func TestFixRunningHeaders_BelowThreshold(t *testing.T) {
	// A heading appearing on only 2 pages out of 20 should NOT be treated
	// as a running header (threshold = max(3, 5) = 5).
	sections := []Section{
		{Heading: "APPEARS TWICE", Content: "a", PageNumber: 1, Level: 1},
		{Heading: "1.0 Chapter", Content: "b", PageNumber: 5, Level: 1},
		{Heading: "APPEARS TWICE", Content: "c", PageNumber: 10, Level: 1},
	}

	result := fixRunningHeaders(sections, 20)

	// Should be unchanged — "APPEARS TWICE" is not frequent enough.
	for _, s := range result {
		if s.Content == "c" && s.Heading != "APPEARS TWICE" {
			t.Errorf("should not replace infrequent heading, got %q", s.Heading)
		}
	}
}

func TestFixRunningHeaders_NoRunningHeaders(t *testing.T) {
	// Document with no repeated headings — nothing should change.
	sections := []Section{
		{Heading: "1.0 Intro", Content: "a", PageNumber: 1, Level: 1},
		{Heading: "2.0 Body", Content: "b", PageNumber: 2, Level: 1},
		{Heading: "3.0 Conclusion", Content: "c", PageNumber: 3, Level: 1},
	}

	result := fixRunningHeaders(sections, 3)

	for i, s := range result {
		if s.Heading != sections[i].Heading {
			t.Errorf("section %d: heading changed from %q to %q", i, sections[i].Heading, s.Heading)
		}
	}
}

func TestFixRunningHeaders_EmptySections(t *testing.T) {
	result := fixRunningHeaders(nil, 0)
	if len(result) != 0 {
		t.Error("expected nil/empty result for nil input")
	}

	result = fixRunningHeaders([]Section{}, 10)
	if len(result) != 0 {
		t.Error("expected empty result for empty input")
	}
}

func TestFixRunningHeaders_FirstPageRunningHeader(t *testing.T) {
	// If the very first section is a running header and there's no prior
	// real heading to carry over, the heading should remain unchanged
	// (no previous heading to inherit).
	sections := []Section{
		{Heading: "DOC TITLE", Content: "page 1 intro", PageNumber: 1, Level: 1},
		{Heading: "DOC TITLE", Content: "page 2 intro", PageNumber: 2, Level: 1},
		{Heading: "DOC TITLE", Content: "page 3 intro", PageNumber: 3, Level: 1},
		{Heading: "1.0 First Real", Content: "content", PageNumber: 4, Level: 1},
		{Heading: "DOC TITLE", Content: "continuation", PageNumber: 5, Level: 1},
	}

	result := fixRunningHeaders(sections, 5)

	// Pages 1-3: no prior real heading, should keep "DOC TITLE"
	for _, s := range result {
		if s.Content == "page 1 intro" && s.Heading != "DOC TITLE" {
			t.Errorf("page 1: should keep original heading when no prior real heading, got %q", s.Heading)
		}
		if s.Content == "page 2 intro" && s.Heading != "DOC TITLE" {
			t.Errorf("page 2: should keep original heading when no prior real heading, got %q", s.Heading)
		}
	}

	// Page 5: should inherit "1.0 First Real"
	for _, s := range result {
		if s.Content == "continuation" && s.Heading != "1.0 First Real" {
			t.Errorf("page 5: expected %q, got %q", "1.0 First Real", s.Heading)
		}
	}
}

func TestFixRunningHeaders_MultipleRunningHeaders(t *testing.T) {
	// Some PDFs have both a header and a footer repeated on every page.
	sections := []Section{
		{Heading: "HEADER TEXT", Content: "", PageNumber: 1, Level: 1},
		{Heading: "1.0 Intro", Content: "intro text", PageNumber: 1, Level: 1},
		{Heading: "FOOTER TEXT", Content: "", PageNumber: 1, Level: 1},
		{Heading: "HEADER TEXT", Content: "", PageNumber: 2, Level: 1},
		{Heading: "2.0 Body", Content: "body text", PageNumber: 2, Level: 1},
		{Heading: "FOOTER TEXT", Content: "", PageNumber: 2, Level: 1},
		{Heading: "HEADER TEXT", Content: "", PageNumber: 3, Level: 1},
		{Heading: "FOOTER TEXT", Content: "", PageNumber: 3, Level: 1},
		{Heading: "HEADER TEXT", Content: "continuation", PageNumber: 4, Level: 1},
		{Heading: "FOOTER TEXT", Content: "", PageNumber: 4, Level: 1},
	}

	result := fixRunningHeaders(sections, 4)

	// Both "HEADER TEXT" and "FOOTER TEXT" appear on 4 pages each → running.
	// Page 4 "continuation" should inherit "2.0 Body" from page 2.
	for _, s := range result {
		if s.Content == "continuation" && s.Heading != "2.0 Body" {
			t.Errorf("page 4 continuation: expected %q, got %q", "2.0 Body", s.Heading)
		}
	}
}

func TestFixRunningHeaders_PreservesLevel(t *testing.T) {
	// When a running header is replaced, the level should be updated
	// to match the carried-over heading.
	sections := []Section{
		{Heading: "DOC TITLE", Content: "a", PageNumber: 1, Level: 1},
		{Heading: "DOC TITLE", Content: "b", PageNumber: 2, Level: 1},
		{Heading: "DOC TITLE", Content: "c", PageNumber: 3, Level: 1},
		{Heading: "4.1.2 Subsection", Content: "detail", PageNumber: 4, Level: 3},
		{Heading: "DOC TITLE", Content: "continuation", PageNumber: 5, Level: 1},
	}

	result := fixRunningHeaders(sections, 5)

	for _, s := range result {
		if s.Content == "continuation" {
			if s.Level != 3 {
				t.Errorf("expected level 3 (from 4.1.2), got %d", s.Level)
			}
			if s.Heading != "4.1.2 Subsection" {
				t.Errorf("expected heading %q, got %q", "4.1.2 Subsection", s.Heading)
			}
		}
	}
}

func TestFixRunningHeaders_NormalizeTrailingGarbage(t *testing.T) {
	// PDF extraction often leaves trailing non-ASCII artifacts.
	// normalizeHeading should strip them for matching.
	tests := []struct {
		input    string
		expected string
	}{
		{"MANUAL TÉCNICO\uf0d2", "MANUAL TÉCNICO"},
		{"MANUAL TÉCNICO\ufffd", "MANUAL TÉCNICO"},
		{"MANUAL TÉCNICO  ", "MANUAL TÉCNICO"},
		{"Clean Heading", "Clean Heading"},
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeHeading(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeHeading(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFixRunningHeaders_TrailingGarbageMatches(t *testing.T) {
	// Same heading with and without trailing garbage should match.
	sections := []Section{
		{Heading: "DOC TITLE\uf0d2", Content: "a", PageNumber: 1, Level: 1},
		{Heading: "1.0 Real", Content: "b", PageNumber: 2, Level: 1},
		{Heading: "DOC TITLE", Content: "c", PageNumber: 3, Level: 1},
		{Heading: "DOC TITLE\uf0d2", Content: "d", PageNumber: 4, Level: 1},
	}

	result := fixRunningHeaders(sections, 4)

	// "DOC TITLE" (with/without garbage) appears on 3 pages → running.
	for _, s := range result {
		if s.Content == "c" && s.Heading != "1.0 Real" {
			t.Errorf("page 3: expected %q, got %q", "1.0 Real", s.Heading)
		}
		if s.Content == "d" && s.Heading != "1.0 Real" {
			t.Errorf("page 4: expected %q, got %q", "1.0 Real", s.Heading)
		}
	}
}

func TestFixRunningHeaders_ShortDocument(t *testing.T) {
	// A 2-page document: even if a heading appears on both pages, the
	// threshold is max(3, 0) = 3 so it won't be treated as running.
	sections := []Section{
		{Heading: "TITLE", Content: "a", PageNumber: 1, Level: 1},
		{Heading: "TITLE", Content: "b", PageNumber: 2, Level: 1},
	}

	result := fixRunningHeaders(sections, 2)

	// Should be unchanged — only 2 pages, threshold is 3.
	if result[0].Heading != "TITLE" || result[1].Heading != "TITLE" {
		t.Error("short document headings should not be changed")
	}
}

func TestFixRunningHeaders_HeadingOnlyOnSamePage(t *testing.T) {
	// A heading appearing multiple times but all on the SAME page should
	// not be treated as a running header (it only counts distinct pages).
	sections := []Section{
		{Heading: "REPEATED", Content: "a", PageNumber: 1, Level: 1},
		{Heading: "REPEATED", Content: "b", PageNumber: 1, Level: 1},
		{Heading: "REPEATED", Content: "c", PageNumber: 1, Level: 1},
		{Heading: "1.0 Real", Content: "d", PageNumber: 2, Level: 1},
	}

	result := fixRunningHeaders(sections, 10)

	// "REPEATED" appears on only 1 distinct page — not running.
	if result[0].Heading != "REPEATED" {
		t.Errorf("same-page repetition should not trigger running header, got %q", result[0].Heading)
	}
}

func TestFixRunningHeaders_ChainedCarryOver(t *testing.T) {
	// Running header appears across many pages. Real headings change
	// midway — verify the carry-over updates correctly.
	sections := []Section{
		{Heading: "RH", Content: "a", PageNumber: 1, Level: 1},
		{Heading: "RH", Content: "b", PageNumber: 2, Level: 1},
		{Heading: "RH", Content: "c", PageNumber: 3, Level: 1},
		{Heading: "1.0 First", Content: "d", PageNumber: 4, Level: 1},
		{Heading: "RH", Content: "e", PageNumber: 5, Level: 1},  // should get "1.0 First"
		{Heading: "2.0 Second", Content: "f", PageNumber: 6, Level: 1},
		{Heading: "RH", Content: "g", PageNumber: 7, Level: 1},  // should get "2.0 Second"
		{Heading: "RH", Content: "h", PageNumber: 8, Level: 1},  // should get "2.0 Second"
	}

	result := fixRunningHeaders(sections, 8)

	expected := map[string]string{
		"e": "1.0 First",
		"g": "2.0 Second",
		"h": "2.0 Second",
	}

	for _, s := range result {
		if want, ok := expected[s.Content]; ok {
			if s.Heading != want {
				t.Errorf("content %q: expected heading %q, got %q", s.Content, want, s.Heading)
			}
		}
	}
}

func TestFixRunningHeaders_DoesNotAffectContent(t *testing.T) {
	// Verify that only Heading and Level are modified — Content, PageNumber,
	// and Type are never touched.
	sections := []Section{
		{Heading: "RH", Content: "alpha", PageNumber: 1, Level: 1, Type: "section"},
		{Heading: "RH", Content: "beta", PageNumber: 2, Level: 1, Type: "table"},
		{Heading: "RH", Content: "gamma", PageNumber: 3, Level: 1, Type: "annex"},
		{Heading: "1.0 Real", Content: "delta", PageNumber: 4, Level: 2, Type: "section"},
		{Heading: "RH", Content: "epsilon", PageNumber: 5, Level: 1, Type: "requirement"},
	}

	result := fixRunningHeaders(sections, 5)

	// Content, PageNumber, Type should be preserved exactly.
	if result[0].Content != "alpha" || result[0].PageNumber != 1 || result[0].Type != "section" {
		t.Error("section 0 metadata was modified")
	}
	if result[3].Content != "delta" || result[3].PageNumber != 4 || result[3].Type != "section" {
		t.Error("section 3 metadata was modified")
	}
	if result[4].Content != "epsilon" || result[4].PageNumber != 5 || result[4].Type != "requirement" {
		t.Error("section 4 metadata was modified")
	}
}

func TestFixRunningHeaders_MergedHeadingsNotRunning(t *testing.T) {
	// Headings produced by the empty-content merge (e.g. "3.9 Desc — 3.9.1 Modelo A")
	// should NOT be treated as running headers even if a prefix matches.
	sections := []Section{
		{Heading: "DOC TITLE", Content: "a", PageNumber: 1, Level: 1},
		{Heading: "DOC TITLE", Content: "b", PageNumber: 2, Level: 1},
		{Heading: "DOC TITLE", Content: "c", PageNumber: 3, Level: 1},
		{Heading: "3.9 Desc — 3.9.1 Modelo A — 3.9.1.1 Material:", Content: "Peso: 153kg", PageNumber: 4, Level: 3},
		{Heading: "DOC TITLE", Content: "continuation", PageNumber: 5, Level: 1},
	}

	result := fixRunningHeaders(sections, 5)

	// The merged heading should be untouched.
	for _, s := range result {
		if strings.Contains(s.Content, "153kg") {
			if !strings.Contains(s.Heading, "Modelo A") {
				t.Errorf("merged heading should be preserved, got %q", s.Heading)
			}
		}
	}

	// Page 5 should inherit the merged heading.
	for _, s := range result {
		if s.Content == "continuation" {
			if !strings.Contains(s.Heading, "Modelo A") {
				t.Errorf("page 5 should inherit merged heading, got %q", s.Heading)
			}
		}
	}
}

func TestFixRunningHeaders_ALTAVisionScenario(t *testing.T) {
	// Reproduces the exact ALTAVision bug: "MANUAL TÉCNICO AV-FM, AV-FF"
	// appears on almost every page. Section "4.1 Tarjeta de Control / Tracker"
	// starts on page 45, but page 46 gets "MANUAL TÉCNICO..." as heading.
	title := "MANUAL TÉCNICO AV-FM, AV-FF"

	var sections []Section
	// Pages 1-44: running header + various real sections
	for i := 1; i <= 44; i++ {
		sections = append(sections, Section{
			Heading: title, Content: "page content", PageNumber: i, Level: 1,
		})
	}
	// Page 45: running header, then real section
	sections = append(sections,
		Section{Heading: title, Content: "", PageNumber: 45, Level: 1},
		Section{Heading: "4.1 Tarjeta de Control / Tracker (P/N: E1375.1)", Content: "tracker overview and labels", PageNumber: 45, Level: 2},
	)
	// Page 46: running header with Tracker continuation content
	sections = append(sections, Section{
		Heading:    title,
		Content:    "Fusibles:\nSe proporcionan fusibles de 6.3Amp/250V\n16 Entradas y 16 Salidas",
		PageNumber: 46,
		Level:      1,
	})
	// Pages 47-50: more running headers
	for i := 47; i <= 50; i++ {
		sections = append(sections, Section{
			Heading: title, Content: "more content", PageNumber: i, Level: 1,
		})
	}

	result := fixRunningHeaders(sections, 50)

	// The Tracker continuation on page 46 should now have the Tracker heading.
	for _, s := range result {
		if s.PageNumber == 46 && strings.Contains(s.Content, "Fusibles") {
			if !strings.Contains(s.Heading, "Tracker") {
				t.Errorf("page 46 Tracker content: expected heading containing 'Tracker', got %q", s.Heading)
			}
			return
		}
	}
	t.Error("did not find the Tracker continuation section on page 46")
}
