//go:build cgo

package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := New(dbPath, 4) // dim=4 for test vectors
	if err != nil {
		t.Fatalf("creating store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// ---------------------------------------------------------------------------
// Schema / construction
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	s := newTestStore(t)
	if s.EmbeddingDim() != 4 {
		t.Fatalf("expected embedding dim 4, got %d", s.EmbeddingDim())
	}
	if s.DB() == nil {
		t.Fatal("expected non-nil *sql.DB")
	}
}

func TestNewCreatesParentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "dir")
	dbPath := filepath.Join(dir, "test.db")
	s, err := New(dbPath, 4)
	if err != nil {
		t.Fatalf("creating store in nested dir: %v", err)
	}
	s.Close()
}

// ---------------------------------------------------------------------------
// Document CRUD
// ---------------------------------------------------------------------------

func sampleDoc(path string) Document {
	return Document{
		Path:        path,
		Filename:    "test.pdf",
		Format:      "pdf",
		ContentHash: "abc123",
		ParseMethod: "native",
		Status:      "pending",
		Metadata:    `{"pages":10}`,
	}
}

func TestUpsertAndGetDocument(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	doc := sampleDoc("/tmp/test.pdf")
	id, err := s.UpsertDocument(ctx, doc)
	if err != nil {
		t.Fatalf("upserting document: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero document id")
	}

	// Get by ID
	got, err := s.GetDocument(ctx, id)
	if err != nil {
		t.Fatalf("getting document by id: %v", err)
	}
	if got.Path != doc.Path {
		t.Errorf("path: got %q, want %q", got.Path, doc.Path)
	}
	if got.Filename != doc.Filename {
		t.Errorf("filename: got %q, want %q", got.Filename, doc.Filename)
	}
	if got.Format != doc.Format {
		t.Errorf("format: got %q, want %q", got.Format, doc.Format)
	}
	if got.Status != "pending" {
		t.Errorf("status: got %q, want %q", got.Status, "pending")
	}
}

func TestGetDocumentByPath(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	doc := sampleDoc("/docs/report.pdf")
	_, err := s.UpsertDocument(ctx, doc)
	if err != nil {
		t.Fatalf("upserting: %v", err)
	}

	got, err := s.GetDocumentByPath(ctx, "/docs/report.pdf")
	if err != nil {
		t.Fatalf("getting by path: %v", err)
	}
	if got.Filename != "test.pdf" {
		t.Errorf("filename: got %q, want %q", got.Filename, "test.pdf")
	}
}

func TestGetDocumentByPathNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.GetDocumentByPath(ctx, "/nonexistent")
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpsertDocumentUpdate(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	doc := sampleDoc("/tmp/update.pdf")
	id1, err := s.UpsertDocument(ctx, doc)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Upsert again with different hash -- same path triggers UPDATE.
	doc.ContentHash = "def456"
	doc.Status = "ready"
	id2, err := s.UpsertDocument(ctx, doc)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if id2 != id1 {
		t.Fatalf("upsert returned different id: %d vs %d", id2, id1)
	}

	got, err := s.GetDocument(ctx, id1)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.ContentHash != "def456" {
		t.Errorf("content_hash not updated: got %q", got.ContentHash)
	}
}

func TestListDocuments(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	for i, p := range []string{"/a.pdf", "/b.pdf", "/c.pdf"} {
		doc := sampleDoc(p)
		doc.Filename = p
		if _, err := s.UpsertDocument(ctx, doc); err != nil {
			t.Fatalf("insert doc %d: %v", i, err)
		}
	}

	docs, err := s.ListDocuments(ctx)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(docs))
	}
}

func TestUpdateDocumentStatus(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id, err := s.UpsertDocument(ctx, sampleDoc("/status.pdf"))
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := s.UpdateDocumentStatus(ctx, id, "ready"); err != nil {
		t.Fatalf("update status: %v", err)
	}

	got, err := s.GetDocument(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "ready" {
		t.Errorf("status: got %q, want %q", got.Status, "ready")
	}
}

// ---------------------------------------------------------------------------
// DeleteDocument (cascade)
// ---------------------------------------------------------------------------

func TestDeleteDocument(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id, err := s.UpsertDocument(ctx, sampleDoc("/delete.pdf"))
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Insert chunks for the document.
	chunks := []Chunk{
		{DocumentID: id, Content: "chunk one", ChunkType: "paragraph", Heading: "H1", PositionInDoc: 0, TokenCount: 2},
	}
	chunkIDs, err := s.InsertChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("insert chunks: %v", err)
	}

	// Insert an embedding for the chunk.
	if err := s.InsertEmbedding(ctx, chunkIDs[0], []float32{1, 0, 0, 0}); err != nil {
		t.Fatalf("insert embedding: %v", err)
	}

	// Delete the document; cascaded data should also be removed.
	if err := s.DeleteDocument(ctx, id); err != nil {
		t.Fatalf("delete document: %v", err)
	}

	_, err = s.GetDocument(ctx, id)
	if err != sql.ErrNoRows {
		t.Fatalf("expected document gone, got err=%v", err)
	}

	remaining, err := s.GetChunksByDocument(ctx, id)
	if err != nil {
		t.Fatalf("get chunks after delete: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected 0 chunks after cascade, got %d", len(remaining))
	}
}

// ---------------------------------------------------------------------------
// Chunk operations
// ---------------------------------------------------------------------------

func TestInsertAndGetChunks(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, err := s.UpsertDocument(ctx, sampleDoc("/chunks.pdf"))
	if err != nil {
		t.Fatalf("upsert doc: %v", err)
	}

	chunks := []Chunk{
		{DocumentID: docID, Content: "first chunk", ChunkType: "paragraph", Heading: "Intro", PageNumber: 1, PositionInDoc: 0, TokenCount: 2},
		{DocumentID: docID, Content: "second chunk", ChunkType: "paragraph", Heading: "Body", PageNumber: 1, PositionInDoc: 1, TokenCount: 2},
		{DocumentID: docID, Content: "third chunk", ChunkType: "section", Heading: "Conclusion", PageNumber: 2, PositionInDoc: 2, TokenCount: 2},
	}

	ids, err := s.InsertChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("inserting chunks: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(ids))
	}

	got, err := s.GetChunksByDocument(ctx, docID)
	if err != nil {
		t.Fatalf("getting chunks: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(got))
	}

	// Verify ordering by position_in_doc.
	if got[0].Content != "first chunk" {
		t.Errorf("first chunk content: got %q", got[0].Content)
	}
	if got[2].Heading != "Conclusion" {
		t.Errorf("third chunk heading: got %q", got[2].Heading)
	}
	// content_hash should be populated automatically.
	if got[0].ContentHash == "" {
		t.Error("expected non-empty content_hash")
	}
}

// ---------------------------------------------------------------------------
// Embedding / vector search
// ---------------------------------------------------------------------------

func TestInsertEmbeddingAndVectorSearch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, err := s.UpsertDocument(ctx, sampleDoc("/vec.pdf"))
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	chunks := []Chunk{
		{DocumentID: docID, Content: "alpha content", ChunkType: "paragraph", Heading: "A", PositionInDoc: 0, TokenCount: 2},
		{DocumentID: docID, Content: "beta content", ChunkType: "paragraph", Heading: "B", PositionInDoc: 1, TokenCount: 2},
	}
	ids, err := s.InsertChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("insert chunks: %v", err)
	}

	// Orthogonal embeddings so distance is clear.
	if err := s.InsertEmbedding(ctx, ids[0], []float32{1, 0, 0, 0}); err != nil {
		t.Fatalf("embedding 0: %v", err)
	}
	if err := s.InsertEmbedding(ctx, ids[1], []float32{0, 1, 0, 0}); err != nil {
		t.Fatalf("embedding 1: %v", err)
	}

	// Query vector close to first embedding.
	results, err := s.VectorSearch(ctx, []float32{1, 0, 0, 0}, 2)
	if err != nil {
		t.Fatalf("vector search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First result should be the one with embedding {1,0,0,0}.
	if results[0].Content != "alpha content" {
		t.Errorf("expected nearest to be 'alpha content', got %q", results[0].Content)
	}
	if results[0].Filename != "test.pdf" {
		t.Errorf("filename: got %q, want %q", results[0].Filename, "test.pdf")
	}

	// The nearest result should have a higher score than the second.
	if results[0].Score <= results[1].Score {
		t.Errorf("expected first result score (%f) > second (%f)", results[0].Score, results[1].Score)
	}
}

func TestVectorSearchTopK(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, _ := s.UpsertDocument(ctx, sampleDoc("/topk.pdf"))
	chunks := []Chunk{
		{DocumentID: docID, Content: "c1", ChunkType: "p", PositionInDoc: 0, TokenCount: 1},
		{DocumentID: docID, Content: "c2", ChunkType: "p", PositionInDoc: 1, TokenCount: 1},
		{DocumentID: docID, Content: "c3", ChunkType: "p", PositionInDoc: 2, TokenCount: 1},
	}
	ids, _ := s.InsertChunks(ctx, chunks)

	_ = s.InsertEmbedding(ctx, ids[0], []float32{1, 0, 0, 0})
	_ = s.InsertEmbedding(ctx, ids[1], []float32{0, 1, 0, 0})
	_ = s.InsertEmbedding(ctx, ids[2], []float32{0, 0, 1, 0})

	// Request only top-1.
	results, err := s.VectorSearch(ctx, []float32{0, 0, 1, 0}, 1)
	if err != nil {
		t.Fatalf("vector search k=1: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Content != "c3" {
		t.Errorf("expected c3, got %q", results[0].Content)
	}
}

// ---------------------------------------------------------------------------
// FTS search
// ---------------------------------------------------------------------------

func TestFTSSearch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, err := s.UpsertDocument(ctx, sampleDoc("/fts.pdf"))
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	chunks := []Chunk{
		{DocumentID: docID, Content: "the quick brown fox jumps over the lazy dog", ChunkType: "paragraph", Heading: "Animals", PositionInDoc: 0, TokenCount: 9},
		{DocumentID: docID, Content: "artificial intelligence and machine learning", ChunkType: "paragraph", Heading: "Tech", PositionInDoc: 1, TokenCount: 5},
		{DocumentID: docID, Content: "quantum computing uses qubits", ChunkType: "paragraph", Heading: "Physics", PositionInDoc: 2, TokenCount: 4},
	}
	if _, err := s.InsertChunks(ctx, chunks); err != nil {
		t.Fatalf("insert chunks: %v", err)
	}

	results, err := s.FTSSearch(ctx, "artificial intelligence", 10)
	if err != nil {
		t.Fatalf("fts search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one FTS result")
	}
	if results[0].Content != "artificial intelligence and machine learning" {
		t.Errorf("top FTS result: got %q", results[0].Content)
	}
	if results[0].Score <= 0 {
		t.Errorf("expected positive score, got %f", results[0].Score)
	}
}

func TestFTSSearchNoMatch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, _ := s.UpsertDocument(ctx, sampleDoc("/fts2.pdf"))
	chunks := []Chunk{
		{DocumentID: docID, Content: "hello world", ChunkType: "paragraph", PositionInDoc: 0, TokenCount: 2},
	}
	s.InsertChunks(ctx, chunks)

	results, err := s.FTSSearch(ctx, "zzzyyyxxx", 10)
	if err != nil {
		t.Fatalf("fts search: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for nonsense query, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// Entity operations
// ---------------------------------------------------------------------------

func TestUpsertEntityAndGetByNames(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	e1 := Entity{Name: "Alice", EntityType: "person", Description: "Engineer"}
	e2 := Entity{Name: "Bob", EntityType: "person", Description: "Manager"}
	e3 := Entity{Name: "Acme", EntityType: "organization", Description: "Company"}

	id1, err := s.UpsertEntity(ctx, e1)
	if err != nil {
		t.Fatalf("upsert e1: %v", err)
	}
	id2, err := s.UpsertEntity(ctx, e2)
	if err != nil {
		t.Fatalf("upsert e2: %v", err)
	}
	id3, err := s.UpsertEntity(ctx, e3)
	if err != nil {
		t.Fatalf("upsert e3: %v", err)
	}
	if id1 == 0 || id2 == 0 || id3 == 0 {
		t.Fatal("expected non-zero entity ids")
	}

	// Get by names.
	entities, err := s.GetEntitiesByNames(ctx, []string{"Alice", "Acme"})
	if err != nil {
		t.Fatalf("get by names: %v", err)
	}
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(entities))
	}

	names := map[string]bool{}
	for _, e := range entities {
		names[e.Name] = true
	}
	if !names["Alice"] || !names["Acme"] {
		t.Errorf("missing expected entity names: %v", names)
	}
}

func TestUpsertEntityUpdate(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	e := Entity{Name: "Alice", EntityType: "person", Description: "v1"}
	id1, err := s.UpsertEntity(ctx, e)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Upsert same name+type with different description.
	e.Description = "v2"
	id2, err := s.UpsertEntity(ctx, e)
	if err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	if id2 != id1 {
		t.Fatalf("expected same id, got %d vs %d", id2, id1)
	}

	ents, err := s.GetEntitiesByNames(ctx, []string{"Alice"})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(ents) != 1 || ents[0].Description != "v2" {
		t.Errorf("expected updated description 'v2', got %q", ents[0].Description)
	}
}

func TestGetEntitiesByNamesEmpty(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	result, err := s.GetEntitiesByNames(ctx, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil for empty names, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// Relationships and graph search
// ---------------------------------------------------------------------------

func TestInsertRelationshipAndGraphSearch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, _ := s.UpsertDocument(ctx, sampleDoc("/graph.pdf"))
	chunks := []Chunk{
		{DocumentID: docID, Content: "Alice works at Acme", ChunkType: "paragraph", PositionInDoc: 0, TokenCount: 4},
	}
	chunkIDs, err := s.InsertChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("insert chunks: %v", err)
	}

	aliceID, _ := s.UpsertEntity(ctx, Entity{Name: "Alice", EntityType: "person", Description: "Engineer"})
	acmeID, _ := s.UpsertEntity(ctx, Entity{Name: "Acme", EntityType: "org", Description: "Company"})

	// Link entity to chunk.
	if err := s.LinkEntityChunk(ctx, aliceID, chunkIDs[0]); err != nil {
		t.Fatalf("link alice->chunk: %v", err)
	}
	if err := s.LinkEntityChunk(ctx, acmeID, chunkIDs[0]); err != nil {
		t.Fatalf("link acme->chunk: %v", err)
	}

	// Insert relationship.
	rel := Relationship{
		SourceEntityID: aliceID,
		TargetEntityID: acmeID,
		RelationType:   "works_at",
		Weight:         0.9,
		Description:    "Alice works at Acme",
		SourceChunkID:  &chunkIDs[0],
	}
	relID, err := s.InsertRelationship(ctx, rel)
	if err != nil {
		t.Fatalf("insert relationship: %v", err)
	}
	if relID == 0 {
		t.Fatal("expected non-zero relationship id")
	}

	// Graph search from Alice's entity.
	results, err := s.GraphSearch(ctx, []int64{aliceID}, 10)
	if err != nil {
		t.Fatalf("graph search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one graph search result")
	}
	if results[0].Content != "Alice works at Acme" {
		t.Errorf("graph result content: got %q", results[0].Content)
	}
}

func TestGraphSearchEmpty(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	result, err := s.GraphSearch(ctx, []int64{}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil for empty entity ids, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// LinkEntityChunk
// ---------------------------------------------------------------------------

func TestLinkEntityChunkIdempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, _ := s.UpsertDocument(ctx, sampleDoc("/link.pdf"))
	chunkIDs, _ := s.InsertChunks(ctx, []Chunk{
		{DocumentID: docID, Content: "data", ChunkType: "p", PositionInDoc: 0, TokenCount: 1},
	})
	entityID, _ := s.UpsertEntity(ctx, Entity{Name: "Test", EntityType: "thing", Description: "d"})

	// First link.
	if err := s.LinkEntityChunk(ctx, entityID, chunkIDs[0]); err != nil {
		t.Fatalf("first link: %v", err)
	}
	// Second link (INSERT OR IGNORE) should not fail.
	if err := s.LinkEntityChunk(ctx, entityID, chunkIDs[0]); err != nil {
		t.Fatalf("duplicate link should not error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Community operations
// ---------------------------------------------------------------------------

func TestInsertAndGetCommunities(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	c1 := Community{Level: 0, Summary: "Community A", EntityIDs: "[1,2]"}
	c2 := Community{Level: 0, Summary: "Community B", EntityIDs: "[3,4]"}
	c3 := Community{Level: 1, Summary: "Super community", EntityIDs: "[1,2,3,4]"}

	id1, err := s.InsertCommunity(ctx, c1)
	if err != nil {
		t.Fatalf("insert c1: %v", err)
	}
	if id1 == 0 {
		t.Fatal("expected non-zero community id")
	}
	if _, err := s.InsertCommunity(ctx, c2); err != nil {
		t.Fatalf("insert c2: %v", err)
	}
	if _, err := s.InsertCommunity(ctx, c3); err != nil {
		t.Fatalf("insert c3: %v", err)
	}

	// Get level 0.
	got, err := s.GetCommunities(ctx, 0)
	if err != nil {
		t.Fatalf("get communities level 0: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 level-0 communities, got %d", len(got))
	}

	// Get level 1.
	got1, err := s.GetCommunities(ctx, 1)
	if err != nil {
		t.Fatalf("get communities level 1: %v", err)
	}
	if len(got1) != 1 {
		t.Fatalf("expected 1 level-1 community, got %d", len(got1))
	}
	if got1[0].Summary != "Super community" {
		t.Errorf("summary: got %q", got1[0].Summary)
	}
}

func TestClearCommunities(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	s.InsertCommunity(ctx, Community{Level: 0, Summary: "x", EntityIDs: "[1]"})

	if err := s.ClearCommunities(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}

	got, err := s.GetCommunities(ctx, 0)
	if err != nil {
		t.Fatalf("get after clear: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 communities after clear, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// Query log
// ---------------------------------------------------------------------------

func TestLogQuery(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	q := QueryLog{
		Query:           "What is Go?",
		Answer:          "A programming language",
		Confidence:      0.95,
		Sources:         []string{"doc1.pdf"},
		RetrievalMethod: "hybrid",
		ModelUsed:       "llama3",
		Rounds:          2,
	}

	if err := s.LogQuery(ctx, q); err != nil {
		t.Fatalf("log query: %v", err)
	}

	// Verify by reading directly from the table.
	var count int
	err := s.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM query_log").Scan(&count)
	if err != nil {
		t.Fatalf("count query_log: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 log entry, got %d", count)
	}

	var query, answer string
	err = s.DB().QueryRowContext(ctx, "SELECT query, answer FROM query_log LIMIT 1").Scan(&query, &answer)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	if query != "What is Go?" {
		t.Errorf("query: got %q", query)
	}
	if answer != "A programming language" {
		t.Errorf("answer: got %q", answer)
	}
}

// ---------------------------------------------------------------------------
// DeleteDocumentData (keeps document, removes chunks)
// ---------------------------------------------------------------------------

func TestDeleteDocumentData(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, _ := s.UpsertDocument(ctx, sampleDoc("/deldata.pdf"))
	chunks := []Chunk{
		{DocumentID: docID, Content: "keep me?", ChunkType: "p", PositionInDoc: 0, TokenCount: 2},
		{DocumentID: docID, Content: "and me?", ChunkType: "p", PositionInDoc: 1, TokenCount: 2},
	}
	chunkIDs, _ := s.InsertChunks(ctx, chunks)

	// Add embeddings and entity links.
	_ = s.InsertEmbedding(ctx, chunkIDs[0], []float32{1, 0, 0, 0})
	_ = s.InsertEmbedding(ctx, chunkIDs[1], []float32{0, 1, 0, 0})

	eID, _ := s.UpsertEntity(ctx, Entity{Name: "E", EntityType: "t", Description: "d"})
	_ = s.LinkEntityChunk(ctx, eID, chunkIDs[0])

	// Delete data but keep document.
	if err := s.DeleteDocumentData(ctx, docID); err != nil {
		t.Fatalf("delete document data: %v", err)
	}

	// Document should still exist.
	doc, err := s.GetDocument(ctx, docID)
	if err != nil {
		t.Fatalf("document should still exist: %v", err)
	}
	if doc.Path != "/deldata.pdf" {
		t.Errorf("path: got %q", doc.Path)
	}

	// Chunks should be gone.
	remaining, err := s.GetChunksByDocument(ctx, docID)
	if err != nil {
		t.Fatalf("get chunks: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected 0 chunks after data delete, got %d", len(remaining))
	}

	// Vector search should return no results for this doc's embeddings.
	results, err := s.VectorSearch(ctx, []float32{1, 0, 0, 0}, 10)
	if err != nil {
		t.Fatalf("vector search after delete: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 vector results after data delete, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// New metadata fields in search results
// ---------------------------------------------------------------------------

func TestVectorSearchReturnsMetadataFields(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	doc := Document{
		Path: "/meta/test.pdf", Filename: "test.pdf", Format: "pdf",
		ContentHash: "h1", ParseMethod: "native", Status: "ready",
		Metadata: `{"author":"Jane"}`,
	}
	docID, err := s.UpsertDocument(ctx, doc)
	if err != nil {
		t.Fatalf("upsert doc: %v", err)
	}

	chunks := []Chunk{
		{
			DocumentID: docID, Content: "important clause", ChunkType: "table",
			Heading: "Section 3.2", PageNumber: 5, PositionInDoc: 7, TokenCount: 2,
			Metadata: `{"source":"appendix"}`,
		},
	}
	ids, err := s.InsertChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("insert chunks: %v", err)
	}

	if err := s.InsertEmbedding(ctx, ids[0], []float32{1, 0, 0, 0}); err != nil {
		t.Fatalf("embedding: %v", err)
	}

	results, err := s.VectorSearch(ctx, []float32{1, 0, 0, 0}, 1)
	if err != nil {
		t.Fatalf("vector search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.DocumentID != docID {
		t.Errorf("DocumentID: got %d, want %d", r.DocumentID, docID)
	}
	if r.ChunkType != "table" {
		t.Errorf("ChunkType: got %q, want %q", r.ChunkType, "table")
	}
	if r.PositionInDoc != 7 {
		t.Errorf("PositionInDoc: got %d, want 7", r.PositionInDoc)
	}
	if r.PageNumber != 5 {
		t.Errorf("PageNumber: got %d, want 5", r.PageNumber)
	}
	if r.Path != "/meta/test.pdf" {
		t.Errorf("Path: got %q, want %q", r.Path, "/meta/test.pdf")
	}
	if r.ChunkMeta != `{"source":"appendix"}` {
		t.Errorf("ChunkMeta: got %q", r.ChunkMeta)
	}
	if r.DocMeta != `{"author":"Jane"}` {
		t.Errorf("DocMeta: got %q", r.DocMeta)
	}
}

func TestFTSSearchReturnsMetadataFields(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	doc := Document{
		Path: "/fts-meta/contract.pdf", Filename: "contract.pdf", Format: "pdf",
		ContentHash: "h2", ParseMethod: "native", Status: "ready",
		Metadata: `{"type":"legal"}`,
	}
	docID, err := s.UpsertDocument(ctx, doc)
	if err != nil {
		t.Fatalf("upsert doc: %v", err)
	}

	chunks := []Chunk{
		{
			DocumentID: docID, Content: "the indemnification clause states liability",
			ChunkType: "definition", Heading: "Clause 4.1", PageNumber: 3,
			PositionInDoc: 2, TokenCount: 5, Metadata: `{"clause":"4.1"}`,
		},
	}
	if _, err := s.InsertChunks(ctx, chunks); err != nil {
		t.Fatalf("insert chunks: %v", err)
	}

	results, err := s.FTSSearch(ctx, "indemnification liability", 1)
	if err != nil {
		t.Fatalf("fts search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.DocumentID != docID {
		t.Errorf("DocumentID: got %d, want %d", r.DocumentID, docID)
	}
	if r.ChunkType != "definition" {
		t.Errorf("ChunkType: got %q, want %q", r.ChunkType, "definition")
	}
	if r.PositionInDoc != 2 {
		t.Errorf("PositionInDoc: got %d, want 2", r.PositionInDoc)
	}
	if r.ChunkMeta != `{"clause":"4.1"}` {
		t.Errorf("ChunkMeta: got %q", r.ChunkMeta)
	}
	if r.DocMeta != `{"type":"legal"}` {
		t.Errorf("DocMeta: got %q", r.DocMeta)
	}
}

func TestGraphSearchReturnsMetadataFields(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	doc := Document{
		Path: "/graph-meta/spec.pdf", Filename: "spec.pdf", Format: "pdf",
		ContentHash: "h3", ParseMethod: "native", Status: "ready",
		Metadata: `{"version":"2.0"}`,
	}
	docID, err := s.UpsertDocument(ctx, doc)
	if err != nil {
		t.Fatalf("upsert doc: %v", err)
	}

	chunks := []Chunk{
		{
			DocumentID: docID, Content: "motor rated at 5kW",
			ChunkType: "requirement", Heading: "Section 7", PageNumber: 12,
			PositionInDoc: 4, TokenCount: 4, Metadata: `{"req_id":"R-101"}`,
		},
	}
	chunkIDs, err := s.InsertChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("insert chunks: %v", err)
	}

	entityID, _ := s.UpsertEntity(ctx, Entity{Name: "Motor", EntityType: "component", Description: "5kW motor"})
	_ = s.LinkEntityChunk(ctx, entityID, chunkIDs[0])

	results, err := s.GraphSearch(ctx, []int64{entityID}, 1)
	if err != nil {
		t.Fatalf("graph search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.DocumentID != docID {
		t.Errorf("DocumentID: got %d, want %d", r.DocumentID, docID)
	}
	if r.ChunkType != "requirement" {
		t.Errorf("ChunkType: got %q, want %q", r.ChunkType, "requirement")
	}
	if r.PositionInDoc != 4 {
		t.Errorf("PositionInDoc: got %d, want 4", r.PositionInDoc)
	}
	if r.ChunkMeta != `{"req_id":"R-101"}` {
		t.Errorf("ChunkMeta: got %q", r.ChunkMeta)
	}
	if r.DocMeta != `{"version":"2.0"}` {
		t.Errorf("DocMeta: got %q", r.DocMeta)
	}
}

// ---------------------------------------------------------------------------
// Chunk image CRUD
// ---------------------------------------------------------------------------

func TestInsertAndGetChunkImages(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, _ := s.UpsertDocument(ctx, sampleDoc("/images.pdf"))
	chunkIDs, _ := s.InsertChunks(ctx, []Chunk{
		{DocumentID: docID, Content: "page with diagram", ChunkType: "paragraph", PositionInDoc: 0, TokenCount: 3},
		{DocumentID: docID, Content: "page with photo", ChunkType: "paragraph", PositionInDoc: 1, TokenCount: 3},
	})

	images := []ChunkImage{
		{ChunkID: chunkIDs[0], DocumentID: docID, Caption: "Wiring diagram", MIMEType: "image/png", Width: 800, Height: 600, PageNumber: 1, Data: []byte("fake-png-data")},
		{ChunkID: chunkIDs[0], DocumentID: docID, Caption: "", MIMEType: "image/jpeg", Width: 100, Height: 50, PageNumber: 1, Data: []byte("small-img")},
		{ChunkID: chunkIDs[1], DocumentID: docID, Caption: "Photo of motor", MIMEType: "image/jpeg", Width: 640, Height: 480, PageNumber: 2, Data: []byte("motor-photo")},
	}
	if err := s.InsertChunkImages(ctx, images); err != nil {
		t.Fatalf("insert chunk images: %v", err)
	}

	// Get with data
	result, err := s.GetImagesByChunkIDs(ctx, []int64{chunkIDs[0], chunkIDs[1]}, true)
	if err != nil {
		t.Fatalf("get images with data: %v", err)
	}
	if len(result[chunkIDs[0]]) != 2 {
		t.Fatalf("chunk[0] images: expected 2, got %d", len(result[chunkIDs[0]]))
	}
	if len(result[chunkIDs[1]]) != 1 {
		t.Fatalf("chunk[1] images: expected 1, got %d", len(result[chunkIDs[1]]))
	}

	img0 := result[chunkIDs[0]][0]
	if img0.Caption != "Wiring diagram" {
		t.Errorf("caption: got %q", img0.Caption)
	}
	if img0.Width != 800 || img0.Height != 600 {
		t.Errorf("dimensions: got %dx%d", img0.Width, img0.Height)
	}
	if string(img0.Data) != "fake-png-data" {
		t.Errorf("data: got %q", string(img0.Data))
	}

	// Get without data
	resultNoData, err := s.GetImagesByChunkIDs(ctx, []int64{chunkIDs[0]}, false)
	if err != nil {
		t.Fatalf("get images without data: %v", err)
	}
	for _, img := range resultNoData[chunkIDs[0]] {
		if img.Data != nil {
			t.Errorf("expected nil Data when includeData=false, got %d bytes", len(img.Data))
		}
		// Metadata should still be present
		if img.MIMEType == "" {
			t.Error("expected non-empty MIMEType even without data")
		}
	}
}

func TestInsertChunkImagesEmpty(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Should be a no-op, not an error.
	if err := s.InsertChunkImages(ctx, nil); err != nil {
		t.Fatalf("insert empty images: %v", err)
	}
}

func TestGetImagesByChunkIDsEmpty(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	result, err := s.GetImagesByChunkIDs(ctx, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil for empty chunk IDs, got %v", result)
	}
}

func TestDeleteDocumentDataCascadesImages(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, _ := s.UpsertDocument(ctx, sampleDoc("/cascade-img.pdf"))
	chunkIDs, _ := s.InsertChunks(ctx, []Chunk{
		{DocumentID: docID, Content: "data", ChunkType: "p", PositionInDoc: 0, TokenCount: 1},
	})
	_ = s.InsertEmbedding(ctx, chunkIDs[0], []float32{1, 0, 0, 0})
	_ = s.InsertChunkImages(ctx, []ChunkImage{
		{ChunkID: chunkIDs[0], DocumentID: docID, MIMEType: "image/png", Width: 10, Height: 10, Data: []byte("x")},
	})

	// Verify image exists
	imgs, _ := s.GetImagesByChunkIDs(ctx, chunkIDs, false)
	if len(imgs[chunkIDs[0]]) != 1 {
		t.Fatal("expected 1 image before delete")
	}

	// Delete data
	if err := s.DeleteDocumentData(ctx, docID); err != nil {
		t.Fatalf("delete data: %v", err)
	}

	// Images should be gone
	imgs2, _ := s.GetImagesByChunkIDs(ctx, chunkIDs, false)
	if len(imgs2) != 0 {
		t.Fatalf("expected 0 images after cascade, got %d", len(imgs2))
	}
}

func TestDeleteDocumentCascadesImages(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	docID, _ := s.UpsertDocument(ctx, sampleDoc("/del-doc-img.pdf"))
	chunkIDs, _ := s.InsertChunks(ctx, []Chunk{
		{DocumentID: docID, Content: "data", ChunkType: "p", PositionInDoc: 0, TokenCount: 1},
	})
	_ = s.InsertEmbedding(ctx, chunkIDs[0], []float32{1, 0, 0, 0})
	_ = s.InsertChunkImages(ctx, []ChunkImage{
		{ChunkID: chunkIDs[0], DocumentID: docID, MIMEType: "image/png", Width: 10, Height: 10, Data: []byte("x")},
	})

	if err := s.DeleteDocument(ctx, docID); err != nil {
		t.Fatalf("delete doc: %v", err)
	}

	// Verify images are gone (chunk IDs no longer valid, but query shouldn't error)
	var count int
	err := s.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM chunk_images").Scan(&count)
	if err != nil {
		t.Fatalf("count images: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 images after doc delete, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// AllEntities / AllRelationships
// ---------------------------------------------------------------------------

func TestAllEntitiesAndRelationships(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	id1, _ := s.UpsertEntity(ctx, Entity{Name: "X", EntityType: "t", Description: "dx"})
	id2, _ := s.UpsertEntity(ctx, Entity{Name: "Y", EntityType: "t", Description: "dy"})

	s.InsertRelationship(ctx, Relationship{
		SourceEntityID: id1,
		TargetEntityID: id2,
		RelationType:   "links_to",
		Weight:         1.0,
		Description:    "X links Y",
	})

	ents, err := s.AllEntities(ctx)
	if err != nil {
		t.Fatalf("all entities: %v", err)
	}
	if len(ents) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(ents))
	}

	rels, err := s.AllRelationships(ctx)
	if err != nil {
		t.Fatalf("all relationships: %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}
	if rels[0].RelationType != "links_to" {
		t.Errorf("relation type: got %q", rels[0].RelationType)
	}
}
