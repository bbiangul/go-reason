package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"path/filepath"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	sqlite_vec.Auto()
}

// Document represents a row in the documents table.
type Document struct {
	ID          int64  `json:"id"`
	Path        string `json:"path"`
	Filename    string `json:"filename"`
	Format      string `json:"format"`
	ContentHash string `json:"content_hash"`
	ParseMethod string `json:"parse_method"`
	Status      string `json:"status"`
	Metadata    string `json:"metadata,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Chunk represents a row in the chunks table.
type Chunk struct {
	ID            int64  `json:"id"`
	DocumentID    int64  `json:"document_id"`
	ParentChunkID *int64 `json:"parent_chunk_id,omitempty"`
	Content       string `json:"content"`
	ChunkType     string `json:"chunk_type"`
	Heading       string `json:"heading"`
	PageNumber    int    `json:"page_number"`
	PositionInDoc int    `json:"position_in_doc"`
	TokenCount    int    `json:"token_count"`
	Metadata      string `json:"metadata,omitempty"`
	ContentHash   string `json:"content_hash"`
}

// Entity represents a row in the entities table.
type Entity struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	NameEN      string `json:"name_en"`
	EntityType  string `json:"entity_type"`
	Description string `json:"description"`
	EmbeddingID *int64 `json:"embedding_id,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
}

// Relationship represents a row in the relationships table.
type Relationship struct {
	ID             int64   `json:"id"`
	SourceEntityID int64   `json:"source_entity_id"`
	TargetEntityID int64   `json:"target_entity_id"`
	RelationType   string  `json:"relation_type"`
	Weight         float64 `json:"weight"`
	Description    string  `json:"description"`
	SourceChunkID  *int64  `json:"source_chunk_id,omitempty"`
	Metadata       string  `json:"metadata,omitempty"`
}

// Community represents a row in the communities table.
type Community struct {
	ID        int64  `json:"id"`
	Level     int    `json:"level"`
	Summary   string `json:"summary"`
	EntityIDs string `json:"entity_ids"` // JSON array
}

// QueryLog represents a row in the query_log table.
type QueryLog struct {
	Query            string      `json:"query"`
	Answer           string      `json:"answer"`
	Confidence       float64     `json:"confidence"`
	Sources          interface{} `json:"sources"`
	RetrievalMethod  string      `json:"retrieval_method"`
	ModelUsed        string      `json:"model_used"`
	Rounds           int         `json:"rounds"`
	PromptTokens     int         `json:"prompt_tokens"`
	CompletionTokens int         `json:"completion_tokens"`
	TotalTokens      int         `json:"total_tokens"`
}

// RetrievalResult holds a chunk with its retrieval score and document info.
type RetrievalResult struct {
	ChunkID    int64   `json:"chunk_id"`
	DocumentID int64   `json:"document_id"`
	Content    string  `json:"content"`
	Heading    string  `json:"heading"`
	ChunkType  string  `json:"chunk_type"`
	PageNumber int     `json:"page_number"`
	Filename   string  `json:"filename"`
	Path       string  `json:"path"`
	Score      float64 `json:"score"`
}

// Store wraps the SQLite database for all goreason persistence.
type Store struct {
	db           *sql.DB
	embeddingDim int
}

// New opens (or creates) a SQLite database at the given path and
// initialises the schema including sqlite-vec and FTS5 virtual tables.
func New(dbPath string, embeddingDim int) (*Store, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=30000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	// Create schema
	if _, err := db.Exec(schemaSQL(embeddingDim)); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	// Connection pool settings for SQLite.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)

	s := &Store{db: db, embeddingDim: embeddingDim}

	// Run pending migrations.
	if err := s.Migrate(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB for advanced queries.
func (s *Store) DB() *sql.DB {
	return s.db
}

// EmbeddingDim returns the configured embedding dimension.
func (s *Store) EmbeddingDim() int {
	return s.embeddingDim
}

// --- Document operations ---

// UpsertDocument inserts or updates a document record. Returns the document ID.
func (s *Store) UpsertDocument(ctx context.Context, doc Document) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO documents (path, filename, format, content_hash, parse_method, status, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			filename = excluded.filename,
			format = excluded.format,
			content_hash = excluded.content_hash,
			parse_method = excluded.parse_method,
			status = excluded.status,
			metadata = excluded.metadata,
			updated_at = CURRENT_TIMESTAMP
	`, doc.Path, doc.Filename, doc.Format, doc.ContentHash, doc.ParseMethod, doc.Status, doc.Metadata)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// If UPSERT did an UPDATE, LastInsertId may not reflect the existing row.
	if id == 0 {
		row := s.db.QueryRowContext(ctx, "SELECT id FROM documents WHERE path = ?", doc.Path)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// GetDocumentByPath retrieves a document by its file path.
func (s *Store) GetDocumentByPath(ctx context.Context, path string) (*Document, error) {
	doc := &Document{}
	var metadata sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, path, filename, format, content_hash, parse_method, status, metadata, created_at, updated_at
		FROM documents WHERE path = ?
	`, path).Scan(&doc.ID, &doc.Path, &doc.Filename, &doc.Format,
		&doc.ContentHash, &doc.ParseMethod, &doc.Status,
		&metadata, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	doc.Metadata = metadata.String
	return doc, nil
}

// GetDocument retrieves a document by ID.
func (s *Store) GetDocument(ctx context.Context, id int64) (*Document, error) {
	doc := &Document{}
	var metadata sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, path, filename, format, content_hash, parse_method, status, metadata, created_at, updated_at
		FROM documents WHERE id = ?
	`, id).Scan(&doc.ID, &doc.Path, &doc.Filename, &doc.Format,
		&doc.ContentHash, &doc.ParseMethod, &doc.Status,
		&metadata, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	doc.Metadata = metadata.String
	return doc, nil
}

// ListDocuments returns all documents ordered by creation time.
func (s *Store) ListDocuments(ctx context.Context) ([]Document, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path, filename, format, content_hash, parse_method, status, metadata, created_at, updated_at
		FROM documents ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		var metadata sql.NullString
		if err := rows.Scan(&d.ID, &d.Path, &d.Filename, &d.Format,
			&d.ContentHash, &d.ParseMethod, &d.Status,
			&metadata, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.Metadata = metadata.String
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

// UpdateDocumentStatus updates just the status field.
func (s *Store) UpdateDocumentStatus(ctx context.Context, id int64, status string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE documents SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		status, id)
	return err
}

// UpdateDocumentParseMethod updates just the parse_method field.
func (s *Store) UpdateDocumentParseMethod(ctx context.Context, id int64, method string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE documents SET parse_method = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		method, id)
	return err
}

// DeleteDocument removes a document and cascades to all related data.
func (s *Store) DeleteDocument(ctx context.Context, id int64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		// Delete entity_chunks for entities related to this doc's chunks
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM entity_chunks WHERE chunk_id IN (
				SELECT id FROM chunks WHERE document_id = ?
			)`, id); err != nil {
			return err
		}

		// Delete relationships sourced from this doc's chunks
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM relationships WHERE source_chunk_id IN (
				SELECT id FROM chunks WHERE document_id = ?
			)`, id); err != nil {
			return err
		}

		// Delete vec embeddings
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM vec_chunks WHERE chunk_id IN (
				SELECT id FROM chunks WHERE document_id = ?
			)`, id); err != nil {
			return err
		}

		// Delete chunks (triggers will clean up FTS)
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM chunks WHERE document_id = ?", id); err != nil {
			return err
		}

		// Delete the document
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM documents WHERE id = ?", id); err != nil {
			return err
		}

		return nil
	})
}

// DeleteDocumentData removes all chunks, embeddings, and entity data
// for a document but keeps the document record itself.
func (s *Store) DeleteDocumentData(ctx context.Context, docID int64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM entity_chunks WHERE chunk_id IN (
				SELECT id FROM chunks WHERE document_id = ?
			)`, docID); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			DELETE FROM relationships WHERE source_chunk_id IN (
				SELECT id FROM chunks WHERE document_id = ?
			)`, docID); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			DELETE FROM vec_chunks WHERE chunk_id IN (
				SELECT id FROM chunks WHERE document_id = ?
			)`, docID); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx,
			"DELETE FROM chunks WHERE document_id = ?", docID); err != nil {
			return err
		}

		return nil
	})
}

// --- Chunk operations ---

// InsertChunks inserts a batch of chunks and returns their IDs.
// The chunker assigns temporary position-based IDs; this method remaps
// ParentChunkID values to the real database IDs as chunks are inserted.
func (s *Store) InsertChunks(ctx context.Context, chunks []Chunk) ([]int64, error) {
	ids := make([]int64, len(chunks))

	// Map from temporary position-based ID to real DB ID.
	idMap := make(map[int64]int64, len(chunks))

	err := s.inTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO chunks (document_id, parent_chunk_id, content, chunk_type, heading,
				page_number, position_in_doc, token_count, metadata, content_hash)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for i, c := range chunks {
			hash := sha256.Sum256([]byte(c.Content))
			contentHash := hex.EncodeToString(hash[:])

			// Remap parent_chunk_id from temporary to real DB ID.
			var parentID *int64
			if c.ParentChunkID != nil {
				if realID, ok := idMap[*c.ParentChunkID]; ok {
					parentID = &realID
				}
			}

			res, err := stmt.ExecContext(ctx,
				c.DocumentID, parentID, c.Content, c.ChunkType,
				c.Heading, c.PageNumber, c.PositionInDoc, c.TokenCount,
				c.Metadata, contentHash)
			if err != nil {
				return err
			}
			ids[i], err = res.LastInsertId()
			if err != nil {
				return err
			}
			idMap[c.ID] = ids[i]
		}
		return nil
	})

	return ids, err
}

// GetChunksByDocument returns all chunks for a given document.
func (s *Store) GetChunksByDocument(ctx context.Context, docID int64) ([]Chunk, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, parent_chunk_id, content, chunk_type, heading,
			page_number, position_in_doc, token_count, metadata, content_hash
		FROM chunks WHERE document_id = ? ORDER BY position_in_doc
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		var metadata sql.NullString
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.ParentChunkID, &c.Content,
			&c.ChunkType, &c.Heading, &c.PageNumber, &c.PositionInDoc,
			&c.TokenCount, &metadata, &c.ContentHash); err != nil {
			return nil, err
		}
		c.Metadata = metadata.String
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

// --- Embedding operations ---

// InsertEmbedding stores a vector embedding for a chunk.
func (s *Store) InsertEmbedding(ctx context.Context, chunkID int64, embedding []float32) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO vec_chunks (chunk_id, embedding) VALUES (?, ?)",
		chunkID, serializeFloat32(embedding))
	return err
}

// VectorSearch performs a KNN search returning the top-k nearest chunks.
func (s *Store) VectorSearch(ctx context.Context, queryEmbedding []float32, k int) ([]RetrievalResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT v.chunk_id, v.distance,
			c.content, c.heading, c.chunk_type, c.page_number, c.document_id,
			d.filename, d.path
		FROM vec_chunks v
		JOIN chunks c ON c.id = v.chunk_id
		JOIN documents d ON d.id = c.document_id
		WHERE v.embedding MATCH ? AND k = ?
		ORDER BY v.distance
	`, serializeFloat32(queryEmbedding), k)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RetrievalResult
	for rows.Next() {
		var r RetrievalResult
		var distance float64
		if err := rows.Scan(&r.ChunkID, &distance,
			&r.Content, &r.Heading, &r.ChunkType, &r.PageNumber, &r.DocumentID,
			&r.Filename, &r.Path); err != nil {
			return nil, err
		}
		// Convert distance to similarity score (1 - distance for cosine)
		r.Score = 1.0 - distance
		results = append(results, r)
	}
	return results, rows.Err()
}

// FTSSearch performs a full-text search using FTS5 BM25 ranking.
func (s *Store) FTSSearch(ctx context.Context, query string, limit int) ([]RetrievalResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.rowid, f.rank,
			c.content, c.heading, c.chunk_type, c.page_number, c.document_id,
			d.filename, d.path
		FROM chunks_fts f
		JOIN chunks c ON c.id = f.rowid
		JOIN documents d ON d.id = c.document_id
		WHERE chunks_fts MATCH ?
		ORDER BY f.rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RetrievalResult
	for rows.Next() {
		var r RetrievalResult
		var rank float64
		if err := rows.Scan(&r.ChunkID, &rank,
			&r.Content, &r.Heading, &r.ChunkType, &r.PageNumber, &r.DocumentID,
			&r.Filename, &r.Path); err != nil {
			return nil, err
		}
		// FTS5 rank is negative (lower = better), convert to positive score
		r.Score = -rank
		results = append(results, r)
	}
	return results, rows.Err()
}

// --- Entity operations ---

// UpsertEntity inserts or updates an entity. Returns the entity ID.
func (s *Store) UpsertEntity(ctx context.Context, e Entity) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO entities (name, entity_type, description, name_en, metadata)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name, entity_type) DO UPDATE SET
			description = COALESCE(excluded.description, entities.description),
			name_en = COALESCE(excluded.name_en, entities.name_en),
			metadata = excluded.metadata
	`, e.Name, e.EntityType, e.Description, e.NameEN, e.Metadata)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id == 0 {
		row := s.db.QueryRowContext(ctx,
			"SELECT id FROM entities WHERE name = ? AND entity_type = ?",
			e.Name, e.EntityType)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// UpsertEntityAndLink atomically upserts an entity and links it to a chunk
// in a single transaction, preventing FOREIGN KEY failures from concurrent access.
func (s *Store) UpsertEntityAndLink(ctx context.Context, e Entity, chunkID int64) (int64, error) {
	var id int64
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO entities (name, entity_type, description, name_en, metadata)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(name, entity_type) DO UPDATE SET
				description = COALESCE(excluded.description, entities.description),
				name_en = COALESCE(excluded.name_en, entities.name_en),
				metadata = excluded.metadata
		`, e.Name, e.EntityType, e.Description, e.NameEN, e.Metadata)
		if err != nil {
			return err
		}

		id, err = res.LastInsertId()
		if err != nil {
			return err
		}
		if id == 0 {
			row := tx.QueryRowContext(ctx,
				"SELECT id FROM entities WHERE name = ? AND entity_type = ?",
				e.Name, e.EntityType)
			if err := row.Scan(&id); err != nil {
				return err
			}
		}

		_, err = tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO entity_chunks (entity_id, chunk_id) VALUES (?, ?)",
			id, chunkID)
		return err
	})
	return id, err
}

// LinkEntityChunk creates a provenance link between an entity and a chunk.
func (s *Store) LinkEntityChunk(ctx context.Context, entityID, chunkID int64) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO entity_chunks (entity_id, chunk_id) VALUES (?, ?)",
		entityID, chunkID)
	return err
}

// InsertRelationship creates a relationship between two entities.
func (s *Store) InsertRelationship(ctx context.Context, r Relationship) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO relationships (source_entity_id, target_entity_id, relation_type,
			weight, description, source_chunk_id, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, r.SourceEntityID, r.TargetEntityID, r.RelationType,
		r.Weight, r.Description, r.SourceChunkID, r.Metadata)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetEntitiesByNames returns entities matching any of the given names.
func (s *Store) GetEntitiesByNames(ctx context.Context, names []string) ([]Entity, error) {
	if len(names) == 0 {
		return nil, nil
	}

	query := "SELECT id, name, entity_type, description, COALESCE(name_en, ''), metadata FROM entities WHERE name IN (?" +
		repeatPlaceholders(len(names)-1) + ")"

	args := make([]interface{}, len(names))
	for i, n := range names {
		args[i] = n
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		var metadata sql.NullString
		if err := rows.Scan(&e.ID, &e.Name, &e.EntityType, &e.Description, &e.NameEN, &metadata); err != nil {
			return nil, err
		}
		e.Metadata = metadata.String
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// SearchEntitiesByTerms finds entities whose names contain any of the given
// terms as substrings. This enables graph search to work when query terms are
// single words (e.g. "rejected") and entity names are multi-word phrases
// (e.g. "rechazador de envases"). Results are limited to avoid noise from
// very short or generic terms.
func (s *Store) SearchEntitiesByTerms(ctx context.Context, terms []string, limit int) ([]Entity, error) {
	if len(terms) == 0 {
		return nil, nil
	}
	if limit == 0 {
		limit = 50
	}

	// Build OR conditions: name LIKE '%term1%' OR name LIKE '%term2%' ...
	// Only use terms with length >= 4 to avoid noise from short words.
	var conditions []string
	var args []interface{}
	for _, t := range terms {
		if len(t) < 4 {
			continue
		}
		conditions = append(conditions, "name LIKE ?")
		args = append(args, "%"+t+"%")
	}
	if len(conditions) == 0 {
		return nil, nil
	}

	query := "SELECT id, name, entity_type, description, COALESCE(name_en, ''), metadata FROM entities WHERE " +
		strings.Join(conditions, " OR ") +
		" LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		var metadata sql.NullString
		if err := rows.Scan(&e.ID, &e.Name, &e.EntityType, &e.Description, &e.NameEN, &metadata); err != nil {
			return nil, err
		}
		e.Metadata = metadata.String
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// GraphSearch finds chunks reachable via entity relationships.
func (s *Store) GraphSearch(ctx context.Context, entityIDs []int64, limit int) ([]RetrievalResult, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}

	query := `
		SELECT DISTINCT ec.chunk_id, COALESCE(MAX(r.weight), 0.5),
			c.content, c.heading, c.chunk_type, c.page_number, c.document_id,
			d.filename, d.path
		FROM entity_chunks ec
		LEFT JOIN relationships r ON r.source_entity_id = ec.entity_id OR r.target_entity_id = ec.entity_id
		JOIN chunks c ON c.id = ec.chunk_id
		JOIN documents d ON d.id = c.document_id
		WHERE ec.entity_id IN (?` + repeatPlaceholders(len(entityIDs)-1) + `)
		GROUP BY ec.chunk_id
		ORDER BY COALESCE(MAX(r.weight), 0.5) DESC
		LIMIT ?`

	args := make([]interface{}, 0, len(entityIDs)+1)
	for _, id := range entityIDs {
		args = append(args, id)
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RetrievalResult
	for rows.Next() {
		var r RetrievalResult
		if err := rows.Scan(&r.ChunkID, &r.Score,
			&r.Content, &r.Heading, &r.ChunkType, &r.PageNumber, &r.DocumentID,
			&r.Filename, &r.Path); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// GetRelatedEntities performs a 1-hop expansion from the given seed entity IDs
// via the relationships table, returning entities that are directly connected
// but not already in the seed set. Used by synthesis-mode retrieval to discover
// semantically distant entities (e.g., from "seguridad y normativa" → "ip54").
func (s *Store) GetRelatedEntities(ctx context.Context, entityIDs []int64, limit int) ([]Entity, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}
	if limit == 0 {
		limit = 100
	}

	// Build placeholders for the IN clauses
	ph := "?" + repeatPlaceholders(len(entityIDs)-1)

	query := `
		SELECT DISTINCT e.id, e.name, e.entity_type, e.description, COALESCE(e.name_en, ''), e.metadata
		FROM entities e
		JOIN relationships r ON (e.id = r.target_entity_id OR e.id = r.source_entity_id)
		WHERE (r.source_entity_id IN (` + ph + `) OR r.target_entity_id IN (` + ph + `))
		  AND e.id NOT IN (` + ph + `)
		LIMIT ?`

	// Args: source IN (?...) OR target IN (?...) AND e.id NOT IN (?...) LIMIT ?
	args := make([]interface{}, 0, len(entityIDs)*3+1)
	for _, id := range entityIDs {
		args = append(args, id)
	}
	for _, id := range entityIDs {
		args = append(args, id)
	}
	for _, id := range entityIDs {
		args = append(args, id)
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		var metadata sql.NullString
		if err := rows.Scan(&e.ID, &e.Name, &e.EntityType, &e.Description, &e.NameEN, &metadata); err != nil {
			return nil, err
		}
		e.Metadata = metadata.String
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// --- Community operations ---

// InsertCommunity stores a community detection result.
func (s *Store) InsertCommunity(ctx context.Context, c Community) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO communities (level, summary, entity_ids) VALUES (?, ?, ?)",
		c.Level, c.Summary, c.EntityIDs)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetCommunities returns all communities at a given level.
func (s *Store) GetCommunities(ctx context.Context, level int) ([]Community, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, level, summary, entity_ids FROM communities WHERE level = ?", level)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var communities []Community
	for rows.Next() {
		var c Community
		if err := rows.Scan(&c.ID, &c.Level, &c.Summary, &c.EntityIDs); err != nil {
			return nil, err
		}
		communities = append(communities, c)
	}
	return communities, rows.Err()
}

// ClearCommunities removes all community data.
func (s *Store) ClearCommunities(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM communities")
	return err
}

// --- Query log ---

// LogQuery writes an entry to the query audit log.
func (s *Store) LogQuery(ctx context.Context, q QueryLog) error {
	sourcesJSON, _ := json.Marshal(q.Sources)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO query_log (query, answer, confidence, sources, retrieval_method, model_used, rounds, prompt_tokens, completion_tokens, total_tokens)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, q.Query, q.Answer, q.Confidence, string(sourcesJSON), q.RetrievalMethod, q.ModelUsed, q.Rounds,
		q.PromptTokens, q.CompletionTokens, q.TotalTokens)
	return err
}

// --- Graph data for community detection ---

// AllEntities returns every entity in the database.
func (s *Store) AllEntities(ctx context.Context) ([]Entity, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, entity_type, description, COALESCE(name_en, ''), metadata FROM entities")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		var metadata sql.NullString
		if err := rows.Scan(&e.ID, &e.Name, &e.EntityType, &e.Description, &e.NameEN, &metadata); err != nil {
			return nil, err
		}
		e.Metadata = metadata.String
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// AllRelationships returns every relationship in the database.
func (s *Store) AllRelationships(ctx context.Context) ([]Relationship, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_entity_id, target_entity_id, relation_type, weight, description
		FROM relationships
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []Relationship
	for rows.Next() {
		var r Relationship
		var desc sql.NullString
		if err := rows.Scan(&r.ID, &r.SourceEntityID, &r.TargetEntityID,
			&r.RelationType, &r.Weight, &desc); err != nil {
			return nil, err
		}
		r.Description = desc.String
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

// --- Multi-language support ---

// UpdateDocumentLanguage sets the detected language for a document.
func (s *Store) UpdateDocumentLanguage(ctx context.Context, docID int64, language string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE documents SET language = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		language, docID)
	return err
}

// GetCorpusLanguages returns the distinct non-null languages across all documents.
func (s *Store) GetCorpusLanguages(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT DISTINCT language FROM documents WHERE language IS NOT NULL AND language != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var langs []string
	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err != nil {
			return nil, err
		}
		langs = append(langs, lang)
	}
	return langs, rows.Err()
}

// SearchEntitiesByNameEN finds entities whose English canonical name contains
// any of the given terms as substrings. Same pattern as SearchEntitiesByTerms
// but operates on the name_en column for cross-language entity matching.
func (s *Store) SearchEntitiesByNameEN(ctx context.Context, terms []string, limit int) ([]Entity, error) {
	if len(terms) == 0 {
		return nil, nil
	}
	if limit == 0 {
		limit = 50
	}

	var conditions []string
	var args []interface{}
	for _, t := range terms {
		if len(t) < 4 {
			continue
		}
		conditions = append(conditions, "name_en LIKE ?")
		args = append(args, "%"+t+"%")
	}
	if len(conditions) == 0 {
		return nil, nil
	}

	query := "SELECT id, name, entity_type, description, COALESCE(name_en, ''), metadata FROM entities WHERE name_en IS NOT NULL AND (" +
		strings.Join(conditions, " OR ") +
		") LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		var metadata sql.NullString
		if err := rows.Scan(&e.ID, &e.Name, &e.EntityType, &e.Description, &e.NameEN, &metadata); err != nil {
			return nil, err
		}
		e.Metadata = metadata.String
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// --- Diagnostic helpers (used by eval ground-truth checks) ---

// ChunkMatch holds the result of a content substring search.
type ChunkMatch struct {
	ChunkID    int64  `json:"chunk_id"`
	Heading    string `json:"heading"`
	PageNumber int    `json:"page_number"`
}

// SearchChunksByContent searches all chunks for a case-insensitive substring match.
func (s *Store) SearchChunksByContent(ctx context.Context, substring string) ([]ChunkMatch, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, heading, page_number FROM chunks
		WHERE LOWER(content) LIKE '%' || LOWER(?) || '%'
	`, substring)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []ChunkMatch
	for rows.Next() {
		var m ChunkMatch
		if err := rows.Scan(&m.ChunkID, &m.Heading, &m.PageNumber); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// ChunkHasEmbedding checks if a specific chunk has a vector embedding.
func (s *Store) ChunkHasEmbedding(ctx context.Context, chunkID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM vec_chunks WHERE chunk_id = ?", chunkID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DBStats holds counts of key database objects.
type DBStats struct {
	Chunks        int `json:"chunks"`
	Embeddings    int `json:"embeddings"`
	Entities      int `json:"entities"`
	Relationships int `json:"relationships"`
	Communities   int `json:"communities"`
	Documents     int `json:"documents"`
}

// DBStats returns counts of chunks, embeddings, entities, relationships, communities, and documents.
func (s *Store) DBStats(ctx context.Context) (*DBStats, error) {
	stats := &DBStats{}
	queries := []struct {
		query string
		dest  *int
	}{
		{"SELECT COUNT(*) FROM chunks", &stats.Chunks},
		{"SELECT COUNT(*) FROM vec_chunks", &stats.Embeddings},
		{"SELECT COUNT(*) FROM entities", &stats.Entities},
		{"SELECT COUNT(*) FROM relationships", &stats.Relationships},
		{"SELECT COUNT(*) FROM communities", &stats.Communities},
		{"SELECT COUNT(*) FROM documents", &stats.Documents},
	}
	for _, q := range queries {
		if err := s.db.QueryRowContext(ctx, q.query).Scan(q.dest); err != nil {
			return nil, fmt.Errorf("counting %s: %w", q.query, err)
		}
	}
	return stats, nil
}

// SampleChunks returns up to n chunks sampled from the database.
// Used for language detection and other heuristics.
func (s *Store) SampleChunks(ctx context.Context, n int) ([]Chunk, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, parent_chunk_id, content, chunk_type, heading,
			page_number, position_in_doc, token_count, metadata, content_hash
		FROM chunks ORDER BY RANDOM() LIMIT ?
	`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		var metadata sql.NullString
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.ParentChunkID, &c.Content,
			&c.ChunkType, &c.Heading, &c.PageNumber, &c.PositionInDoc,
			&c.TokenCount, &metadata, &c.ContentHash); err != nil {
			return nil, err
		}
		c.Metadata = metadata.String
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

// --- helpers ---

func (s *Store) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func repeatPlaceholders(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += ", ?"
	}
	return s
}

// serializeFloat32 converts a float32 slice to little-endian bytes for sqlite-vec.
func serializeFloat32(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}
