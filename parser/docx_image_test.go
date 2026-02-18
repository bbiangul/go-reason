package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// createTestPNG creates a minimal PNG image with the given dimensions.
func createTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("creating test PNG: %v", err)
	}
	return buf.Bytes()
}

// createTestDOCX builds a minimal .docx ZIP with a document containing an image.
func createTestDOCX(t *testing.T, imgData []byte, imgFilename string) string {
	t.Helper()
	dir := t.TempDir()
	docxPath := filepath.Join(dir, "test.docx")

	f, err := os.Create(docxPath)
	if err != nil {
		t.Fatalf("creating docx file: %v", err)
	}

	w := zip.NewWriter(f)

	// word/document.xml — a simple document with one heading, one paragraph, and a drawing
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
            xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
            xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
            xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>Test Section</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Some paragraph text.</w:t></w:r>
      <w:r>
        <w:drawing>
          <wp:inline>
            <a:graphic>
              <a:graphicData>
                <pic:pic>
                  <pic:blipFill>
                    <a:blip r:embed="rId1"/>
                  </pic:blipFill>
                </pic:pic>
              </a:graphicData>
            </a:graphic>
          </wp:inline>
        </w:drawing>
      </w:r>
    </w:p>
  </w:body>
</w:document>`
	addZipFile(t, w, "word/document.xml", []byte(docXML))

	// word/_rels/document.xml.rels
	type rel struct {
		XMLName xml.Name `xml:"Relationship"`
		ID      string   `xml:"Id,attr"`
		Type    string   `xml:"Type,attr"`
		Target  string   `xml:"Target,attr"`
	}
	type rels struct {
		XMLName xml.Name `xml:"Relationships"`
		Xmlns   string   `xml:"xmlns,attr"`
		Rels    []rel
	}
	relsData, _ := xml.Marshal(rels{
		Xmlns: "http://schemas.openxmlformats.org/package/2006/relationships",
		Rels: []rel{{
			ID:     "rId1",
			Type:   "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image",
			Target: "media/" + imgFilename,
		}},
	})
	addZipFile(t, w, "word/_rels/document.xml.rels", relsData)

	// word/media/<imgFilename>
	addZipFile(t, w, "word/media/"+imgFilename, imgData)

	if err := w.Close(); err != nil {
		t.Fatalf("closing zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing file: %v", err)
	}

	return docxPath
}

func addZipFile(t *testing.T, w *zip.Writer, name string, data []byte) {
	t.Helper()
	fw, err := w.Create(name)
	if err != nil {
		t.Fatalf("creating zip entry %s: %v", name, err)
	}
	if _, err := fw.Write(data); err != nil {
		t.Fatalf("writing zip entry %s: %v", name, err)
	}
}

func TestDOCXImageExtraction(t *testing.T) {
	imgData := createTestPNG(t, 200, 150)
	docxPath := createTestDOCX(t, imgData, "image1.png")

	p := &DOCXParser{}
	result, err := p.Parse(context.Background(), docxPath)
	if err != nil {
		t.Fatalf("parsing DOCX: %v", err)
	}

	if len(result.Sections) == 0 {
		t.Fatal("expected at least one section")
	}

	if len(result.Images) == 0 {
		t.Fatal("expected at least one extracted image")
	}

	img := result.Images[0]
	if img.MIMEType != "image/png" {
		t.Errorf("expected MIME image/png, got %s", img.MIMEType)
	}
	if img.Width != 200 || img.Height != 150 {
		t.Errorf("expected 200x150, got %dx%d", img.Width, img.Height)
	}
	if img.PageNumber != 0 {
		t.Errorf("DOCX images should have PageNumber 0, got %d", img.PageNumber)
	}
}

func TestDOCXImageExtractionSkipsTinyImages(t *testing.T) {
	// Create a 16x16 image (should be skipped)
	imgData := createTestPNG(t, 16, 16)
	docxPath := createTestDOCX(t, imgData, "tiny.png")

	p := &DOCXParser{}
	result, err := p.Parse(context.Background(), docxPath)
	if err != nil {
		t.Fatalf("parsing DOCX: %v", err)
	}

	if len(result.Images) != 0 {
		t.Errorf("expected no images (tiny image should be skipped), got %d", len(result.Images))
	}
}

func TestDOCXNoImages(t *testing.T) {
	// Create a DOCX with no images
	dir := t.TempDir()
	docxPath := filepath.Join(dir, "noimg.docx")

	f, err := os.Create(docxPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)

	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>Title</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Body text.</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`
	addZipFile(t, w, "word/document.xml", []byte(docXML))
	w.Close()
	f.Close()

	p := &DOCXParser{}
	result, err := p.Parse(context.Background(), docxPath)
	if err != nil {
		t.Fatalf("parsing DOCX: %v", err)
	}

	if len(result.Images) != 0 {
		t.Errorf("expected no images, got %d", len(result.Images))
	}
}

func TestDOCXImageSectionAssignment(t *testing.T) {
	// DOCX with 2 headings (creating 2 sections). Image is in the paragraph
	// under the first heading — should be assigned to section 0, not section 1.
	imgData := createTestPNG(t, 200, 150)

	dir := t.TempDir()
	docxPath := filepath.Join(dir, "multi_section.docx")

	f, err := os.Create(docxPath)
	if err != nil {
		t.Fatalf("creating docx file: %v", err)
	}

	w := zip.NewWriter(f)

	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
            xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
            xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
            xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>First Section</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Text under first heading.</w:t></w:r>
      <w:r>
        <w:drawing>
          <wp:inline>
            <a:graphic>
              <a:graphicData>
                <pic:pic>
                  <pic:blipFill>
                    <a:blip r:embed="rId1"/>
                  </pic:blipFill>
                </pic:pic>
              </a:graphicData>
            </a:graphic>
          </wp:inline>
        </w:drawing>
      </w:r>
    </w:p>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>Second Section</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>Text under second heading.</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`
	addZipFile(t, w, "word/document.xml", []byte(docXML))

	type rel struct {
		XMLName xml.Name `xml:"Relationship"`
		ID      string   `xml:"Id,attr"`
		Type    string   `xml:"Type,attr"`
		Target  string   `xml:"Target,attr"`
	}
	type rels struct {
		XMLName xml.Name `xml:"Relationships"`
		Xmlns   string   `xml:"xmlns,attr"`
		Rels    []rel
	}
	relsData, _ := xml.Marshal(rels{
		Xmlns: "http://schemas.openxmlformats.org/package/2006/relationships",
		Rels: []rel{{
			ID:     "rId1",
			Type:   "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image",
			Target: "media/image1.png",
		}},
	})
	addZipFile(t, w, "word/_rels/document.xml.rels", relsData)
	addZipFile(t, w, "word/media/image1.png", imgData)

	w.Close()
	f.Close()

	p := &DOCXParser{}
	result, err := p.Parse(context.Background(), docxPath)
	if err != nil {
		t.Fatalf("parsing DOCX: %v", err)
	}

	if len(result.Sections) < 2 {
		t.Fatalf("expected at least 2 sections, got %d", len(result.Sections))
	}

	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(result.Images))
	}

	// Image appears after "First Section" heading but before "Second Section"
	// heading, so it should be in section 0 (the first section).
	img := result.Images[0]
	if img.SectionIndex != 0 {
		t.Errorf("expected image in section 0 (first section), got section %d", img.SectionIndex)
	}
}

func TestMimeFromExt(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".png", "image/png"},
		{".PNG", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".bmp", "image/bmp"},
		{".svg", ""},
		{".txt", ""},
	}

	for _, tt := range tests {
		got := mimeFromExt(tt.ext)
		if got != tt.want {
			t.Errorf("mimeFromExt(%q) = %q, want %q", tt.ext, got, tt.want)
		}
	}
}
