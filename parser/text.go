package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// TextParser handles plain text (.txt) files.
type TextParser struct{}

func (p *TextParser) SupportedFormats() []string { return []string{"txt"} }

func (p *TextParser) Parse(ctx context.Context, path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading text file: %w", err)
	}

	content := string(data)
	if content == "" {
		return &ParseResult{
			Method: "native",
		}, nil
	}

	return &ParseResult{
		Sections: []Section{
			{
				Heading: filepath.Base(path),
				Content: content,
				Level:   1,
				Type:    "paragraph",
			},
		},
		Method: "native",
	}, nil
}
