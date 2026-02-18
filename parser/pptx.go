package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
)

type PPTXParser struct{}

func (p *PPTXParser) SupportedFormats() []string { return []string{"pptx"} }

func (p *PPTXParser) Parse(ctx context.Context, path string) (*ParseResult, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("opening PPTX: %w", err)
	}
	defer r.Close()

	// Build file index for quick lookup
	fileIndex := make(map[string]*zip.File, len(r.File))
	for _, f := range r.File {
		fileIndex[f.Name] = f
	}

	// Collect slide files (ppt/slides/slide1.xml, slide2.xml, ...)
	slideFiles := make(map[int]*zip.File)
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			num := extractSlideNumber(f.Name)
			if num > 0 {
				slideFiles[num] = f
			}
		}
	}

	// Sort by slide number
	nums := make([]int, 0, len(slideFiles))
	for n := range slideFiles {
		nums = append(nums, n)
	}
	sort.Ints(nums)

	var sections []Section
	var allImages []ExtractedImage
	for _, num := range nums {
		f := slideFiles[num]
		rc, err := f.Open()
		if err != nil {
			continue
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		text := extractPPTXSlideText(data)
		if text == "" {
			continue
		}

		sectionIdx := len(sections)
		sections = append(sections, Section{
			Heading:    fmt.Sprintf("Slide %d", num),
			Content:    text,
			Type:       "section",
			Level:      1,
			PageNumber: num,
		})

		// Extract images from this slide
		slideImages := extractPPTXSlideImages(data, num, sectionIdx, fileIndex)
		allImages = append(allImages, slideImages...)
	}

	if len(sections) == 0 {
		return nil, fmt.Errorf("no text found in PPTX")
	}

	return &ParseResult{
		Sections: sections,
		Images:   allImages,
		Method:   "native",
	}, nil
}

// extractPPTXSlideImages extracts images from a single slide's XML.
func extractPPTXSlideImages(slideXML []byte, slideNum int, sectionIdx int, fileIndex map[string]*zip.File) []ExtractedImage {
	// Parse the slide's .rels file for relationship mappings
	relsPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", slideNum)
	rels := parsePPTXRels(fileIndex, relsPath)
	if rels == nil {
		return nil
	}

	// Find all blip elements (same approach as DOCX — a:blip r:embed="rIdN")
	decoder := xml.NewDecoder(bytes.NewReader(slideXML))

	var images []ExtractedImage
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local != "blip" {
			continue
		}

		var embedID string
		for _, attr := range se.Attr {
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

		// Resolve path — targets are relative to ppt/slides/
		mediaPath := "ppt/slides/" + target
		mediaPath = filepath.Clean(mediaPath)
		mediaPath = strings.ReplaceAll(mediaPath, "\\", "/")

		zf := fileIndex[mediaPath]
		if zf == nil {
			slog.Debug("pptx: image file not found in ZIP", "path", mediaPath, "rId", embedID)
			continue
		}

		imgRC, err := zf.Open()
		if err != nil {
			slog.Debug("pptx: failed to open image file", "path", mediaPath, "error", err)
			continue
		}

		imgData, err := io.ReadAll(imgRC)
		imgRC.Close()
		if err != nil {
			slog.Debug("pptx: failed to read image file", "path", mediaPath, "error", err)
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

		// Skip tiny images
		if w < 32 || h < 32 {
			continue
		}

		images = append(images, ExtractedImage{
			Data:         imgData,
			MIMEType:     mimeType,
			PageNumber:   slideNum,
			SectionIndex: sectionIdx,
			Width:        w,
			Height:       h,
		})
	}

	return images
}

// parsePPTXRels reads a PPTX .rels file and returns rId -> target map.
func parsePPTXRels(fileIndex map[string]*zip.File, relsPath string) map[string]string {
	relsFile := fileIndex[relsPath]
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

	// Reuse the same Relationships struct from DOCX (same OOXML format)
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

// pptxSlide simplified XML structure
type pptxSlide struct {
	CSld struct {
		SpTree struct {
			SPs []pptxSP `xml:"sp"`
		} `xml:"spTree"`
	} `xml:"cSld"`
}

type pptxSP struct {
	TxBody *pptxTxBody `xml:"txBody"`
}

type pptxTxBody struct {
	Paras []pptxAPara `xml:"p"`
}

type pptxAPara struct {
	Runs []pptxARun `xml:"r"`
}

type pptxARun struct {
	Text string `xml:"t"`
}

func extractPPTXSlideText(data []byte) string {
	var slide pptxSlide
	if err := xml.Unmarshal(data, &slide); err != nil {
		return ""
	}

	var parts []string
	for _, sp := range slide.CSld.SpTree.SPs {
		if sp.TxBody == nil {
			continue
		}
		for _, para := range sp.TxBody.Paras {
			var line strings.Builder
			for _, run := range para.Runs {
				line.WriteString(run.Text)
			}
			if t := strings.TrimSpace(line.String()); t != "" {
				parts = append(parts, t)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func extractSlideNumber(name string) int {
	// Extract number from "ppt/slides/slide1.xml"
	name = strings.TrimPrefix(name, "ppt/slides/slide")
	name = strings.TrimSuffix(name, ".xml")
	var num int
	fmt.Sscanf(name, "%d", &num)
	return num
}
