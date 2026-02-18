package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	goreason "github.com/bbiangul/go-reason"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "GOOGLE_API_KEY not set")
		os.Exit(1)
	}

	tmpDir, _ := os.MkdirTemp("", "goreason-e2e-*")
	defer os.RemoveAll(tmpDir)
	dbPath := tmpDir + "/test.db"

	cfg := goreason.Config{
		DBPath: dbPath,
		Chat: goreason.LLMConfig{
			Provider: "gemini",
			Model:    "gemini-2.5-flash",
			APIKey:   apiKey,
		},
		Embedding: goreason.LLMConfig{
			Provider: "gemini",
			Model:    "gemini-embedding-001",
			APIKey:   apiKey,
		},
		WeightVector:        1.0,
		WeightFTS:           1.0,
		WeightGraph:         0.5,
		MaxChunkTokens:      1024,
		ChunkOverlap:        128,
		MaxRounds:           1,
		ConfidenceThreshold: 0.7,
		EmbeddingDim:        3072,
		SkipGraph:           true, // faster for this test
	}

	engine, err := goreason.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating engine: %v\n", err)
		os.Exit(1)
	}
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Ingest
	docPath := "data/corpus/cuad/ACCURAYINC_09_01_2010-EX-10.31-DISTRIBUTOR AGREEMENT.txt"
	fmt.Fprintf(os.Stderr, "\n=== INGESTING %s ===\n", docPath)
	docID, err := engine.Ingest(ctx, docPath, goreason.WithMetadata(map[string]string{
		"type": "legal", "dataset": "cuad",
	}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ingest error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Ingested doc_id=%d\n", docID)

	// Query
	question := "What are the termination conditions in this agreement?"
	fmt.Fprintf(os.Stderr, "\n=== QUERYING: %s ===\n", question)
	answer, err := engine.Query(ctx, question, goreason.WithMaxResults(5), goreason.WithMaxRounds(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "query error: %v\n", err)
		os.Exit(1)
	}

	// Print just the enriched sources to stdout
	type sourceView struct {
		ChunkID          int64             `json:"chunk_id"`
		DocumentID       int64             `json:"document_id"`
		Filename         string            `json:"filename"`
		Path             string            `json:"path,omitempty"`
		Heading          string            `json:"heading"`
		ChunkType        string            `json:"chunk_type,omitempty"`
		PageNumber       int               `json:"page_number"`
		PositionInDoc    int               `json:"position_in_doc,omitempty"`
		Score            float64           `json:"score"`
		ChunkMetadata    map[string]string `json:"chunk_metadata,omitempty"`
		DocumentMetadata map[string]string `json:"document_metadata,omitempty"`
		Snippet          string            `json:"snippet,omitempty"`
		ContentLen       int               `json:"content_length"`
		ImageCount       int               `json:"image_count"`
	}

	fmt.Fprintf(os.Stderr, "\n=== ANSWER ===\n%s\n", answer.Text)

	var sources []sourceView
	for _, s := range answer.Sources {
		sources = append(sources, sourceView{
			ChunkID:          s.ChunkID,
			DocumentID:       s.DocumentID,
			Filename:         s.Filename,
			Path:             s.Path,
			Heading:          s.Heading,
			ChunkType:        s.ChunkType,
			PageNumber:       s.PageNumber,
			PositionInDoc:    s.PositionInDoc,
			Score:            s.Score,
			ChunkMetadata:    s.ChunkMetadata,
			DocumentMetadata: s.DocumentMetadata,
			Snippet:          s.Snippet,
			ContentLen:       len(s.Content),
			ImageCount:       len(s.Images),
		})
	}

	out, _ := json.MarshalIndent(sources, "", "  ")
	fmt.Println(string(out))
}
