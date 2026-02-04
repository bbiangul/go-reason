package store

import "fmt"

// schemaSQL returns the DDL for all tables. embeddingDim controls the
// vec0 virtual table dimension.
func schemaSQL(embeddingDim int) string {
	return fmt.Sprintf(`
-- Document registry with hash-based change detection
CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    filename TEXT NOT NULL,
    format TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    parse_method TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    metadata JSON,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Hierarchical chunks (parent = section, child = paragraph/clause)
CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY,
    document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    parent_chunk_id INTEGER REFERENCES chunks(id),
    content TEXT NOT NULL,
    chunk_type TEXT NOT NULL,
    heading TEXT,
    page_number INTEGER,
    position_in_doc INTEGER,
    token_count INTEGER,
    metadata JSON,
    content_hash TEXT NOT NULL
);

-- Vector embeddings via sqlite-vec
CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunks USING vec0(
    chunk_id INTEGER PRIMARY KEY,
    embedding float[%d]
);

-- Full-text search via FTS5
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
    content,
    heading,
    content='chunks',
    content_rowid='id',
    tokenize='porter unicode61'
);

-- FTS triggers to keep index in sync
CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN
    INSERT INTO chunks_fts(rowid, content, heading) VALUES (new.id, new.content, new.heading);
END;
CREATE TRIGGER IF NOT EXISTS chunks_ad AFTER DELETE ON chunks BEGIN
    INSERT INTO chunks_fts(chunks_fts, rowid, content, heading) VALUES ('delete', old.id, old.content, old.heading);
END;
CREATE TRIGGER IF NOT EXISTS chunks_au AFTER UPDATE ON chunks BEGIN
    INSERT INTO chunks_fts(chunks_fts, rowid, content, heading) VALUES ('delete', old.id, old.content, old.heading);
    INSERT INTO chunks_fts(chunks_fts, rowid, content, heading) VALUES (new.id, new.content, new.heading);
END;

-- Knowledge graph: entities
CREATE TABLE IF NOT EXISTS entities (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    description TEXT,
    embedding_id INTEGER,
    metadata JSON,
    UNIQUE(name, entity_type)
);

-- Knowledge graph: relationships
CREATE TABLE IF NOT EXISTS relationships (
    id INTEGER PRIMARY KEY,
    source_entity_id INTEGER NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    target_entity_id INTEGER NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL,
    weight REAL DEFAULT 1.0,
    description TEXT,
    source_chunk_id INTEGER REFERENCES chunks(id),
    metadata JSON
);

-- Entity-to-chunk mapping (provenance)
CREATE TABLE IF NOT EXISTS entity_chunks (
    entity_id INTEGER NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    chunk_id INTEGER NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
    PRIMARY KEY (entity_id, chunk_id)
);

-- Community detection results
CREATE TABLE IF NOT EXISTS communities (
    id INTEGER PRIMARY KEY,
    level INTEGER NOT NULL,
    summary TEXT,
    entity_ids JSON NOT NULL
);

-- Query audit log
CREATE TABLE IF NOT EXISTS query_log (
    id INTEGER PRIMARY KEY,
    query TEXT NOT NULL,
    answer TEXT,
    confidence REAL,
    sources JSON,
    retrieval_method TEXT,
    model_used TEXT,
    rounds INTEGER,
    prompt_tokens INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_chunks_document ON chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_chunks_parent ON chunks(parent_chunk_id);
CREATE INDEX IF NOT EXISTS idx_chunks_type ON chunks(chunk_type);
CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(entity_type);
CREATE INDEX IF NOT EXISTS idx_relationships_source ON relationships(source_entity_id);
CREATE INDEX IF NOT EXISTS idx_relationships_target ON relationships(target_entity_id);
CREATE INDEX IF NOT EXISTS idx_relationships_type ON relationships(relation_type);
CREATE INDEX IF NOT EXISTS idx_entity_chunks_chunk ON entity_chunks(chunk_id);
CREATE INDEX IF NOT EXISTS idx_documents_hash ON documents(content_hash);
`, embeddingDim)
}
