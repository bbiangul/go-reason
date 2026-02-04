package parser

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/bbiangul/go-reason/llm"
)

// PDFVisionParser uses a vision LLM to extract text from complex PDF pages
// (tables, diagrams, multi-column layouts).
type PDFVisionParser struct {
	visionProvider llm.VisionProvider
}

func NewPDFVisionParser(provider llm.VisionProvider) *PDFVisionParser {
	return &PDFVisionParser{visionProvider: provider}
}

func (p *PDFVisionParser) SupportedFormats() []string { return []string{"pdf"} }

func (p *PDFVisionParser) Parse(ctx context.Context, path string) (*ParseResult, error) {
	// Read the PDF as binary to send to vision model
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading PDF for vision: %w", err)
	}

	b64 := base64.StdEncoding.EncodeToString(data)

	resp, err := p.visionProvider.ChatWithImages(ctx, llm.VisionChatRequest{
		Messages: []llm.VisionMessage{
			{
				Role: "user",
				Content: []llm.ContentPart{
					{
						Type: "text",
						Text: `Extract all text content from this PDF page. Preserve the structure:
- For tables, format as markdown tables
- For headings, prefix with appropriate markdown heading levels
- For lists, use markdown list format
- For diagrams, describe the content in [Diagram: ...] blocks
- Preserve section numbering`,
					},
					{
						Type: "image_url",
						ImageURL: &llm.ImageURL{
							URL: "data:application/pdf;base64," + b64,
						},
					},
				},
			},
		},
		MaxTokens: 4096,
	})
	if err != nil {
		return nil, fmt.Errorf("vision extraction failed: %w", err)
	}

	// Parse the vision output into sections
	sections := splitPageIntoSections(resp.Content, 1)

	return &ParseResult{
		Sections: sections,
		Method:   "vision",
	}, nil
}
