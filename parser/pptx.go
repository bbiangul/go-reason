package parser

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
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

		sections = append(sections, Section{
			Heading:    fmt.Sprintf("Slide %d", num),
			Content:    text,
			Type:       "section",
			Level:      1,
			PageNumber: num,
		})
	}

	if len(sections) == 0 {
		return nil, fmt.Errorf("no text found in PPTX")
	}

	return &ParseResult{
		Sections: sections,
		Method:   "native",
	}, nil
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
