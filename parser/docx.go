package parser

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type DOCXParser struct{}

func (p *DOCXParser) SupportedFormats() []string { return []string{"docx"} }

func (p *DOCXParser) Parse(ctx context.Context, path string) (*ParseResult, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("opening DOCX: %w", err)
	}
	defer r.Close()

	// Find word/document.xml
	var docFile *zip.File
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			docFile = f
			break
		}
	}
	if docFile == nil {
		return nil, fmt.Errorf("word/document.xml not found in DOCX")
	}

	rc, err := docFile.Open()
	if err != nil {
		return nil, fmt.Errorf("opening document.xml: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	sections, err := parseDocxXML(data)
	if err != nil {
		return nil, fmt.Errorf("parsing DOCX XML: %w", err)
	}

	return &ParseResult{
		Sections: sections,
		Method:   "native",
	}, nil
}

// DOCX XML structures (simplified)
type docxBody struct {
	XMLName xml.Name    `xml:"body"`
	Paras   []docxPara  `xml:"p"`
	Tables  []docxTable `xml:"tbl"`
}

type docxDocument struct {
	XMLName xml.Name `xml:"document"`
	Body    docxBody `xml:"body"`
}

type docxPara struct {
	XMLName xml.Name    `xml:"p"`
	PPr     *docxParaPr `xml:"pPr"`
	Runs    []docxRun   `xml:"r"`
}

type docxParaPr struct {
	PStyle *docxPStyle `xml:"pStyle"`
}

type docxPStyle struct {
	Val string `xml:"val,attr"`
}

type docxRun struct {
	Text []docxText `xml:"t"`
}

type docxText struct {
	Content string `xml:",chardata"`
}

type docxTable struct {
	Rows []docxRow `xml:"tr"`
}

type docxRow struct {
	Cells []docxCell `xml:"tc"`
}

type docxCell struct {
	Paras []docxPara `xml:"p"`
}

func parseDocxXML(data []byte) ([]Section, error) {
	var doc docxDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	var sections []Section
	var currentContent strings.Builder
	var currentHeading string
	currentLevel := 0

	for _, para := range doc.Body.Paras {
		text := extractParaText(para)
		if text == "" {
			continue
		}

		style := ""
		if para.PPr != nil && para.PPr.PStyle != nil {
			style = para.PPr.PStyle.Val
		}

		isHeading := strings.HasPrefix(strings.ToLower(style), "heading") ||
			strings.HasPrefix(strings.ToLower(style), "title")

		if isHeading {
			// Save previous section
			if currentContent.Len() > 0 || currentHeading != "" {
				sections = append(sections, Section{
					Heading: currentHeading,
					Content: strings.TrimSpace(currentContent.String()),
					Level:   currentLevel,
					Type:    classifySectionType(currentHeading, currentContent.String()),
				})
				currentContent.Reset()
			}
			currentHeading = text
			currentLevel = headingStyleLevel(style)
		} else {
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			currentContent.WriteString(text)
		}
	}

	// Process tables
	for _, tbl := range doc.Body.Tables {
		var tableContent strings.Builder
		for _, row := range tbl.Rows {
			cells := make([]string, 0, len(row.Cells))
			for _, cell := range row.Cells {
				var cellText strings.Builder
				for _, p := range cell.Paras {
					t := extractParaText(p)
					if cellText.Len() > 0 {
						cellText.WriteString(" ")
					}
					cellText.WriteString(t)
				}
				cells = append(cells, cellText.String())
			}
			tableContent.WriteString("| " + strings.Join(cells, " | ") + " |\n")
		}
		sections = append(sections, Section{
			Content: tableContent.String(),
			Type:    "table",
		})
	}

	// Final section
	if currentContent.Len() > 0 {
		sections = append(sections, Section{
			Heading: currentHeading,
			Content: strings.TrimSpace(currentContent.String()),
			Level:   currentLevel,
			Type:    classifySectionType(currentHeading, currentContent.String()),
		})
	}

	return sections, nil
}

func extractParaText(para docxPara) string {
	var b strings.Builder
	for _, run := range para.Runs {
		for _, t := range run.Text {
			b.WriteString(t.Content)
		}
	}
	return b.String()
}

func headingStyleLevel(style string) int {
	lower := strings.ToLower(style)
	if strings.Contains(lower, "title") {
		return 1
	}
	// Extract number from "Heading1", "Heading2", etc.
	for i := 1; i <= 9; i++ {
		if strings.Contains(lower, fmt.Sprintf("%d", i)) {
			return i
		}
	}
	return 1
}
