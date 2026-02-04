# GoReason Cost Breakdown

Based on the full v5 eval run (140 queries) against the ALTAVision manual (214 pages, 942 chunks).

## Pricing

| Provider | Model | Input | Output |
|----------|-------|-------|--------|
| Groq | openai/gpt-oss-120b | $0.15 / 1M tokens | $0.60 / 1M tokens |
| OpenAI | text-embedding-3-small | $0.01 / 1M tokens | — |

---

## 1. Ingestion Cost (one-time per document)

### 1.1 Embedding Generation

| Metric | Value |
|--------|-------|
| Chunks embedded | 942 |
| Total tokens | ~68,000 |
| Cost | 68,000 × $0.01 / 1M = **$0.0007** |

### 1.2 Graph Extraction (LLM)

Two LLM calls per chunk: entity extraction then relationship extraction.

| Step | Chunks | Input tokens | Output tokens |
|------|--------|--------------|---------------|
| Entity extraction | 517 | ~421,000 | ~119,000 |
| Relationship extraction | 477 | ~293,000 | ~72,000 |
| **Total** | | **~714,000** | **~191,000** |

| Metric | Value |
|--------|-------|
| Input cost | 714,000 × $0.15 / 1M = **$0.107** |
| Output cost | 191,000 × $0.60 / 1M = **$0.115** |
| **Graph extraction total** | **$0.222** |

### 1.3 Ingestion Summary

| Component | Cost |
|-----------|------|
| Embedding (OpenAI) | $0.001 |
| Graph extraction (Groq) | $0.222 |
| **Total ingestion** | **$0.223** |

For a 214-page technical manual. Time: ~5m 47s (33s embedding + 4m 27s graph + misc).

---

## 2. Query Cost

### 2.1 Per-Query Breakdown

Each query involves:
1. **Translation** (1 LLM call) — translate query terms to document language
2. **Embedding** (1 API call) — embed the query for vector search
3. **Reasoning** (1-3 LLM calls) — multi-round reasoning over retrieved chunks

| Component | Avg input tokens | Avg output tokens | Avg cost |
|-----------|-----------------|-------------------|----------|
| Translation | ~200 | ~150 | $0.00012 |
| Query embedding | ~20 | — | $0.0000002 |
| Reasoning (chat LLM) | ~4,200 | ~890 | $0.00117 |
| **Total per query** | **~4,420** | **~1,040** | **$0.00129** |

**Average query cost: ~$0.0013 (0.13 cents)**

### 2.2 By Difficulty Level

| Difficulty | Queries | Avg tokens | Avg input | Avg output | Avg cost | Total cost |
|-----------|---------|------------|-----------|------------|----------|------------|
| Easy | 30 | 3,928 | 3,620 | 307 | $0.00073 | $0.022 |
| Medium | 30 | 4,235 | 3,574 | 661 | $0.00093 | $0.028 |
| Hard | 30 | 5,135 | 4,192 | 943 | $0.00119 | $0.036 |
| Super-Hard | 50 | 6,294 | 4,939 | 1,354 | $0.00155 | $0.078 |
| **All** | **140** | **5,098** | **4,204** | **893** | **$0.00117** | **$0.164** |

Note: Query costs above are for the reasoning LLM only. Adding translation and embedding:

| Component | 140-query total |
|-----------|----------------|
| Reasoning (Groq) | $0.164 |
| Translation (Groq) | $0.016 |
| Query embedding (OpenAI) | $0.00003 |
| **Total eval queries** | **$0.180** |

### 2.3 Sample Queries — Cheapest to Most Expensive

| Query | Difficulty | Tokens | Cost |
|-------|-----------|--------|------|
| "What is the weight of Model C Standard XL?" | Easy | 2,675 | $0.00048 |
| "What are the three SKU states?" | Medium | 2,493 | $0.00055 |
| "What are the exact weights for Model A/B/C?" | Super-Hard | 2,860 | $0.00061 |
| "How many pulses per revolution is the encoder set?" | Easy | 5,526 | $0.00092 |
| "What components are tracked by the Tracker board?" | Medium | 5,849 | $0.00112 |
| "What are all the conditions that void the warranty?" | Hard | 6,960 | $0.00132 |
| "What are all possible failure modes and alarm responses?" | Super-Hard | 22,947 | $0.00682 |

The most expensive query (22,947 tokens) triggered synthesis mode + follow-up retrieval, resulting in 2 reasoning rounds with an expanded chunk window.

### 2.4 Token Distribution

| | Min | Median | Max | Avg |
|---|-----|--------|-----|-----|
| Easy | 2,675 | 3,969 | 5,526 | 3,928 |
| Medium | 2,493 | 4,416 | 5,849 | 4,235 |
| Hard | 2,960 | 5,186 | 6,960 | 5,135 |
| Super-Hard | 2,860 | 5,282 | 22,947 | 6,294 |

---

## 3. Total Cost Summary

### Full Pipeline (ingest + 140 queries)

| Component | Provider | Cost |
|-----------|----------|------|
| Ingestion embedding | OpenAI | $0.001 |
| Graph extraction | Groq | $0.222 |
| Query embedding (140×) | OpenAI | $0.00003 |
| Translation (132 calls) | Groq | $0.016 |
| Reasoning (140 queries) | Groq | $0.164 |
| **Grand total** | | **$0.403** |

### Per-Unit Costs

| Metric | Cost |
|--------|------|
| Ingest 1 document (214 pages) | **$0.22** |
| 1 simple query (easy/medium) | **$0.0008** |
| 1 complex query (hard) | **$0.0012** |
| 1 synthesis query (super-hard) | **$0.0016** |
| 1 worst-case query (follow-up retrieval) | **$0.0068** |
| Average query (all difficulties) | **$0.0013** |
| 1,000 average queries | **$1.30** |
| 10,000 average queries | **$13.00** |

### Cost Structure

| Phase | % of total |
|-------|-----------|
| Ingestion (one-time) | 55% |
| Query reasoning | 41% |
| Translation | 4% |
| Embedding (query + ingest) | <1% |

Ingestion dominates on a per-document basis, but amortizes to near-zero over query volume. At 200+ queries per document, query reasoning becomes the dominant cost.

---

## 4. Comparison: RAG vs Full-Context Window

Baseline: sending the entire document (~120K tokens) in the context window for every query, using Gemini Flash pricing ($0.50/1M input, $3.00/1M output).

### Per-Query Cost

| | Gemini Flash (full context) | GoReason (RAG) | Savings |
|---|---|---|---|
| Input tokens | 120,000 | ~4,400 | 27x fewer |
| Output tokens | ~890 | ~890 | Same |
| Input cost | $0.0600 | $0.00066 | 91x |
| Output cost | $0.0027 | $0.00054 | 5x |
| **Total per query** | **$0.063** | **$0.0013** | **48x cheaper** |

### At Scale

| Volume | Gemini Flash | GoReason | Ratio |
|--------|-------------|----------|-------|
| 1 query | $0.063 | $0.22 (ingest) + $0.001 | 0.3x (ingest overhead) |
| 4 queries | $0.25 | $0.23 | **Break-even** |
| 140 queries | $8.77 | $0.40 | **22x cheaper** |
| 1,000 queries | $62.70 | $1.52 | **41x cheaper** |
| 10,000 queries | $627 | $13.22 | **47x cheaper** |

GoReason breaks even at ~4 queries per document. After that the gap widens because ingestion is one-time while full-context re-uploads the entire document on every query.

### Why the Gap Widens

| Factor | Full Context | GoReason |
|--------|-------------|----------|
| Input per query | 120,000 tokens (full doc) | ~4,200 tokens (25 chunks) |
| Scales with | query count × doc size | query count × chunk window |
| Document re-read | Every query | Never (indexed once) |
| Marginal cost at 10K queries | $627 | $13 |

The fundamental difference: full-context pays O(doc_size) per query. RAG pays O(doc_size) once at ingestion, then O(chunk_window) per query. For a 120K-token document with a 4K-token retrieval window, that's a 30x input reduction on every query.

---

## 5. Notes

- All costs based on Groq pricing for openai/gpt-oss-120b and OpenAI pricing for text-embedding-3-small as of Feb 2026.
- Graph extraction token counts are estimates (no per-call token logging during ingestion). Actual costs may vary ±20%.
- Query reasoning tokens are exact (reported by the LLM provider).
- Translation costs are estimated at ~200 input + ~150 output tokens per call based on prompt size and JSON response.
- Embedding costs are negligible (<1% of total) at current pricing.
- The document tested is a 214-page Spanish technical manual (942 chunks, 1,972 entities, 2,265 relationships).
