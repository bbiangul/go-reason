package parser

import "context"

// ExtractedImage represents an image extracted from a document during parsing.
type ExtractedImage struct {
	Data         []byte
	MIMEType     string // "image/jpeg" or "image/png"
	PageNumber   int    // page/slide number (0 for DOCX)
	SectionIndex int    // index into ParseResult.Sections this image belongs to
	Width        int
	Height       int
}

// ParseResult is what a parser produces from a document file.
type ParseResult struct {
	Sections []Section         // Ordered sections extracted from the document
	Images   []ExtractedImage  // Images extracted from the document
	Method   string            // "native", "llamaparse", "vision"
	Metadata map[string]string
}

// Section represents a logical section of a parsed document.
type Section struct {
	Heading    string
	Content    string
	Level      int    // Heading level (1=top, 2=sub, etc.)
	PageNumber int
	Type       string // "section", "table", "definition", "requirement", "paragraph"
	Children   []Section
	Metadata   map[string]string
}

// Parser can parse a specific document format.
type Parser interface {
	Parse(ctx context.Context, path string) (*ParseResult, error)
	SupportedFormats() []string
}
