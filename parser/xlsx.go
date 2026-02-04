package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

type XLSXParser struct{}

func (p *XLSXParser) SupportedFormats() []string { return []string{"xlsx", "xls"} }

func (p *XLSXParser) Parse(ctx context.Context, path string) (*ParseResult, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("opening XLSX: %w", err)
	}
	defer f.Close()

	var sections []Section

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}

		if len(rows) == 0 {
			continue
		}

		var content strings.Builder
		for _, row := range rows {
			content.WriteString("| " + strings.Join(row, " | ") + " |\n")
		}

		sections = append(sections, Section{
			Heading: sheet,
			Content: content.String(),
			Type:    "table",
			Level:   1,
			Metadata: map[string]string{
				"sheet_name": sheet,
				"row_count":  fmt.Sprintf("%d", len(rows)),
			},
		})
	}

	if len(sections) == 0 {
		return nil, fmt.Errorf("no data found in XLSX")
	}

	return &ParseResult{
		Sections: sections,
		Method:   "native",
	}, nil
}
