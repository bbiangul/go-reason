# GoReason

A production-ready Graph RAG engine written in Go. Combines knowledge graph extraction, vector embeddings, and full-text search with multi-round reasoning to answer questions about your documents.

```
PDF/DOCX/XLSX/PPTX
        |
    [ Parser ]
        |
    [ Chunker ]  -->  1024 tokens, 128 overlap
        |
   +---------+-----------+
   |         |           |
[Embed]  [Graph LLM]  [FTS5 Index]
   |         |           |
sqlite-vec  entities   Porter stemmer
            relations
            communities
```

## Features

- **Hybrid Retrieval** -- Vector search + FTS5 full-text + knowledge graph, fused with Reciprocal Rank Fusion (RRF)
- **Multi-Round Reasoning** -- Up to 3 rounds of answer generation, validation, and refinement
- **Knowledge Graph** -- Automated entity/relationship extraction with community detection
- **Multi-Step Extraction** -- 2 focused LLM calls per chunk (entities, then relationships) optimized for 7B models
- **Regex Pre-Extraction** -- Detects part numbers, standards, IPs, voltages before LLM to improve accuracy
- **Identifier-Aware Routing** -- Boosts FTS weight when queries contain structured identifiers
- **7 LLM Providers** -- Ollama, OpenAI, Groq, OpenRouter, xAI, LM Studio, any OpenAI-compatible endpoint
- **4 Document Formats** -- PDF, DOCX, XLSX, PPTX (+ LlamaParse integration)
- **SQLite Everything** -- Single-file database with sqlite-vec, FTS5, knowledge graph, audit log
- **Production Middleware** -- Auth, CORS, panic recovery, graceful shutdown, structured logging
- **Built-in Evaluation** -- 140-question benchmark suite across 4 difficulty levels

## Quick Start

### Prerequisites

- Go 1.25+
- CGO enabled (required for SQLite)
- [Ollama](https://ollama.com) running locally (or any supported LLM provider)

### Install and Run

```bash
# Clone
git clone https://github.com/bbiangul/go-reason.git
cd go-reason

# Pull models (if using Ollama)
ollama pull llama3.1:8b
ollama pull nomic-embed-text

# Build and run the server
CGO_ENABLED=1 go build -tags sqlite_fts5 -o goreason-server ./cmd/server
./goreason-server
```

The server starts on `:8080` with default config (Ollama on localhost).

### Ingest a Document

```bash
# Upload a file
curl -X POST http://localhost:8080/ingest \
  -F "file=@/path/to/document.pdf"

# Or provide a path
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d '{"path": "/path/to/document.pdf"}'
```

### Ask a Question

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"question": "What is the operating temperature range?"}'
```

Response:
```json
{
  "text": "The operating temperature range is 5C to 40C...",
  "confidence": 0.95,
  "sources": [
    {"chunk_id": 42, "filename": "manual.pdf", "page_number": 28, "score": 0.87}
  ],
  "reasoning": [
    {"round": 1, "action": "initial_answer", "chunks_used": 5},
    {"round": 2, "action": "validation", "output": "Citations verified"}
  ],
  "rounds": 2,
  "total_tokens": 2450
}
```

## Configuration

### JSON Config File

```bash
./goreason-server -config config.json
```

```json
{
  "db_path": "/data/goreason.db",
  "chat": {
    "provider": "groq",
    "model": "llama-3.3-70b-versatile",
    "api_key": "gsk_..."
  },
  "embedding": {
    "provider": "openai",
    "model": "text-embedding-3-small",
    "api_key": "sk-..."
  },
  "embedding_dim": 1536,
  "weight_vector": 1.0,
  "weight_fts": 1.0,
  "weight_graph": 0.5,
  "max_chunk_tokens": 1024,
  "chunk_overlap": 128,
  "skip_graph": false,
  "graph_concurrency": 8,
  "max_rounds": 3,
  "confidence_threshold": 0.7
}
```

### Environment Variables

All config fields can be overridden via environment variables:

| Variable | Description |
|----------|-------------|
| `GOREASON_DB_PATH` | SQLite database path |
| `GOREASON_CHAT_PROVIDER` | Chat provider name |
| `GOREASON_CHAT_MODEL` | Chat model name |
| `GOREASON_CHAT_BASE_URL` | Chat provider URL |
| `GOREASON_CHAT_API_KEY` | Chat provider API key |
| `GOREASON_EMBED_PROVIDER` | Embedding provider name |
| `GOREASON_EMBED_MODEL` | Embedding model name |
| `GOREASON_EMBED_BASE_URL` | Embedding provider URL |
| `GOREASON_EMBED_API_KEY` | Embedding provider API key |
| `GOREASON_API_KEY` | Server authentication key (Bearer token) |
| `GOREASON_CORS_ORIGINS` | Allowed CORS origins (comma-separated) |
| `OPENAI_API_KEY` | Fallback for OpenAI provider |
| `GROQ_API_KEY` | Fallback for Groq provider |

### Default Config

When no config is provided, GoReason uses Ollama on localhost:

| Setting | Default |
|---------|---------|
| Chat model | `llama3.1:8b` via Ollama |
| Embedding model | `nomic-embed-text` via Ollama (768 dim) |
| Chunk size | 1024 tokens, 128 overlap |
| Retrieval weights | Vector: 1.0, FTS: 1.0, Graph: 0.5 |
| Reasoning | 3 rounds max, 0.7 confidence threshold |
| Graph concurrency | 8 parallel LLM calls |
| Database | `~/.goreason/goreason.db` |

## LLM Providers

GoReason supports 7 providers through a unified OpenAI-compatible interface:

| Provider | Name | Default URL | Default Model | Best For |
|----------|------|-------------|---------------|----------|
| **Ollama** | `ollama` | `http://localhost:11434` | -- | Local inference, free |
| **OpenAI** | `openai` | `https://api.openai.com` | `text-embedding-3-small` | Embeddings |
| **Groq** | `groq` | `https://api.groq.com/openai` | `llama-3.3-70b-versatile` | Fast chat inference |
| **OpenRouter** | `openrouter` | `https://openrouter.ai/api` | -- | Access to 200+ models |
| **xAI** | `xai` | `https://api.x.ai` | -- | Grok models |
| **LM Studio** | `lmstudio` | `http://localhost:1234` | -- | Local inference |
| **Custom** | `custom` | (user-specified) | -- | Any OpenAI-compatible API |

### OpenAI Embedding Models

| Model | Dimensions | Cost per 1M tokens |
|-------|-----------|-------------------|
| `text-embedding-3-small` | 1536 | $0.02 |
| `text-embedding-3-large` | 3072 | $0.13 |
| `text-embedding-ada-002` | 1536 | $0.10 |

### Recommended Configurations

**Local (free, privacy-first):**
```json
{
  "chat": {"provider": "ollama", "model": "llama3.1:8b"},
  "embedding": {"provider": "ollama", "model": "nomic-embed-text"},
  "embedding_dim": 768
}
```

**Balanced (fast + cheap embeddings):**
```json
{
  "chat": {"provider": "groq", "model": "llama-3.3-70b-versatile"},
  "embedding": {"provider": "openai", "model": "text-embedding-3-small"},
  "embedding_dim": 1536
}
```

**High accuracy (cloud):**
```json
{
  "chat": {"provider": "openrouter", "model": "qwen/qwen3-30b-a3b"},
  "embedding": {"provider": "openai", "model": "text-embedding-3-large"},
  "embedding_dim": 3072
}
```

## API Reference

### `POST /ingest`

Ingest a document into the system.

**Multipart upload:**
```bash
curl -X POST http://localhost:8080/ingest -F "file=@document.pdf"
```

**JSON path:**
```bash
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d '{"path": "/path/to/file.pdf", "options": {"force": "true"}}'
```

Options: `force` (re-parse even if hash unchanged), `parse_method` (override parser selection).

Response: `{"document_id": 1, "filename": "document.pdf"}`

### `POST /query`

Ask a question about ingested documents.

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "question": "What voltage does the equipment operate at?",
    "max_results": 20,
    "max_rounds": 3,
    "weight_vector": 1.0,
    "weight_fts": 1.0,
    "weight_graph": 0.5
  }'
```

### `POST /update`

Re-check a document and re-ingest if changed.

```bash
curl -X POST http://localhost:8080/update \
  -H "Content-Type: application/json" \
  -d '{"path": "/path/to/file.pdf"}'
```

### `POST /update-all`

Check all ingested documents for changes.

```bash
curl -X POST http://localhost:8080/update-all
```

### `DELETE /documents/{id}`

Remove a document and all associated data.

```bash
curl -X DELETE http://localhost:8080/documents/1
```

### `GET /documents`

List all ingested documents.

```bash
curl http://localhost:8080/documents
```

### `GET /health`

Health check endpoint.

```bash
curl http://localhost:8080/health
```

## Architecture

### Ingestion Pipeline

```
Document
  -> Format detection (PDF/DOCX/XLSX/PPTX)
  -> Parser (native or LlamaParse)
  -> Chunker (1024 tokens, 128 overlap, hierarchical sections)
  -> Parallel embedding generation (batches of 32)
  -> Knowledge graph extraction (2 LLM calls per chunk, 8 concurrent)
     1. Entity extraction (with regex pre-extracted hints)
     2. Relationship extraction (given known entities)
  -> Community detection + summarization
  -> Content hash stored for change detection
```

### Query Pipeline

```
Question
  -> Identifier detection (boost FTS if part numbers/standards found)
  -> Parallel hybrid retrieval:
     1. Vector search (sqlite-vec cosine similarity)
     2. FTS5 search (Porter stemmer, Unicode)
     3. Graph search (entity lookup + traversal)
  -> RRF fusion (k=60, configurable weights)
  -> Multi-round reasoning:
     Round 1: Initial answer from retrieved chunks
     Round 2: Validate citations, identify gaps
     Round 3: Refine if confidence < threshold
  -> Audit logging (query, answer, tokens, sources)
```

### Knowledge Graph

Entities and relationships are extracted from each chunk using a multi-step pipeline optimized for 7B-class models:

**Entity types:** `person`, `organization`, `standard`, `clause`, `concept`, `term`, `regulation`

**Relation types:** `references`, `defines`, `amends`, `requires`, `contradicts`, `supersedes`

Regex pre-extraction detects structured identifiers (part numbers, standards, IPs, voltages, measurements) and feeds them as hints to the LLM, reducing missed entities.

### Database Schema

Single SQLite file with:

| Table | Purpose |
|-------|---------|
| `documents` | Document registry with SHA-256 hash change detection |
| `chunks` | Hierarchical chunks (parent-child relationships) |
| `vec_chunks` | Vector embeddings (sqlite-vec virtual table) |
| `chunks_fts` | Full-text search index (FTS5 with triggers) |
| `entities` | Knowledge graph nodes |
| `relationships` | Knowledge graph edges with weights |
| `entity_chunks` | Entity-to-chunk provenance mapping |
| `communities` | Community detection results |
| `query_log` | Audit log with token usage tracking |
| `schema_version` | Migration tracking |

## Docker

### Docker Compose (with Ollama)

```bash
docker compose up -d
```

This starts:
- **Ollama** on port 11434 (GPU-enabled if available)
- **GoReason** on port 8080

### Standalone Docker Build

```bash
docker build -t goreason .
docker run -p 8080:8080 \
  -v goreason_data:/data \
  -e GOREASON_CHAT_PROVIDER=groq \
  -e GROQ_API_KEY=gsk_... \
  -e GOREASON_EMBED_PROVIDER=openai \
  -e OPENAI_API_KEY=sk-... \
  goreason
```

## Evaluation

GoReason includes a built-in evaluation framework with 140 questions across 4 difficulty levels, tested against an industrial technical manual (ALTAVision AV-FM, 214 pages, Spanish).

### Run Evaluation

```bash
# Using OpenRouter for chat + Ollama for embeddings
CGO_ENABLED=1 go run -tags sqlite_fts5 ./cmd/eval \
  --pdf /path/to/manual.pdf \
  --chat-provider openrouter \
  --chat-model qwen/qwen3-30b-a3b \
  --embed-provider ollama \
  --embed-model nomic-embed-text \
  --difficulty easy \
  --output eval-report.json

# Using Groq for chat + OpenAI for embeddings
CGO_ENABLED=1 go run -tags sqlite_fts5 ./cmd/eval \
  --pdf /path/to/manual.pdf \
  --chat-provider groq \
  --chat-model llama-3.3-70b-versatile \
  --embed-provider openai \
  --embed-model text-embedding-3-small \
  --embed-dim 1536 \
  --difficulty easy
```

### Difficulty Levels

| Level | Tests | Description |
|-------|-------|-------------|
| Easy | 30 | Single-fact lookup (specs, part numbers, safety) |
| Medium | 30 | Multi-fact retrieval (GUI steps, standards, features) |
| Hard | 30 | Multi-hop reasoning (system relationships, data flow) |
| Super-Hard | 50 | Synthesis & inference (design rationale, troubleshooting) |

### v1 Results (Easy)

| Metric | Score |
|--------|-------|
| **Pass Rate** | **93.3%** (28/30) |
| Faithfulness | 0.993 |
| Accuracy | 0.917 |
| Confidence | 0.948 |
| Citation Quality | 0.700 |

Config: Qwen3-30B-A3B (OpenRouter) + nomic-embed-text (Ollama, 768d). Full results in `evals/v1.md`.

## Project Structure

```
goreason/
  config.go          # Configuration types and defaults
  goreason.go        # Engine interface and implementation
  errors.go          # Sentinel errors

  llm/               # LLM provider abstractions
    provider.go      # Interface + factory
    openai_compat.go # Shared OpenAI-compatible client (retry, timeout)
    ollama.go        # Ollama (native embed endpoint)
    openai.go        # OpenAI
    groq.go          # Groq
    openrouter.go    # OpenRouter
    xai.go           # xAI (Grok)
    lmstudio.go      # LM Studio

  parser/            # Document parsing
    parser.go        # Interface + types
    registry.go      # Format router
    pdf.go           # Native PDF parser
    pdf_vision.go    # Vision-based PDF parsing
    docx.go          # DOCX parser
    xlsx.go          # XLSX parser
    pptx.go          # PPTX parser
    complexity.go    # Document complexity analysis
    llamaparse.go    # LlamaParse integration

  chunker/           # Document chunking
    chunker.go       # Token-aware chunking with overlap
    engineering.go   # Engineering document heuristics
    legal.go         # Legal document heuristics
    structure.go     # Document structure analysis

  graph/             # Knowledge graph
    builder.go       # Multi-step extraction pipeline
    entity.go        # Entity/relationship types
    community.go     # Community detection + summarization
    traversal.go     # Graph traversal for retrieval

  retrieval/         # Hybrid retrieval
    retrieval.go     # Vector + FTS5 + Graph search
    rrf.go           # Reciprocal Rank Fusion
    translations.go  # Multi-language query support
    helpers.go       # Shared utilities

  reasoning/         # Multi-round reasoning
    reasoning.go     # Answer, validate, refine pipeline
    validator.go     # Answer validation
    confidence.go    # Confidence scoring
    citation.go      # Citation extraction

  store/             # SQLite persistence
    store.go         # Database operations
    schema.go        # Schema definition
    migrations.go    # Schema migrations

  eval/              # Evaluation framework
    evaluator.go     # Test runner + scoring
    dataset.go       # Test case types
    metrics.go       # Evaluation metrics
    altavision_dataset.go  # 140-question benchmark

  cmd/
    server/          # HTTP server
      main.go        # Server entry point
      handlers.go    # API handlers
      middleware.go   # Auth, CORS, recovery, logging
    eval/            # Evaluation CLI
      main.go        # Eval entry point

  evals/             # Evaluation reports

  Dockerfile         # Multi-stage production build
  docker-compose.yml # Ollama + GoReason stack
  .github/workflows/ # CI/CD pipeline
```

## Development

### Build

```bash
CGO_ENABLED=1 go build -tags sqlite_fts5 -o goreason-server ./cmd/server
```

### Test

```bash
CGO_ENABLED=1 go test -tags sqlite_fts5 ./...
```

### Lint

```bash
CGO_ENABLED=1 go vet -tags sqlite_fts5 ./...
```

### Build Tags

- `sqlite_fts5` -- Required. Enables FTS5 full-text search in SQLite.

## Go Module

```
require (
    github.com/asg017/sqlite-vec-go-bindings v0.1.6
    github.com/ledongthuc/pdf v0.0.0-20250511090121
    github.com/mattn/go-sqlite3 v1.14.33
    github.com/xuri/excelize/v2 v2.10.0
)
```

No external web frameworks. The HTTP server uses only the standard library (`net/http`).
