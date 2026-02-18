package parser

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

type PDFParser struct{}

func (p *PDFParser) SupportedFormats() []string { return []string{"pdf"} }

func (p *PDFParser) Parse(ctx context.Context, path string) (*ParseResult, error) {
	f, reader, err := pdf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening PDF: %w", err)
	}
	defer f.Close()

	totalPages := reader.NumPage()
	sections := make([]Section, 0)
	var allImages []ExtractedImage

	for i := 1; i <= totalPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := extractPageTextOrdered(page)
		if err != nil {
			// Skip pages that fail to extract
			continue
		}

		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		// Split page text into sections by detecting heading patterns
		sectionStartIdx := len(sections)
		pageSections := splitPageIntoSections(text, i)
		sections = append(sections, pageSections...)

		// Extract images from this page
		pageImages := extractPageImages(page, i, sectionStartIdx)
		allImages = append(allImages, pageImages...)
	}

	// Post-process: detect running headers and carry over real headings
	// across page boundaries.
	sections = fixRunningHeaders(sections, totalPages)

	if len(sections) == 0 {
		return &ParseResult{
			Method: "native",
			Sections: []Section{{
				Content:    "Unable to extract text from PDF",
				Type:       "paragraph",
				PageNumber: 1,
			}},
		}, nil
	}

	return &ParseResult{
		Sections: sections,
		Images:   allImages,
		Method:   "native",
	}, nil
}

// extractPageImages extracts images from a PDF page's XObject resources.
func extractPageImages(page pdf.Page, pageNum int, sectionStartIdx int) []ExtractedImage {
	resources := page.Resources()
	if resources.IsNull() {
		return nil
	}

	xobjects := resources.Key("XObject")
	if xobjects.IsNull() {
		return nil
	}

	var images []ExtractedImage
	for _, name := range xobjects.Keys() {
		xobj := xobjects.Key(name)
		if xobj.Key("Subtype").Name() != "Image" {
			continue
		}

		// Skip image masks (1-bit stencil masks used for transparency)
		if xobj.Key("ImageMask").Bool() {
			continue
		}

		width := int(xobj.Key("Width").Int64())
		height := int(xobj.Key("Height").Int64())
		if width == 0 || height == 0 {
			continue
		}

		// Skip tiny images (icons, bullets, decorative elements)
		if width < 32 || height < 32 {
			continue
		}

		filter := xobj.Key("Filter").Name()

		imgData, mimeType := extractSingleImage(xobj, filter, width, height, pageNum, name)
		if imgData == nil {
			continue
		}

		images = append(images, ExtractedImage{
			Data:         imgData,
			MIMEType:     mimeType,
			PageNumber:   pageNum,
			SectionIndex: sectionStartIdx, // associate with first section on this page
			Width:        width,
			Height:       height,
		})
	}

	return images
}

// extractSingleImage reads image data from a PDF XObject, handling panics from
// the ledongthuc/pdf library which can panic on unsupported filter combinations.
func extractSingleImage(xobj pdf.Value, filter string, width, height, pageNum int, name string) (data []byte, mimeType string) {
	// Recover from panics in the pdf library's Reader() method, which can panic
	// on certain filter types (e.g. DCTDecode in some PDF versions).
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("pdf: panic reading image stream, skipping", "page", pageNum, "name", name, "panic", r)
			data = nil
			mimeType = ""
		}
	}()

	switch filter {
	case "DCTDecode":
		// JPEG — the raw stream bytes ARE the JPEG data. The ledongthuc/pdf
		// library's Reader() panics on DCTDecode because it tries to apply
		// filters it doesn't support. We bypass the filter chain by reading
		// raw bytes directly from the underlying file via reflection.
		raw, err := readRawStreamBytes(xobj)
		if err != nil {
			slog.Debug("pdf: failed to read raw JPEG stream", "page", pageNum, "name", name, "error", err)
			return nil, ""
		}
		if len(raw) > 2 && raw[0] == 0xff && raw[1] == 0xd8 {
			return raw, "image/jpeg"
		}
		slog.Debug("pdf: DCTDecode image missing JPEG magic", "page", pageNum, "name", name)
		return nil, ""

	case "FlateDecode", "":
		// Raw pixel data (decompressed by Reader) — re-encode as PNG
		rc := xobj.Reader()
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			slog.Debug("pdf: failed to read FlateDecode image", "page", pageNum, "name", name, "error", err)
			return nil, ""
		}

		pngData, err := rawPixelsToPNG(raw, width, height, xobj.Key("ColorSpace").Name(), int(xobj.Key("BitsPerComponent").Int64()))
		if err != nil {
			slog.Debug("pdf: failed to encode PNG", "page", pageNum, "name", name, "error", err)
			return nil, ""
		}
		return pngData, "image/png"

	default:
		// JPXDecode, CCITTFaxDecode, etc. — skip with debug log
		slog.Debug("pdf: unsupported image filter", "page", pageNum, "name", name, "filter", filter)
		return nil, ""
	}
}

// readRawStreamBytes reads the raw (unfiltered) stream bytes from a pdf.Value
// by accessing the library's internal fields via reflection. This is necessary
// because Reader() tries to apply filters like DCTDecode and panics, but for
// JPEG images the raw stream bytes are already valid JPEG data.
//
// Internal layout used (ledongthuc/pdf):
//
//	Value  { r *Reader; ptr objptr; data interface{} }
//	Reader { f io.ReaderAt; ... }
//	stream { hdr dict; ptr objptr; offset int64 }
func readRawStreamBytes(v pdf.Value) ([]byte, error) {
	length := v.Key("Length").Int64()
	if length <= 0 {
		return nil, fmt.Errorf("stream has no length")
	}

	// Access Value's unexported fields via reflect + unsafe.
	val := reflect.ValueOf(v)

	// v.data (field index 2) → stream struct
	dataField := val.Field(2) // data interface{}
	if dataField.IsNil() {
		return nil, fmt.Errorf("value has nil data")
	}
	streamVal := dataField.Elem() // concrete value inside interface
	if streamVal.Kind() == reflect.Ptr {
		streamVal = streamVal.Elem()
	}

	// stream.offset (field index 2)
	offsetField := streamVal.Field(2) // offset int64
	offset := offsetField.Int()

	// v.r (field index 0) → *Reader
	rField := val.Field(0) // r *Reader
	if rField.IsNil() {
		return nil, fmt.Errorf("value has nil reader")
	}

	// Reader.f (field index 0) → io.ReaderAt
	// Use UnsafePointer() to avoid the uintptr→unsafe.Pointer conversion
	// that go vet flags as a possible misuse.
	readerStruct := reflect.NewAt(rField.Type().Elem(), rField.UnsafePointer()).Elem()
	fField := readerStruct.Field(0) // f io.ReaderAt
	readerAt, ok := fField.Interface().(io.ReaderAt)
	if !ok {
		return nil, fmt.Errorf("reader.f is not io.ReaderAt")
	}

	// Read raw bytes from file
	buf := make([]byte, length)
	n, err := readerAt.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("reading stream at offset %d: %w", offset, err)
	}
	return buf[:n], nil
}

// rawPixelsToPNG converts raw pixel data to PNG format.
func rawPixelsToPNG(data []byte, width, height int, colorSpace string, bitsPerComponent int) ([]byte, error) {
	if bitsPerComponent == 0 {
		bitsPerComponent = 8
	}

	var img image.Image
	switch colorSpace {
	case "DeviceRGB", "":
		// 3 bytes per pixel (RGB)
		expected := width * height * 3
		if len(data) < expected {
			return nil, fmt.Errorf("insufficient data for RGB image: got %d, expected %d", len(data), expected)
		}
		rgba := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				offset := (y*width + x) * 3
				rgba.SetRGBA(x, y, color.RGBA{
					R: data[offset],
					G: data[offset+1],
					B: data[offset+2],
					A: 255,
				})
			}
		}
		img = rgba

	case "DeviceGray":
		// 1 byte per pixel (grayscale)
		expected := width * height
		if len(data) < expected {
			return nil, fmt.Errorf("insufficient data for gray image: got %d, expected %d", len(data), expected)
		}
		gray := image.NewGray(image.Rect(0, 0, width, height))
		copy(gray.Pix, data[:expected])
		img = gray

	case "DeviceCMYK":
		// 4 bytes per pixel — convert to RGB
		expected := width * height * 4
		if len(data) < expected {
			return nil, fmt.Errorf("insufficient data for CMYK image: got %d, expected %d", len(data), expected)
		}
		rgba := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				offset := (y*width + x) * 4
				c, m, yk, k := data[offset], data[offset+1], data[offset+2], data[offset+3]
				r := 255 - min(255, int(c)+int(k))
				g := 255 - min(255, int(m)+int(k))
				b := 255 - min(255, int(yk)+int(k))
				rgba.SetRGBA(x, y, color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255})
			}
		}
		img = rgba

	default:
		return nil, fmt.Errorf("unsupported color space: %s", colorSpace)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
}


// extractPageTextOrdered extracts text from a PDF page sorted by visual
// position (top-to-bottom, left-to-right). The default GetPlainText reads
// text in PDF object order which can differ from visual layout — headings
// may appear after the body text they label.
//
// This function groups Content() elements into visual lines by Y proximity
// (preserving the content-stream order within each line — which GetPlainText
// relies on for correct character sequencing), then sorts the lines by Y so
// the result follows top-to-bottom reading order.
func extractPageTextOrdered(page pdf.Page) (string, error) {
	content := page.Content()
	if len(content.Text) == 0 {
		return page.GetPlainText(nil)
	}

	// Group consecutive text elements into visual lines by Y proximity.
	// We preserve the content-stream order within each line — sorting by X
	// would garble text because some PDFs use negative text matrices.
	const lineTolerance = 3.0

	type visualLine struct {
		y   float64 // representative Y (from first element)
		buf strings.Builder
	}

	var lines []*visualLine
	var cur *visualLine

	for _, t := range content.Text {
		if cur == nil || math.Abs(t.Y-cur.y) > lineTolerance {
			lines = append(lines, &visualLine{y: t.Y})
			cur = lines[len(lines)-1]
		}
		cur.buf.WriteString(t.S)
	}

	// Sort lines by Y descending — higher Y = higher on the page in PDF
	// coordinates (origin at bottom-left).
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].y > lines[j].y
	})

	// Build the result.
	var parts []string
	for _, l := range lines {
		text := strings.TrimSpace(l.buf.String())
		if text != "" {
			parts = append(parts, text)
		}
	}

	result := strings.Join(parts, "\n")
	if strings.TrimSpace(result) == "" {
		return page.GetPlainText(nil)
	}

	return result, nil
}

// splitPageIntoSections breaks page text into logical sections.
func splitPageIntoSections(text string, pageNum int) []Section {
	lines := strings.Split(text, "\n")
	var sections []Section
	var currentContent strings.Builder
	var currentHeading string
	currentLevel := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			continue
		}

		// Detect headings: all-caps lines, numbered sections, short bold-like lines
		if isLikelyHeading(trimmed) {
			// Save previous section
			if currentContent.Len() > 0 || currentHeading != "" {
				sections = append(sections, Section{
					Heading:    currentHeading,
					Content:    strings.TrimSpace(currentContent.String()),
					Level:      currentLevel,
					PageNumber: pageNum,
					Type:       classifySectionType(currentHeading, currentContent.String()),
				})
				currentContent.Reset()
			}
			currentHeading = trimmed
			currentLevel = detectHeadingLevel(trimmed)
		} else {
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			currentContent.WriteString(trimmed)
		}
	}

	// Final section — save even if content is empty so trailing headings
	// are not silently dropped (they provide context for the next page's content).
	if currentContent.Len() > 0 || currentHeading != "" {
		sections = append(sections, Section{
			Heading:    currentHeading,
			Content:    strings.TrimSpace(currentContent.String()),
			Level:      currentLevel,
			PageNumber: pageNum,
			Type:       classifySectionType(currentHeading, currentContent.String()),
		})
	}

	// Merge empty-content sections into the next section. When a parent
	// heading (e.g. "3.9.1 Modelo A") has no body because the next line is
	// a sub-heading (e.g. "3.9.1.1 Material de Fabricación:"), prepend the
	// parent heading so the model name stays co-located with spec data.
	for i := len(sections) - 2; i >= 0; i-- {
		if sections[i].Content == "" && sections[i].Heading != "" &&
			i+1 < len(sections) && sections[i+1].Level > sections[i].Level {
			if sections[i+1].Heading != "" {
				sections[i+1].Heading = sections[i].Heading + " — " + sections[i+1].Heading
			} else {
				sections[i+1].Heading = sections[i].Heading
			}
			sections[i+1].Level = sections[i].Level
			sections = append(sections[:i], sections[i+1:]...)
		}
	}

	// If no sections were created, return the whole page as one section
	if len(sections) == 0 && strings.TrimSpace(text) != "" {
		sections = append(sections, Section{
			Content:    text,
			PageNumber: pageNum,
			Type:       "paragraph",
		})
	}

	return sections
}

func isLikelyHeading(line string) bool {
	// All caps and short
	if len(line) < 100 && line == strings.ToUpper(line) && len(line) > 2 {
		return true
	}
	// Numbered section like "1.", "1.1", "1.1.1", "3.9.1", "7.3.1.2"
	if len(line) < 120 {
		if len(line) > 0 && line[0] >= '0' && line[0] <= '9' && strings.Contains(line[:min(10, len(line))], ".") {
			return true
		}
		lower := strings.ToLower(line)
		// English heading prefixes
		if strings.HasPrefix(lower, "section ") || strings.HasPrefix(lower, "article ") ||
			strings.HasPrefix(lower, "chapter ") || strings.HasPrefix(lower, "part ") {
			return true
		}
		// Spanish heading prefixes
		if strings.HasPrefix(lower, "sección ") || strings.HasPrefix(lower, "seccion ") ||
			strings.HasPrefix(lower, "capítulo ") || strings.HasPrefix(lower, "capitulo ") ||
			strings.HasPrefix(lower, "anexo ") {
			return true
		}
		// Portuguese heading prefixes
		if strings.HasPrefix(lower, "seção ") || strings.HasPrefix(lower, "secao ") ||
			strings.HasPrefix(lower, "capítulo ") || // same as Spanish
			strings.HasPrefix(lower, "artigo ") ||
			strings.HasPrefix(lower, "anexo ") { // same as Spanish
			return true
		}
		// French heading prefixes
		if strings.HasPrefix(lower, "chapitre ") || strings.HasPrefix(lower, "partie ") ||
			strings.HasPrefix(lower, "annexe ") || strings.HasPrefix(lower, "article ") { // "article" also English
			return true
		}
		// "Tabla/Tabela/Tableau N..." / "Figura/Figure N..." — only when
		// followed by a digit to avoid matching mid-paragraph text.
		for _, prefix := range []string{
			"tabla ", "tabela ", "tableau ",        // es, pt, fr
			"figura ", "figure ",                   // es/pt, en/fr
			"cuadro ", "quadro ", "gráfico ", "graphique ", // es, pt, es, fr
		} {
			if strings.HasPrefix(lower, prefix) {
				afterPrefix := len(prefix)
				if len(lower) > afterPrefix && lower[afterPrefix] >= '0' && lower[afterPrefix] <= '9' {
					return true
				}
			}
		}
	}
	return false
}

func detectHeadingLevel(heading string) int {
	// Count dots in numbering to determine depth
	parts := strings.SplitN(heading, " ", 2)
	if len(parts) > 0 {
		dots := strings.Count(parts[0], ".")
		if dots > 0 {
			return dots
		}
	}
	// All-caps = top level
	if heading == strings.ToUpper(heading) {
		return 1
	}
	return 2
}

func classifySectionType(heading, content string) string {
	headingLower := strings.ToLower(heading)
	contentLower := strings.ToLower(content)

	// Definition: check heading and content for definition-related keywords
	if strings.Contains(headingLower, "definition") || strings.Contains(headingLower, "definición") ||
		strings.Contains(headingLower, "glosario") || strings.Contains(headingLower, "glossary") ||
		strings.Contains(contentLower, "definition") || strings.Contains(contentLower, "definición") {
		return "definition"
	}
	// Requirement: check heading and content for requirement-related keywords
	if strings.Contains(headingLower, "shall") || strings.Contains(headingLower, "must") || strings.Contains(headingLower, "requirement") ||
		strings.Contains(headingLower, "requisito") || strings.Contains(headingLower, "especificación") ||
		strings.Contains(contentLower, "shall") || strings.Contains(contentLower, "must") || strings.Contains(contentLower, "requirement") ||
		strings.Contains(contentLower, "requisito") || strings.Contains(contentLower, "especificación") {
		return "requirement"
	}
	// Table: check heading for table keywords
	if strings.Contains(headingLower, "table") || strings.Contains(headingLower, "tabla") {
		return "table"
	}
	// Structural table detection via content: tabs/pipes indicate actual table formatting
	if strings.Count(content, "\t") > 3 || strings.Count(content, "|") > 3 {
		return "table"
	}
	if strings.Contains(headingLower, "anexo") || strings.Contains(headingLower, "annex") {
		return "annex"
	}
	return "section"
}

// fixRunningHeaders detects repeated headers/footers (e.g. document titles that
// appear on every page) and replaces them with the last meaningful heading.
// This fixes the page-boundary problem where a section starting on page N
// continues onto page N+1 but the content on N+1 gets assigned to the generic
// running header instead of the real section heading.
func fixRunningHeaders(sections []Section, totalPages int) []Section {
	if len(sections) == 0 || totalPages == 0 {
		return sections
	}

	// Step 1: Count on how many distinct pages each heading text appears.
	headingPages := make(map[string]map[int]bool) // heading → set of page numbers
	for _, s := range sections {
		h := normalizeHeading(s.Heading)
		if h == "" {
			continue
		}
		if headingPages[h] == nil {
			headingPages[h] = make(map[int]bool)
		}
		headingPages[h][s.PageNumber] = true
	}

	// Step 2: A heading appearing on >25% of pages is a running header.
	// Require at least 3 distinct pages to avoid false positives on short docs.
	threshold := max(3, totalPages/4)
	runningHeaders := make(map[string]bool)
	for h, pages := range headingPages {
		if len(pages) >= threshold {
			runningHeaders[h] = true
		}
	}

	if len(runningHeaders) == 0 {
		return sections
	}

	// Step 3: Replace running headers with the last real heading.
	var lastRealHeading string
	var lastRealLevel int
	for i := range sections {
		h := normalizeHeading(sections[i].Heading)
		if runningHeaders[h] {
			// This is a running header — replace with carried-over heading.
			if lastRealHeading != "" {
				sections[i].Heading = lastRealHeading
				sections[i].Level = lastRealLevel
			}
		} else if sections[i].Heading != "" {
			lastRealHeading = sections[i].Heading
			lastRealLevel = sections[i].Level
		}
	}

	return sections
}

// normalizeHeading strips trailing page-number artifacts and whitespace
// so that "MANUAL TÉCNICO AV-FM, AV-FF\uf0d2" matches across pages.
func normalizeHeading(h string) string {
	h = strings.TrimSpace(h)
	// Strip trailing non-printable/replacement chars often left by PDF extraction.
	for len(h) > 0 {
		r := rune(h[len(h)-1])
		if r > 127 || r == '\uf0d2' || r == '\ufffd' {
			h = h[:len(h)-1]
			h = strings.TrimSpace(h)
		} else {
			break
		}
	}
	return h
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
