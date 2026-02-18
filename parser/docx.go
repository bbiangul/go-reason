package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"path/filepath"
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

	// Build file index for quick lookup
	fileIndex := make(map[string]*zip.File, len(r.File))
	for _, f := range r.File {
		fileIndex[f.Name] = f
	}

	// Find word/document.xml
	docFile := fileIndex["word/document.xml"]
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

	// Parse relationships for image references
	rels := parseDocxRels(fileIndex)

	sections, err := parseDocxXML(data)
	if err != nil {
		return nil, fmt.Errorf("parsing DOCX XML: %w", err)
	}

	// Extract images
	images := extractDocxImages(data, rels, fileIndex, len(sections))

	return &ParseResult{
		Sections: sections,
		Images:   images,
		Method:   "native",
	}, nil
}

// parseDocxRels reads word/_rels/document.xml.rels and returns a map of rId -> target path.
func parseDocxRels(fileIndex map[string]*zip.File) map[string]string {
	relsFile := fileIndex["word/_rels/document.xml.rels"]
	if relsFile == nil {
		return nil
	}

	rc, err := relsFile.Open()
	if err != nil {
		return nil
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil
	}

	var rels docxRelationships
	if err := xml.Unmarshal(data, &rels); err != nil {
		return nil
	}

	result := make(map[string]string, len(rels.Rels))
	for _, rel := range rels.Rels {
		result[rel.ID] = rel.Target
	}
	return result
}

// docxRelationships represents the .rels XML structure.
type docxRelationships struct {
	XMLName xml.Name          `xml:"Relationships"`
	Rels    []docxRelationship `xml:"Relationship"`
}

type docxRelationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
	Type   string `xml:"Type,attr"`
}

// extractDocxImages finds all embedded images in the document XML via drawing/blip elements.
// It tracks the current section index by watching for heading-style paragraphs so each
// image is attributed to the correct section (not just the last one).
func extractDocxImages(docXML []byte, rels map[string]string, fileIndex map[string]*zip.File, sectionCount int) []ExtractedImage {
	if rels == nil {
		return nil
	}

	decoder := xml.NewDecoder(bytes.NewReader(docXML))

	var images []ExtractedImage

	// Track section context to mirror parseDocxXML's heading-based splitting.
	//
	// In parseDocxXML, a heading paragraph flushes the accumulated content as
	// a section and starts a new one. So content between heading N and heading
	// N+1 belongs to section N (0-indexed). The first heading creates section 0
	// unless there was body content before it (which becomes section 0 instead,
	// pushing the first heading's content to section 1).
	sectionIdx := 0
	headingCount := 0
	contentSeen := false // body-text seen in current section group
	inPara := false
	inPPr := false
	paraIsHeading := false
	paraHasText := false

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				if t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" || t.Name.Space == "" {
					inPara = true
					paraIsHeading = false
					paraHasText = false
				}
			case "pPr":
				if inPara {
					inPPr = true
				}
			case "pStyle":
				if inPPr {
					for _, attr := range t.Attr {
						if attr.Name.Local == "val" {
							lower := strings.ToLower(attr.Value)
							if strings.HasPrefix(lower, "heading") || strings.HasPrefix(lower, "title") {
								paraIsHeading = true
							}
						}
					}
				}
			case "t":
				if inPara {
					paraHasText = true
				}
			case "blip":
				var embedID string
				for _, attr := range t.Attr {
					if attr.Name.Local == "embed" {
						embedID = attr.Value
						break
					}
				}
				if embedID == "" {
					continue
				}

				target, ok := rels[embedID]
				if !ok {
					continue
				}

				mediaPath := "word/" + target
				mediaPath = filepath.Clean(mediaPath)
				mediaPath = strings.ReplaceAll(mediaPath, "\\", "/")

				zf := fileIndex[mediaPath]
				if zf == nil {
					slog.Debug("docx: image file not found in ZIP", "path", mediaPath, "rId", embedID)
					continue
				}

				imgRC, err := zf.Open()
				if err != nil {
					slog.Debug("docx: failed to open image file", "path", mediaPath, "error", err)
					continue
				}

				imgData, err := io.ReadAll(imgRC)
				imgRC.Close()
				if err != nil {
					slog.Debug("docx: failed to read image file", "path", mediaPath, "error", err)
					continue
				}

				mimeType := mimeFromExt(filepath.Ext(zf.Name))
				if mimeType == "" {
					continue
				}

				w, h := imageSize(imgData)
				if w == 0 || h == 0 {
					continue
				}

				if w < 32 || h < 32 {
					continue
				}

				// Clamp to valid range
				idx := sectionIdx
				if idx >= sectionCount {
					idx = sectionCount - 1
				}
				if idx < 0 {
					idx = 0
				}

				images = append(images, ExtractedImage{
					Data:         imgData,
					MIMEType:     mimeType,
					PageNumber:   0,
					SectionIndex: idx,
					Width:        w,
					Height:       h,
				})
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if inPara {
					if paraIsHeading && paraHasText {
						headingCount++
						// The first heading with no prior body content starts
						// section 0 — don't increment. Subsequent headings (or
						// a first heading after body content) close the previous
						// section and start a new one.
						if headingCount > 1 || contentSeen {
							sectionIdx++
						}
						contentSeen = false
					} else if paraHasText {
						contentSeen = true
					}
					inPara = false
				}
			case "pPr":
				inPPr = false
			}
		}
	}

	return images
}

// mimeFromExt returns the MIME type for common image extensions.
func mimeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".tiff", ".tif":
		return "image/tiff"
	case ".emf":
		return "image/emf"
	case ".wmf":
		return "image/wmf"
	default:
		return ""
	}
}

// imageSize returns the width and height of an image from its encoded bytes.
func imageSize(data []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
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
