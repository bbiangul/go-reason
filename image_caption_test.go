package goreason

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/bbiangul/go-reason/llm"
	"github.com/bbiangul/go-reason/parser"
)

// mockVisionProvider implements both llm.Provider and llm.VisionProvider for testing.
type mockVisionProvider struct {
	captionResponse string
	captionErr      error
	callCount       int
}

func (m *mockVisionProvider) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "mock"}, nil
}

func (m *mockVisionProvider) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, nil
}

func (m *mockVisionProvider) ChatWithImages(_ context.Context, _ llm.VisionChatRequest) (*llm.ChatResponse, error) {
	m.callCount++
	if m.captionErr != nil {
		return nil, m.captionErr
	}
	return &llm.ChatResponse{Content: m.captionResponse}, nil
}

func TestCaptionImages_CaptioningEnabled(t *testing.T) {
	mock := &mockVisionProvider{captionResponse: "A wiring diagram showing power connections"}

	e := &engine{
		cfg:       Config{CaptionImages: true},
		visionLLM: mock,
	}

	sections := []parser.Section{
		{Heading: "Section 1", Content: "Some text about wiring."},
		{Heading: "Section 2", Content: "More text."},
	}

	images := []parser.ExtractedImage{
		{Data: []byte("fake-img"), MIMEType: "image/png", PageNumber: 1, SectionIndex: 0, Width: 800, Height: 600},
	}

	result, _ := e.captionImages(context.Background(), sections, images)

	if mock.callCount != 1 {
		t.Errorf("expected 1 vision call, got %d", mock.callCount)
	}

	if !strings.Contains(result[0].Content, "[Image: A wiring diagram showing power connections]") {
		t.Errorf("expected caption in section content, got: %s", result[0].Content)
	}
}

func TestCaptionImages_CaptioningDisabled(t *testing.T) {
	mock := &mockVisionProvider{captionResponse: "should not be called"}

	e := &engine{
		cfg:       Config{CaptionImages: false}, // disabled
		visionLLM: mock,
	}

	sections := []parser.Section{
		{Heading: "Section 1", Content: "Text."},
	}

	images := []parser.ExtractedImage{
		{Data: []byte("fake"), MIMEType: "image/png", PageNumber: 1, SectionIndex: 0, Width: 100, Height: 100},
	}

	result, _ := e.captionImages(context.Background(), sections, images)

	if mock.callCount != 0 {
		t.Errorf("expected 0 vision calls when captioning disabled, got %d", mock.callCount)
	}

	if !strings.Contains(result[0].Content, "[image]") {
		t.Errorf("expected [image] marker when captioning disabled, got: %s", result[0].Content)
	}
}

func TestCaptionImages_LargestImagePerPage(t *testing.T) {
	mock := &mockVisionProvider{captionResponse: "The large chart"}

	e := &engine{
		cfg:       Config{CaptionImages: true},
		visionLLM: mock,
	}

	sections := []parser.Section{
		{Heading: "Page 1", Content: "Content."},
	}

	// Two images on the same page — only the larger should be captioned
	images := []parser.ExtractedImage{
		{Data: []byte("small"), MIMEType: "image/png", PageNumber: 1, SectionIndex: 0, Width: 100, Height: 100},
		{Data: []byte("large"), MIMEType: "image/jpeg", PageNumber: 1, SectionIndex: 0, Width: 800, Height: 600},
	}

	result, _ := e.captionImages(context.Background(), sections, images)

	// Should only make 1 call (one per page, not per image)
	if mock.callCount != 1 {
		t.Errorf("expected 1 vision call (one per page), got %d", mock.callCount)
	}

	content := result[0].Content
	if !strings.Contains(content, "[Image: The large chart]") {
		t.Errorf("expected captioned largest image, got: %s", content)
	}
	// The other image should get [image]
	if strings.Count(content, "[image]") != 1 {
		t.Errorf("expected 1 [image] marker for the non-captioned image, got content: %s", content)
	}
}

func TestCaptionImages_FailureFallback(t *testing.T) {
	mock := &mockVisionProvider{captionErr: fmt.Errorf("API error")}

	e := &engine{
		cfg:       Config{CaptionImages: true},
		visionLLM: mock,
	}

	sections := []parser.Section{
		{Heading: "Section 1", Content: "Text."},
	}

	images := []parser.ExtractedImage{
		{Data: []byte("fake"), MIMEType: "image/png", PageNumber: 1, SectionIndex: 0, Width: 200, Height: 200},
	}

	result, _ := e.captionImages(context.Background(), sections, images)

	// Should fall back to [image] on error
	if !strings.Contains(result[0].Content, "[image]") {
		t.Errorf("expected [image] fallback on error, got: %s", result[0].Content)
	}
	if strings.Contains(result[0].Content, "[Image:") {
		t.Errorf("should not contain captioned image on error, got: %s", result[0].Content)
	}
}

func TestCaptionImages_NilVisionLLM(t *testing.T) {
	e := &engine{
		cfg:       Config{CaptionImages: true},
		visionLLM: nil, // no vision LLM configured
	}

	sections := []parser.Section{
		{Heading: "Section 1", Content: "Text."},
	}

	images := []parser.ExtractedImage{
		{Data: []byte("fake"), MIMEType: "image/png", PageNumber: 1, SectionIndex: 0, Width: 200, Height: 200},
	}

	result, _ := e.captionImages(context.Background(), sections, images)

	if !strings.Contains(result[0].Content, "[image]") {
		t.Errorf("expected [image] when no vision LLM, got: %s", result[0].Content)
	}
}

func TestCaptionImages_MultiplePages(t *testing.T) {
	mock := &mockVisionProvider{captionResponse: "Chart description"}

	e := &engine{
		cfg:       Config{CaptionImages: true},
		visionLLM: mock,
	}

	sections := []parser.Section{
		{Heading: "Page 1", Content: "Text1.", PageNumber: 1},
		{Heading: "Page 2", Content: "Text2.", PageNumber: 2},
	}

	// One image per page
	images := []parser.ExtractedImage{
		{Data: []byte("img1"), MIMEType: "image/png", PageNumber: 1, SectionIndex: 0, Width: 400, Height: 300},
		{Data: []byte("img2"), MIMEType: "image/jpeg", PageNumber: 2, SectionIndex: 1, Width: 500, Height: 400},
	}

	result, _ := e.captionImages(context.Background(), sections, images)

	// Should make 2 calls (one per page)
	if mock.callCount != 2 {
		t.Errorf("expected 2 vision calls (one per page), got %d", mock.callCount)
	}

	if !strings.Contains(result[0].Content, "[Image: Chart description]") {
		t.Errorf("page 1 missing caption, got: %s", result[0].Content)
	}
	if !strings.Contains(result[1].Content, "[Image: Chart description]") {
		t.Errorf("page 2 missing caption, got: %s", result[1].Content)
	}
}

func TestCaptionImages_EmptyImages(t *testing.T) {
	e := &engine{
		cfg: Config{CaptionImages: true},
	}

	sections := []parser.Section{
		{Heading: "Section 1", Content: "Text."},
	}

	result, collected := e.captionImages(context.Background(), sections, nil)

	// No images — sections should be unchanged
	if collected != nil {
		t.Errorf("expected nil collected images, got %d", len(collected))
	}
	if result[0].Content != "Text." {
		t.Errorf("sections should be unchanged with no images, got: %s", result[0].Content)
	}
}
