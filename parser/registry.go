package parser

import "fmt"

type LlamaParseConfig struct {
	APIKey  string
	BaseURL string
}

type Registry struct {
	parsers    map[string]Parser
	llamaParse *LlamaParseConfig
}

func NewRegistry() *Registry {
	r := &Registry{parsers: make(map[string]Parser)}
	// Register built-in parsers
	pdf := &PDFParser{}
	docx := &DOCXParser{}
	xlsx := &XLSXParser{}
	pptx := &PPTXParser{}

	for _, p := range []Parser{pdf, docx, xlsx, pptx} {
		for _, f := range p.SupportedFormats() {
			r.parsers[f] = p
		}
	}
	return r
}

func (r *Registry) SetLlamaParse(cfg LlamaParseConfig) {
	r.llamaParse = &cfg
	lp := &LlamaParseParser{cfg: cfg}
	// Register legacy formats
	for _, f := range lp.SupportedFormats() {
		r.parsers[f] = lp
	}
}

func (r *Registry) Get(format string) (Parser, error) {
	p, ok := r.parsers[format]
	if !ok {
		return nil, fmt.Errorf("no parser for format: %s", format)
	}
	return p, nil
}

func (r *Registry) Register(format string, p Parser) {
	r.parsers[format] = p
}
