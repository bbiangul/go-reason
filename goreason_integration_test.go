//go:build integration && cgo

package goreason

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	ollamaURL   = "http://localhost:11434"
	chatModel   = "qwen3:8b"
	embedModel  = "qwen3-embedding"
	embedDim    = 4096
	testTimeout = 10 * time.Minute
)

// shared holds the engine and ingested document ID set up once for all tests.
var shared struct {
	once    sync.Once
	eng     Engine
	docID   int64
	docPath string
	dbDir   string
	err     error
}

func ollamaAvailable() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(ollamaURL + "/api/tags")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

// warmModel sends a tiny request to force Ollama to load a model into memory.
func warmModel(model string) error {
	body := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hi"}],"stream":false,"options":{"num_predict":1}}`, model)
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Post(ollamaURL+"/api/chat", "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// warmEmbedModel sends a tiny embedding request.
func warmEmbedModel(model string) error {
	body := fmt.Sprintf(`{"model":%q,"input":["test"]}`, model)
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Post(ollamaURL+"/api/embed", "application/json", strings.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// setupShared creates the shared engine and ingests the test document once.
func setupShared(t *testing.T) {
	t.Helper()
	shared.once.Do(func() {
		if !ollamaAvailable() {
			shared.err = fmt.Errorf("ollama not available")
			return
		}

		// Warm up both models sequentially (avoid concurrent loading).
		t.Log("Warming up embedding model...")
		if err := warmEmbedModel(embedModel); err != nil {
			shared.err = fmt.Errorf("warming embed model: %w", err)
			return
		}
		t.Log("Warming up chat model...")
		if err := warmModel(chatModel); err != nil {
			shared.err = fmt.Errorf("warming chat model: %w", err)
			return
		}

		// Create temp directory for DB.
		dir, err := os.MkdirTemp("", "goreason-integration-*")
		if err != nil {
			shared.err = err
			return
		}
		shared.dbDir = dir

		cfg := Config{
			DBPath: filepath.Join(dir, "integration_test.db"),
			Chat: LLMConfig{
				Provider: "ollama",
				Model:    chatModel,
				BaseURL:  ollamaURL,
			},
			Embedding: LLMConfig{
				Provider: "ollama",
				Model:    embedModel,
				BaseURL:  ollamaURL,
			},
			WeightVector:        1.0,
			WeightFTS:           1.0,
			WeightGraph:         0.5,
			MaxChunkTokens:      512,
			ChunkOverlap:        64,
			MaxRounds:           2,
			ConfidenceThreshold: 0.3,
			EmbeddingDim:        embedDim,
		}

		eng, err := New(cfg)
		if err != nil {
			shared.err = fmt.Errorf("creating engine: %w", err)
			return
		}
		shared.eng = eng

		// Create and ingest the test document.
		docPath := createTestDOCX(dir)
		shared.docPath = docPath

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		t.Log("Ingesting test document...")
		docID, err := eng.Ingest(ctx, docPath)
		if err != nil {
			shared.err = fmt.Errorf("ingesting document: %w", err)
			eng.Close()
			return
		}
		shared.docID = docID
		t.Logf("Document ingested: ID=%d", docID)
	})
}

func skipOrSetup(t *testing.T) {
	t.Helper()
	setupShared(t)
	if shared.err != nil {
		t.Skipf("shared setup failed: %v", shared.err)
	}
}

// createTestDOCX creates a minimal DOCX file with engineering content.
func createTestDOCX(dir string) string {
	path := filepath.Join(dir, "spec-doc.docx")

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	ct, _ := w.Create("[Content_Types].xml")
	ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))

	rels, _ := w.Create("_rels/.rels")
	rels.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))

	doc, _ := w.Create("word/document.xml")
	doc.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>Material Specifications</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>This document defines the material requirements for the structural components used in the bridge construction project. All materials shall comply with ISO 9001 quality management standards.</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading2"/></w:pPr>
      <w:r><w:t>Section 3.2 Tensile Strength Requirements</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>The minimum tensile strength for Grade A structural steel shall be 500 MPa as measured according to ASTM D638 testing procedures. Each batch of material must be tested and certified before use on site. The contractor shall maintain records of all test results for a minimum period of 10 years.</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading2"/></w:pPr>
      <w:r><w:t>Section 4.1 Definitions</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>"Force Majeure" means any event or circumstance beyond the reasonable control of a party, including but not limited to acts of God, war, terrorism, pandemic, earthquake, flood, or government action that prevents a party from performing its obligations under this contract.</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading2"/></w:pPr>
      <w:r><w:t>Section 5.0 Quality Assurance</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>The quality management system shall be certified to ISO 9001:2015. Audits shall be conducted quarterly by an independent third-party auditor. Non-conformance reports must be resolved within 30 business days. The project manager, John Smith, is responsible for overall quality oversight.</w:t></w:r>
    </w:p>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading2"/></w:pPr>
      <w:r><w:t>Section 6.0 Contract Terms</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>This contract is effective from January 1, 2025 and shall remain in force for a period of 36 months unless terminated earlier in accordance with the provisions set forth herein. The total contract value is USD 2,500,000. Payment shall be made in monthly installments based on certified progress.</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`))

	w.Close()
	os.WriteFile(path, buf.Bytes(), 0644)
	return path
}

// --- Engine creation tests ---

func TestIntegrationEngineNew(t *testing.T) {
	if !ollamaAvailable() {
		t.Skip("Ollama not reachable")
	}

	dir := t.TempDir()
	cfg := Config{
		DBPath: filepath.Join(dir, "test.db"),
		Chat: LLMConfig{
			Provider: "ollama",
			Model:    chatModel,
			BaseURL:  ollamaURL,
		},
		Embedding: LLMConfig{
			Provider: "ollama",
			Model:    embedModel,
			BaseURL:  ollamaURL,
		},
		EmbeddingDim: embedDim,
	}

	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer eng.Close()

	docs, err := eng.ListDocuments(context.Background())
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 documents in fresh DB, got %d", len(docs))
	}
}

// --- Ingest tests ---

func TestIntegrationIngestDOCX(t *testing.T) {
	skipOrSetup(t)

	if shared.docID <= 0 {
		t.Fatalf("expected valid docID, got %d", shared.docID)
	}

	ctx := context.Background()
	docs, err := shared.eng.ListDocuments(ctx)
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) < 1 {
		t.Fatalf("expected at least 1 document, got %d", len(docs))
	}

	doc := docs[0]
	if doc.Format != "docx" {
		t.Errorf("document format: got %q, want %q", doc.Format, "docx")
	}
	if doc.Status != "ready" {
		t.Errorf("document status: got %q, want %q", doc.Status, "ready")
	}
	if doc.Filename != "spec-doc.docx" {
		t.Errorf("document filename: got %q, want %q", doc.Filename, "spec-doc.docx")
	}
}

func TestIntegrationIngestIdempotent(t *testing.T) {
	skipOrSetup(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Re-ingest same document; should return same ID (hash unchanged).
	id2, err := shared.eng.Ingest(ctx, shared.docPath)
	if err != nil {
		t.Fatalf("second Ingest: %v", err)
	}
	if shared.docID != id2 {
		t.Errorf("idempotent Ingest: got different IDs %d vs %d", shared.docID, id2)
	}
}

// --- Query tests ---

func TestIntegrationQueryTensileStrength(t *testing.T) {
	skipOrSetup(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	answer, err := shared.eng.Query(ctx, "What is the minimum tensile strength requirement?")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if answer.Text == "" {
		t.Fatal("Query returned empty answer text")
	}
	if answer.Rounds < 1 {
		t.Errorf("expected at least 1 reasoning round, got %d", answer.Rounds)
	}
	if answer.ModelUsed == "" {
		t.Error("expected ModelUsed to be set")
	}
	if len(answer.Sources) == 0 {
		t.Error("expected at least one source in the answer")
	}
	if len(answer.Reasoning) == 0 {
		t.Error("expected at least one reasoning step")
	}

	lowerAnswer := strings.ToLower(answer.Text)
	if !strings.Contains(lowerAnswer, "500") {
		t.Errorf("answer should mention 500 MPa, got: %s", answer.Text)
	}

	t.Logf("Answer: %s", answer.Text)
	t.Logf("Confidence: %.2f, Rounds: %d, Sources: %d",
		answer.Confidence, answer.Rounds, len(answer.Sources))
}

func TestIntegrationQueryForceMajeure(t *testing.T) {
	skipOrSetup(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	answer, err := shared.eng.Query(ctx, "What is the definition of Force Majeure?")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if answer.Text == "" {
		t.Fatal("Query returned empty answer text")
	}

	lowerAnswer := strings.ToLower(answer.Text)
	if !strings.Contains(lowerAnswer, "control") && !strings.Contains(lowerAnswer, "event") {
		t.Errorf("answer should describe force majeure, got: %s", answer.Text)
	}

	t.Logf("Answer: %s", answer.Text)
	t.Logf("Confidence: %.2f, Rounds: %d", answer.Confidence, answer.Rounds)
}

func TestIntegrationQueryQualityAssurance(t *testing.T) {
	skipOrSetup(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	answer, err := shared.eng.Query(ctx,
		"Who is responsible for quality oversight and what standard must be followed?")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if answer.Text == "" {
		t.Fatal("Query returned empty answer text")
	}

	lowerAnswer := strings.ToLower(answer.Text)
	hasName := strings.Contains(lowerAnswer, "john smith") ||
		strings.Contains(lowerAnswer, "project manager")
	hasISO := strings.Contains(lowerAnswer, "iso 9001") ||
		strings.Contains(lowerAnswer, "iso")

	if !hasName {
		t.Errorf("answer should mention John Smith or project manager, got: %s", answer.Text)
	}
	if !hasISO {
		t.Errorf("answer should mention ISO 9001, got: %s", answer.Text)
	}

	t.Logf("Answer: %s", answer.Text)
}

func TestIntegrationQueryContractTerms(t *testing.T) {
	skipOrSetup(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	queries := []struct {
		name     string
		question string
		expect   string
	}{
		{"contract_value", "What is the total contract value?", "2,500,000"},
		{"contract_duration", "How long is the contract effective?", "36"},
		{"testing_standard", "What testing standard is referenced for tensile strength?", "astm"},
		{"audit_frequency", "How often are quality audits conducted?", "quarter"},
	}

	for _, q := range queries {
		t.Run(q.name, func(t *testing.T) {
			answer, err := shared.eng.Query(ctx, q.question)
			if err != nil {
				t.Fatalf("Query(%q): %v", q.question, err)
			}
			if answer.Text == "" {
				t.Fatalf("empty answer for: %s", q.question)
			}
			if !strings.Contains(strings.ToLower(answer.Text), strings.ToLower(q.expect)) {
				t.Errorf("answer for %q should contain %q, got: %s",
					q.question, q.expect, answer.Text)
			}
			t.Logf("Q: %s\nA: %s\nConfidence: %.2f",
				q.question, answer.Text, answer.Confidence)
		})
	}
}

func TestIntegrationQueryNoResults(t *testing.T) {
	if !ollamaAvailable() {
		t.Skip("Ollama not reachable")
	}

	// Use a separate empty engine for this test.
	dir := t.TempDir()
	cfg := Config{
		DBPath: filepath.Join(dir, "empty.db"),
		Chat: LLMConfig{
			Provider: "ollama",
			Model:    chatModel,
			BaseURL:  ollamaURL,
		},
		Embedding: LLMConfig{
			Provider: "ollama",
			Model:    embedModel,
			BaseURL:  ollamaURL,
		},
		EmbeddingDim: embedDim,
	}
	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer eng.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = eng.Query(ctx, "What is the tensile strength?")
	if err == nil {
		t.Fatal("expected error querying empty database")
	}
}

func TestIntegrationQueryWithOptions(t *testing.T) {
	skipOrSetup(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	answer, err := shared.eng.Query(ctx, "What is the tensile strength?",
		WithMaxRounds(1),
		WithMaxResults(5),
	)
	if err != nil {
		t.Fatalf("Query with options: %v", err)
	}

	if answer.Rounds != 1 {
		t.Errorf("expected 1 round with MaxRounds=1, got %d", answer.Rounds)
	}
	if answer.Text == "" {
		t.Error("empty answer")
	}

	t.Logf("Single-round answer: %s", answer.Text)
}

// --- Answer structure test ---

func TestIntegrationAnswerStructure(t *testing.T) {
	skipOrSetup(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	answer, err := shared.eng.Query(ctx, "What is the effective date of the contract?")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if answer.Text == "" {
		t.Error("Text is empty")
	}
	if answer.Confidence < 0 || answer.Confidence > 1 {
		t.Errorf("Confidence out of range [0,1]: %f", answer.Confidence)
	}
	if answer.Rounds < 1 {
		t.Errorf("Rounds < 1: %d", answer.Rounds)
	}
	if answer.ModelUsed == "" {
		t.Error("ModelUsed is empty")
	}

	if len(answer.Sources) == 0 {
		t.Fatal("no sources returned")
	}
	for i, src := range answer.Sources {
		if src.ChunkID <= 0 {
			t.Errorf("source[%d].ChunkID invalid: %d", i, src.ChunkID)
		}
		if src.DocumentID != shared.docID {
			t.Errorf("source[%d].DocumentID: got %d, want %d",
				i, src.DocumentID, shared.docID)
		}
		if src.Content == "" {
			t.Errorf("source[%d].Content is empty", i)
		}
		if src.Filename == "" {
			t.Errorf("source[%d].Filename is empty", i)
		}
	}

	if len(answer.Reasoning) == 0 {
		t.Fatal("no reasoning steps returned")
	}
	for i, step := range answer.Reasoning {
		if step.Round < 1 {
			t.Errorf("reasoning[%d].Round < 1: %d", i, step.Round)
		}
		if step.Action == "" {
			t.Errorf("reasoning[%d].Action is empty", i)
		}
	}

	t.Logf("Answer: %s", answer.Text)
	t.Logf("Confidence: %.2f, Model: %s, Rounds: %d, Sources: %d, Steps: %d",
		answer.Confidence, answer.ModelUsed, answer.Rounds,
		len(answer.Sources), len(answer.Reasoning))
}

// --- Update tests ---

func TestIntegrationUpdateNoChange(t *testing.T) {
	skipOrSetup(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	changed, err := shared.eng.Update(ctx, shared.docPath)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if changed {
		t.Error("Update should return false for unchanged document")
	}
}

// --- Delete test (uses a separate engine to avoid breaking shared state) ---

func TestIntegrationDelete(t *testing.T) {
	if !ollamaAvailable() {
		t.Skip("Ollama not reachable")
	}
	// Warm embed model for this separate test (it may have been swapped out).
	warmEmbedModel(embedModel)

	dir := t.TempDir()
	cfg := Config{
		DBPath: filepath.Join(dir, "delete_test.db"),
		Chat: LLMConfig{
			Provider: "ollama",
			Model:    chatModel,
			BaseURL:  ollamaURL,
		},
		Embedding: LLMConfig{
			Provider: "ollama",
			Model:    embedModel,
			BaseURL:  ollamaURL,
		},
		WeightVector:        1.0,
		WeightFTS:           1.0,
		WeightGraph:         0.5,
		MaxChunkTokens:      512,
		ChunkOverlap:        64,
		MaxRounds:           1,
		ConfidenceThreshold: 0.3,
		EmbeddingDim:        embedDim,
	}

	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer eng.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	docPath := createTestDOCX(dir)
	docID, err := eng.Ingest(ctx, docPath)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	if err := eng.Delete(ctx, docID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	docs, err := eng.ListDocuments(ctx)
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 documents after delete, got %d", len(docs))
	}
}
