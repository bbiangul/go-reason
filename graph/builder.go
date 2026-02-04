package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/brunobiangulo/goreason/llm"
	"github.com/brunobiangulo/goreason/store"
)

// estimateTokens approximates token count using a word-based heuristic.
func estimateTokens(text string) int {
	words := len(strings.Fields(text))
	return int(math.Ceil(float64(words) * 1.3))
}

// entityExtractionPrompt is a focused prompt that asks the LLM to extract
// only entities (nouns) from the text. This is a simpler, atomic task
// optimised for 7B-class models.
const entityExtractionPrompt = `You are an entity extraction engine for technical and industrial documents.
Given the following text chunk, extract all entities (nouns: things, standards, parts, people, organisations, concepts).

ENTITY TYPES (use exactly these values):
- person       : a named individual
- organization : a company, body, committee, or institution
- standard     : a published standard (e.g. ISO 9001, EN 1366-1, IEC 61850)
- clause       : a specific clause, section, or article within a standard or regulation
- concept      : an abstract idea, principle, or methodology
- term         : a defined technical term, abbreviation, part number, model number, or identifier
- regulation   : a law, directive, or regulatory framework

Return a JSON object with exactly one key:
  "entities" : array of {"name": string, "type": string, "description": string}

Rules:
- Entity names must be normalised to lowercase.
- Only include entities clearly supported by the text.
- If there are none, return an empty array.
- Do NOT include any text outside the JSON object.

EXAMPLES:

Input: "The AV-FM fire damper complies with EN 1366-2 and is rated for 120VAC operation. Part number E1375 Rev G02."
Output:
{"entities": [{"name": "av-fm", "type": "term", "description": "Fire damper model"}, {"name": "en 1366-2", "type": "standard", "description": "Fire resistance test standard for dampers"}, {"name": "e1375", "type": "term", "description": "Part number for the fire damper"}, {"name": "rev g02", "type": "term", "description": "Revision code G02"}, {"name": "120vac", "type": "term", "description": "Operating voltage specification"}, {"name": "fire damper", "type": "concept", "description": "A device to prevent fire spread through ducts"}]}

Input: "ISO 9001 clause 7.1 requires organisations to determine the resources needed for quality management."
Output:
{"entities": [{"name": "iso 9001", "type": "standard", "description": "Quality management systems standard"}, {"name": "clause 7.1", "type": "clause", "description": "Clause on resource determination in ISO 9001"}, {"name": "quality management", "type": "concept", "description": "Systematic management of quality processes"}]}

Input: "MIL-STD-810 specifies environmental testing at 75 PSIG and 70 dB noise level. Contact John Smith at Belimo Corp."
Output:
{"entities": [{"name": "mil-std-810", "type": "standard", "description": "Military standard for environmental testing"}, {"name": "75 psig", "type": "term", "description": "Pressure specification"}, {"name": "70 db", "type": "term", "description": "Noise level measurement"}, {"name": "john smith", "type": "person", "description": "Contact person"}, {"name": "belimo corp", "type": "organization", "description": "Corporation mentioned in context"}]}

%s
TEXT:
%s`

// relationshipExtractionPrompt is a focused prompt that, given the already-
// extracted entities, asks the LLM to find only relationships (verbs) between
// them. This second atomic call is simpler because the entity set is fixed.
const relationshipExtractionPrompt = `You are a relationship extraction engine for technical and industrial documents.
Given the text and a list of known entities, extract all relationships (verbs connecting entities).

KNOWN ENTITIES:
%s

RELATION TYPES (use exactly these values):
- references   : source mentions or cites target
- defines      : source provides the definition of target
- amends       : source modifies or updates target
- requires     : source mandates or depends on target
- contradicts  : source conflicts with target
- supersedes   : source replaces target

Return a JSON object with exactly one key:
  "relationships" : array of {"source": string, "target": string, "relation_type": string, "description": string, "weight": number}

Rules:
- Source and target must be entity names from the KNOWN ENTITIES list above (lowercase).
- Weight is a float between 0.0 and 1.0 indicating confidence.
- Only include relationships clearly supported by the text.
- If there are none, return an empty array.
- Do NOT include any text outside the JSON object.

EXAMPLES:

Input entities: ["av-fm", "en 1366-2", "e1375"]
Input text: "The AV-FM fire damper complies with EN 1366-2. Part number E1375."
Output:
{"relationships": [{"source": "av-fm", "target": "en 1366-2", "relation_type": "references", "description": "AV-FM complies with EN 1366-2", "weight": 0.95}, {"source": "e1375", "target": "av-fm", "relation_type": "defines", "description": "E1375 is the part number for AV-FM", "weight": 0.9}]}

Input entities: ["iso 9001", "clause 7.1", "quality management"]
Input text: "ISO 9001 clause 7.1 requires organisations to determine the resources needed for quality management."
Output:
{"relationships": [{"source": "iso 9001", "target": "clause 7.1", "relation_type": "defines", "description": "ISO 9001 contains clause 7.1", "weight": 0.95}, {"source": "clause 7.1", "target": "quality management", "relation_type": "requires", "description": "Clause 7.1 requires resources for quality management", "weight": 0.9}]}

Input entities: ["mil-std-810", "mil-std-461"]
Input text: "MIL-STD-810 has been superseded by MIL-STD-461 for electromagnetic testing."
Output:
{"relationships": [{"source": "mil-std-461", "target": "mil-std-810", "relation_type": "supersedes", "description": "MIL-STD-461 replaces MIL-STD-810 for EM testing", "weight": 0.85}]}

TEXT:
%s`

// defaultConcurrency is the default semaphore size for parallel chunk processing.
const defaultConcurrency = 16

// minChunkTokens skips chunks below this threshold (headers, TOC lines, etc.)
const minChunkTokens = 30

// perChunkTimeout caps how long a single chunk extraction can take.
const perChunkTimeout = 90 * time.Second

// ---------------------------------------------------------------------------
// Regex patterns for pre-extracting technical identifiers from text.
// These are fed as hints to the entity extraction prompt so the LLM does not
// miss structured data that 7B models tend to overlook.
// ---------------------------------------------------------------------------
var (
	// Part numbers: E1375, E-1306, PN: XXXXX, PN:XXXXX, P/N XXXXX
	rePartNumber = regexp.MustCompile(`(?i)(?:PN[:\s]*|P/N[:\s]*)?[A-Z]{1,3}[-]?\d{3,6}`)
	// Revision codes: Rev, RevG02, Rev2, Rev.A, Rev 2
	reRevision = regexp.MustCompile(`(?i)Rev\.?\s*[A-Z0-9]{1,5}`)
	// Standards: ISO XXXXX, EN XXXXX, IEC XXXXX, MIL-STD-XXX, ASTM DXXX, IEEE XXX
	reStandard = regexp.MustCompile(`(?i)(?:ISO|EN|IEC|MIL-STD|ASTM|IEEE|NIST|AS|BS)\s*[-]?\s*\d[\w.-]*`)
	// IP addresses: XXX.XXX.XXX.XXX
	reIPAddress = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	// Model numbers: AV-FM, AV-FF, AV-L (2-4 uppercase letters, dash, 1-4 uppercase letters)
	reModelNumber = regexp.MustCompile(`\b[A-Z]{2,4}-[A-Z]{1,4}\b`)
	// Voltage/current specs: 120VAC, 24VDC, 5Vdc, 3.3V
	reVoltage = regexp.MustCompile(`(?i)\d+(?:\.\d+)?\s*[Vv](?:AC|DC|ac|dc)?\b`)
	// Measurements with units: 75 PSIG, 70 dB, 28 mm, 512 tokens, 100 kPa
	reMeasurement = regexp.MustCompile(`\b\d+(?:\.\d+)?\s*(?:PSIG|psig|dB|db|mm|cm|m|kg|lb|kPa|MPa|Hz|kHz|MHz|GHz|tokens?|°[CF])\b`)
)

// preExtractIdentifiers uses regex to find technical identifiers in text.
// These are fed as hints to the entity extraction prompt so the LLM does not
// miss structured data that 7B models tend to overlook.
func preExtractIdentifiers(text string) []string {
	seen := make(map[string]bool)
	var identifiers []string

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		key := strings.ToLower(s)
		if !seen[key] {
			seen[key] = true
			identifiers = append(identifiers, s)
		}
	}

	patterns := []*regexp.Regexp{
		reStandard,
		rePartNumber,
		reRevision,
		reIPAddress,
		reModelNumber,
		reVoltage,
		reMeasurement,
	}

	for _, p := range patterns {
		for _, m := range p.FindAllString(text, -1) {
			add(m)
		}
	}

	return identifiers
}

// Builder constructs the knowledge graph from document chunks.
type Builder struct {
	store       *store.Store
	chat        llm.Provider
	embed       llm.Provider
	concurrency int
}

// NewBuilder creates a new graph builder.
func NewBuilder(s *store.Store, chat, embed llm.Provider, concurrency int) *Builder {
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}
	return &Builder{
		store:       s,
		chat:        chat,
		embed:       embed,
		concurrency: concurrency,
	}
}

// Build extracts entities and relationships from chunks and stores them.
// chunks and chunkIDs correspond by index.
func (b *Builder) Build(ctx context.Context, docID int64, chunks []store.Chunk, chunkIDs []int64) error {
	if len(chunks) != len(chunkIDs) {
		return fmt.Errorf("graph.Build: chunks and chunkIDs length mismatch (%d vs %d)", len(chunks), len(chunkIDs))
	}

	// Filter out trivial chunks (headers, TOC entries, etc.)
	type indexedChunk struct {
		chunk   store.Chunk
		chunkID int64
	}
	var eligible []indexedChunk
	for i := range chunks {
		if estimateTokens(chunks[i].Content) < minChunkTokens {
			slog.Debug("graph: skipping trivial chunk", "chunk_id", chunkIDs[i],
				"tokens", estimateTokens(chunks[i].Content))
			continue
		}
		eligible = append(eligible, indexedChunk{chunks[i], chunkIDs[i]})
	}

	if len(eligible) == 0 {
		return nil
	}

	slog.Info("graph: processing chunks", "total", len(chunks), "eligible", len(eligible),
		"skipped", len(chunks)-len(eligible), "concurrency", b.concurrency)

	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		sem       = make(chan struct{}, b.concurrency)
		errs      []string
		completed int
		buildStart = time.Now()
	)

	total := len(eligible)

	for _, ic := range eligible {
		wg.Add(1)
		go func(chunk store.Chunk, chunkID int64) {
			defer wg.Done()

			// Acquire semaphore slot.
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				errs = append(errs, fmt.Sprintf("chunk %d: %v", chunkID, ctx.Err()))
				mu.Unlock()
				return
			}

			// Per-chunk timeout to avoid hanging on slow LLM responses.
			chunkCtx, cancel := context.WithTimeout(ctx, perChunkTimeout)
			defer cancel()

			chunkStart := time.Now()
			if err := b.processChunk(chunkCtx, chunk, chunkID); err != nil {
				slog.Warn("graph: chunk failed",
					"chunk_id", chunkID, "error", err,
					"elapsed", time.Since(chunkStart).Round(time.Millisecond))
				mu.Lock()
				errs = append(errs, fmt.Sprintf("chunk %d: %v", chunkID, err))
				completed++
				mu.Unlock()
			} else {
				mu.Lock()
				completed++
				n := completed
				mu.Unlock()
				slog.Info("graph: chunk processed",
					"progress", fmt.Sprintf("%d/%d", n, total),
					"chunk_id", chunkID,
					"elapsed", time.Since(chunkStart).Round(time.Millisecond),
					"total_elapsed", time.Since(buildStart).Round(time.Millisecond))
			}
		}(ic.chunk, ic.chunkID)
	}

	wg.Wait()

	if len(errs) == len(eligible) && len(eligible) > 0 {
		return fmt.Errorf("graph.Build: all %d eligible chunks failed; first error: %s", len(eligible), errs[0])
	}
	if len(errs) > 0 {
		slog.Warn("graph: build completed with failures",
			"succeeded", len(eligible)-len(errs), "failed", len(errs), "total", len(eligible))
	}
	return nil
}

// codeBlockRe strips markdown code fences from LLM output.
var codeBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(.*?)\\n?```")

// extractJSON attempts to find a valid JSON object in the LLM response text.
// It handles common LLM quirks: markdown code blocks, text before/after JSON.
func extractJSON(raw string) (string, error) {
	// Strip markdown code blocks first.
	if m := codeBlockRe.FindStringSubmatch(raw); len(m) > 1 {
		raw = m[1]
	}

	raw = strings.TrimSpace(raw)

	// If it already starts with '{', try as-is.
	if strings.HasPrefix(raw, "{") {
		return raw, nil
	}

	// Find the first '{' and last '}' to extract the JSON object.
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1], nil
	}

	return "", fmt.Errorf("no JSON object found in response")
}

// entityResult is the JSON shape returned by the entity extraction LLM call.
type entityResult struct {
	Entities []ExtractedEntity `json:"entities"`
}

// relationshipResult is the JSON shape returned by the relationship extraction
// LLM call.
type relationshipResult struct {
	Relationships []ExtractedRelationship `json:"relationships"`
}

// extractEntities calls the LLM with a focused entity-only prompt.
// Pre-extracted identifiers are included as hints so the model does not miss
// structured data like part numbers, standards, and measurements.
func (b *Builder) extractEntities(ctx context.Context, chunk store.Chunk) ([]ExtractedEntity, error) {
	identifiers := preExtractIdentifiers(chunk.Content)

	var hintsSection string
	if len(identifiers) > 0 {
		hintsSection = fmt.Sprintf(
			"HINTS: The following identifiers were detected in the text. Make sure to include them as entities:\n%s\n",
			strings.Join(identifiers, ", "),
		)
	}

	prompt := fmt.Sprintf(entityExtractionPrompt, hintsSection, chunk.Content)

	resp, err := b.chat.Chat(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Temperature:    0.0,
		ResponseFormat: "json_object",
	})
	if err != nil {
		return nil, fmt.Errorf("entity extraction llm chat: %w", err)
	}

	jsonStr, err := extractJSON(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("parsing entity extraction result: %w", err)
	}

	var result entityResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("unmarshalling entity extraction result: %w", err)
	}

	return result.Entities, nil
}

// extractRelationships calls the LLM with the known entities and asks it to
// find only relationships (verbs) between them.
func (b *Builder) extractRelationships(ctx context.Context, chunk store.Chunk, entities []ExtractedEntity) ([]ExtractedRelationship, error) {
	if len(entities) < 2 {
		// Need at least two entities to form a relationship.
		return nil, nil
	}

	// Build the entity list for the prompt.
	entityNames := make([]string, 0, len(entities))
	for _, e := range entities {
		name := strings.TrimSpace(strings.ToLower(e.Name))
		if name != "" {
			entityNames = append(entityNames, name)
		}
	}

	entitiesJSON, _ := json.Marshal(entityNames)
	prompt := fmt.Sprintf(relationshipExtractionPrompt, string(entitiesJSON), chunk.Content)

	resp, err := b.chat.Chat(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Temperature:    0.0,
		ResponseFormat: "json_object",
	})
	if err != nil {
		return nil, fmt.Errorf("relationship extraction llm chat: %w", err)
	}

	jsonStr, err := extractJSON(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("parsing relationship extraction result: %w", err)
	}

	var result relationshipResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("unmarshalling relationship extraction result: %w", err)
	}

	return result.Relationships, nil
}

// processChunk orchestrates the multi-step extraction pipeline for a single
// chunk: first extracts entities, then extracts relationships given those
// entities, and finally persists the results.
func (b *Builder) processChunk(ctx context.Context, chunk store.Chunk, chunkID int64) error {
	// Step 1: Extract entities (atomic LLM call).
	entities, err := b.extractEntities(ctx, chunk)
	if err != nil {
		return fmt.Errorf("step 1 (entities): %w", err)
	}

	// Step 2: Extract relationships using the found entities (atomic LLM call).
	relationships, err := b.extractRelationships(ctx, chunk, entities)
	if err != nil {
		// Non-fatal: we still have entities to persist.
		slog.Warn("graph: relationship extraction failed, persisting entities only",
			"chunk_id", chunkID, "error", err)
		relationships = nil
	}

	// Build a combined result for persistence (preserves ExtractionResult type).
	result := ExtractionResult{
		Entities:      entities,
		Relationships: relationships,
	}

	// Build a map from entity name to its stored ID so relationships can
	// reference the correct rows.
	entityIDMap := make(map[string]int64, len(result.Entities))

	for _, e := range result.Entities {
		name := strings.TrimSpace(strings.ToLower(e.Name))
		if name == "" {
			continue
		}
		eType := strings.TrimSpace(strings.ToLower(e.Type))
		if eType == "" {
			eType = EntityConcept
		}

		// Upsert + link in a single transaction to avoid FK race conditions.
		id, err := b.store.UpsertEntityAndLink(ctx, store.Entity{
			Name:        name,
			EntityType:  eType,
			Description: e.Description,
		}, chunkID)
		if err != nil {
			slog.Warn("graph: entity upsert+link failed, skipping",
				"entity", name, "chunk", chunkID, "error", err)
			continue
		}
		entityIDMap[name] = id
	}

	for _, r := range result.Relationships {
		srcName := strings.TrimSpace(strings.ToLower(r.Source))
		tgtName := strings.TrimSpace(strings.ToLower(r.Target))
		if srcName == "" || tgtName == "" {
			continue
		}

		srcID, ok := entityIDMap[srcName]
		if !ok {
			// Source entity was not extracted in this chunk; try to look it up.
			entities, err := b.store.GetEntitiesByNames(ctx, []string{srcName})
			if err != nil || len(entities) == 0 {
				continue
			}
			srcID = entities[0].ID
		}

		tgtID, ok := entityIDMap[tgtName]
		if !ok {
			entities, err := b.store.GetEntitiesByNames(ctx, []string{tgtName})
			if err != nil || len(entities) == 0 {
				continue
			}
			tgtID = entities[0].ID
		}

		weight := r.Weight
		if weight <= 0 {
			weight = 1.0
		}

		chunkIDPtr := &chunkID
		if _, err := b.store.InsertRelationship(ctx, store.Relationship{
			SourceEntityID: srcID,
			TargetEntityID: tgtID,
			RelationType:   strings.TrimSpace(strings.ToLower(r.RelationType)),
			Weight:         weight,
			Description:    r.Description,
			SourceChunkID:  chunkIDPtr,
		}); err != nil {
			slog.Warn("graph: relationship insert failed, skipping",
				"source", srcName, "target", tgtName, "error", err)
			continue
		}
	}

	return nil
}
