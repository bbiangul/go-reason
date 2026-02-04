package parser

import (
	"context"
	"fmt"
)

// LegacyParser routes legacy binary formats to an external service.
type LegacyParser struct{}

func (p *LegacyParser) SupportedFormats() []string { return []string{"doc", "xls", "ppt"} }

func (p *LegacyParser) Parse(ctx context.Context, path string) (*ParseResult, error) {
	return nil, fmt.Errorf("legacy format requires external parser (LlamaParse); configure llamaparse in config")
}
