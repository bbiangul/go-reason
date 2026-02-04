//go:build cgo

package graph

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/brunobiangulo/goreason/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.New(dbPath, 4)
	if err != nil {
		t.Fatalf("creating store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// seedEntitiesAndRelationships inserts a small graph into the store and returns
// entity IDs and chunk IDs that were created.
func seedEntitiesAndRelationships(t *testing.T, s *store.Store) (entityIDs map[string]int64, chunkIDs []int64) {
	t.Helper()
	ctx := context.Background()

	// Insert a document so chunks have a valid document_id.
	docID, err := s.UpsertDocument(ctx, store.Document{
		Path:        "/tmp/test.pdf",
		Filename:    "test.pdf",
		Format:      "pdf",
		ContentHash: "abc123",
		ParseMethod: "native",
		Status:      "ready",
	})
	if err != nil {
		t.Fatalf("upserting document: %v", err)
	}

	// Insert chunks.
	chunks := []store.Chunk{
		{DocumentID: docID, Content: "ISO 9001 defines quality management systems.", ChunkType: "text", Heading: "Quality", PageNumber: 1, PositionInDoc: 0, TokenCount: 10},
		{DocumentID: docID, Content: "Risk assessment requires ISO 31000 compliance.", ChunkType: "text", Heading: "Risk", PageNumber: 2, PositionInDoc: 1, TokenCount: 10},
		{DocumentID: docID, Content: "The audit process follows ISO 19011 guidelines.", ChunkType: "text", Heading: "Audit", PageNumber: 3, PositionInDoc: 2, TokenCount: 10},
	}
	ids, err := s.InsertChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("inserting chunks: %v", err)
	}
	chunkIDs = ids

	// Insert entities.
	entityIDs = make(map[string]int64)
	entities := []store.Entity{
		{Name: "iso 9001", EntityType: EntityStandard, Description: "Quality management standard"},
		{Name: "iso 31000", EntityType: EntityStandard, Description: "Risk management standard"},
		{Name: "iso 19011", EntityType: EntityStandard, Description: "Audit guidelines standard"},
		{Name: "quality management", EntityType: EntityConcept, Description: "Management of quality processes"},
		{Name: "risk assessment", EntityType: EntityConcept, Description: "Assessing risks"},
		{Name: "audit process", EntityType: EntityConcept, Description: "Process of conducting audits"},
	}
	for _, e := range entities {
		id, err := s.UpsertEntity(ctx, e)
		if err != nil {
			t.Fatalf("upserting entity %q: %v", e.Name, err)
		}
		entityIDs[e.Name] = id
	}

	// Link entities to chunks.
	links := map[string]int{
		"iso 9001":           0,
		"quality management": 0,
		"iso 31000":          1,
		"risk assessment":    1,
		"iso 19011":          2,
		"audit process":      2,
	}
	for name, chunkIdx := range links {
		if err := s.LinkEntityChunk(ctx, entityIDs[name], chunkIDs[chunkIdx]); err != nil {
			t.Fatalf("linking entity %q to chunk: %v", name, err)
		}
	}

	// Insert relationships.
	relationships := []store.Relationship{
		{SourceEntityID: entityIDs["iso 9001"], TargetEntityID: entityIDs["quality management"], RelationType: RelDefines, Weight: 0.9, Description: "ISO 9001 defines quality management"},
		{SourceEntityID: entityIDs["iso 31000"], TargetEntityID: entityIDs["risk assessment"], RelationType: RelDefines, Weight: 0.85, Description: "ISO 31000 defines risk assessment"},
		{SourceEntityID: entityIDs["iso 19011"], TargetEntityID: entityIDs["audit process"], RelationType: RelDefines, Weight: 0.8, Description: "ISO 19011 defines audit process"},
		{SourceEntityID: entityIDs["iso 9001"], TargetEntityID: entityIDs["iso 31000"], RelationType: RelReferences, Weight: 0.7, Description: "ISO 9001 references ISO 31000"},
		{SourceEntityID: entityIDs["iso 9001"], TargetEntityID: entityIDs["iso 19011"], RelationType: RelReferences, Weight: 0.6, Description: "ISO 9001 references ISO 19011"},
	}
	for _, r := range relationships {
		if _, err := s.InsertRelationship(ctx, r); err != nil {
			t.Fatalf("inserting relationship: %v", err)
		}
	}

	return entityIDs, chunkIDs
}

func TestExtractionResultParsing(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantEntities  int
		wantRelations int
		wantErr       bool
	}{
		{
			name: "valid full extraction",
			input: `{
				"entities": [
					{"name": "ISO 9001", "type": "standard", "description": "Quality management standard"},
					{"name": "quality management", "type": "concept", "description": "A management approach"}
				],
				"relationships": [
					{"source": "ISO 9001", "target": "quality management", "relation_type": "defines", "description": "defines QM", "weight": 0.95}
				]
			}`,
			wantEntities:  2,
			wantRelations: 1,
		},
		{
			name:          "empty arrays",
			input:         `{"entities": [], "relationships": []}`,
			wantEntities:  0,
			wantRelations: 0,
		},
		{
			name: "entities only",
			input: `{
				"entities": [
					{"name": "NIST 800-53", "type": "standard", "description": "Security controls"}
				],
				"relationships": []
			}`,
			wantEntities:  1,
			wantRelations: 0,
		},
		{
			name:    "invalid json",
			input:   `{not valid json}`,
			wantErr: true,
		},
		{
			name: "multiple relationships",
			input: `{
				"entities": [
					{"name": "iso 27001", "type": "standard", "description": "Information security"},
					{"name": "iso 27002", "type": "standard", "description": "Code of practice"},
					{"name": "information security", "type": "concept", "description": "Protecting info"}
				],
				"relationships": [
					{"source": "iso 27001", "target": "iso 27002", "relation_type": "references", "description": "references", "weight": 0.8},
					{"source": "iso 27001", "target": "information security", "relation_type": "defines", "description": "defines", "weight": 0.9}
				]
			}`,
			wantEntities:  3,
			wantRelations: 2,
		},
		{
			name: "weight boundaries",
			input: `{
				"entities": [
					{"name": "entity a", "type": "concept", "description": "test"}
				],
				"relationships": [
					{"source": "entity a", "target": "entity a", "relation_type": "references", "description": "self-ref", "weight": 0.0},
					{"source": "entity a", "target": "entity a", "relation_type": "defines", "description": "self-def", "weight": 1.0}
				]
			}`,
			wantEntities:  1,
			wantRelations: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result ExtractionResult
			err := json.Unmarshal([]byte(tt.input), &result)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := len(result.Entities); got != tt.wantEntities {
				t.Errorf("entities count: got %d, want %d", got, tt.wantEntities)
			}
			if got := len(result.Relationships); got != tt.wantRelations {
				t.Errorf("relationships count: got %d, want %d", got, tt.wantRelations)
			}

			// Verify fields are populated for non-empty results.
			for i, e := range result.Entities {
				if e.Name == "" {
					t.Errorf("entity[%d] has empty name", i)
				}
				if e.Type == "" {
					t.Errorf("entity[%d] has empty type", i)
				}
			}
			for i, r := range result.Relationships {
				if r.Source == "" {
					t.Errorf("relationship[%d] has empty source", i)
				}
				if r.Target == "" {
					t.Errorf("relationship[%d] has empty target", i)
				}
				if r.RelationType == "" {
					t.Errorf("relationship[%d] has empty relation_type", i)
				}
			}
		})
	}
}

func TestCommunityDetection(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	entityIDs, _ := seedEntitiesAndRelationships(t, s)

	communities, err := DetectCommunities(ctx, s)
	if err != nil {
		t.Fatalf("DetectCommunities: %v", err)
	}

	if len(communities) == 0 {
		t.Fatal("expected at least one community, got none")
	}

	// All entities are connected (iso 9001 links to iso 31000 and iso 19011,
	// and each of those links to their concept). Therefore they should all
	// be in one level-0 community.
	var level0 []store.Community
	for _, c := range communities {
		if c.Level == 0 {
			level0 = append(level0, c)
		}
	}

	if len(level0) == 0 {
		t.Fatal("expected at least one level-0 community")
	}

	// Verify that the level-0 community contains all entity IDs.
	var allFoundIDs []int64
	for _, c := range level0 {
		var ids []int64
		if err := json.Unmarshal([]byte(c.EntityIDs), &ids); err != nil {
			t.Fatalf("parsing community entity_ids: %v", err)
		}
		allFoundIDs = append(allFoundIDs, ids...)
	}

	expectedEntityCount := len(entityIDs)
	if len(allFoundIDs) != expectedEntityCount {
		t.Errorf("expected %d entity IDs across level-0 communities, got %d", expectedEntityCount, len(allFoundIDs))
	}

	// Verify communities are persisted in the store.
	storedL0, err := s.GetCommunities(ctx, 0)
	if err != nil {
		t.Fatalf("GetCommunities(0): %v", err)
	}
	if len(storedL0) != len(level0) {
		t.Errorf("stored level-0 communities: got %d, want %d", len(storedL0), len(level0))
	}
}

func TestCommunityDetectionEmptyGraph(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	communities, err := DetectCommunities(ctx, s)
	if err != nil {
		t.Fatalf("DetectCommunities on empty graph: %v", err)
	}
	if communities != nil {
		t.Errorf("expected nil communities for empty graph, got %d", len(communities))
	}
}

func TestTraverse(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	entityIDs, chunkIDs := seedEntitiesAndRelationships(t, s)

	t.Run("single seed entity with depth 1", func(t *testing.T) {
		// Start from "iso 9001" with depth 1. It is directly connected to:
		// - quality management (via defines)
		// - iso 31000 (via references)
		// - iso 19011 (via references)
		result, err := Traverse(ctx, s, []string{"iso 9001"}, 1)
		if err != nil {
			t.Fatalf("Traverse: %v", err)
		}

		if len(result.EntityIDs) == 0 {
			t.Fatal("expected at least one entity in traversal result")
		}

		// iso 9001 itself plus its 3 direct neighbours.
		expectedEntities := 4
		if len(result.EntityIDs) != expectedEntities {
			t.Errorf("entity count: got %d, want %d", len(result.EntityIDs), expectedEntities)
		}

		// Verify all expected entity IDs are present.
		foundEntities := make(map[int64]bool)
		for _, eid := range result.EntityIDs {
			foundEntities[eid] = true
		}
		for _, name := range []string{"iso 9001", "quality management", "iso 31000", "iso 19011"} {
			if !foundEntities[entityIDs[name]] {
				t.Errorf("expected entity %q (ID %d) in result", name, entityIDs[name])
			}
		}

		// Verify chunks are found.
		if len(result.ChunkIDs) == 0 {
			t.Error("expected at least one chunk in traversal result")
		}
	})

	t.Run("single seed entity with depth 0", func(t *testing.T) {
		// Depth 0 means only the seed itself.
		result, err := Traverse(ctx, s, []string{"iso 9001"}, 0)
		if err != nil {
			t.Fatalf("Traverse: %v", err)
		}

		if len(result.EntityIDs) != 1 {
			t.Errorf("entity count at depth 0: got %d, want 1", len(result.EntityIDs))
		}
		if result.EntityIDs[0] != entityIDs["iso 9001"] {
			t.Errorf("expected seed entity ID %d, got %d", entityIDs["iso 9001"], result.EntityIDs[0])
		}
	})

	t.Run("multiple seed entities", func(t *testing.T) {
		result, err := Traverse(ctx, s, []string{"iso 9001", "audit process"}, 1)
		if err != nil {
			t.Fatalf("Traverse: %v", err)
		}

		// Both seeds plus their neighbours.
		if len(result.EntityIDs) < 2 {
			t.Errorf("expected at least 2 entities with multiple seeds, got %d", len(result.EntityIDs))
		}
	})

	t.Run("nonexistent seed entity", func(t *testing.T) {
		result, err := Traverse(ctx, s, []string{"nonexistent entity"}, 1)
		if err != nil {
			t.Fatalf("Traverse: %v", err)
		}
		if len(result.EntityIDs) != 0 {
			t.Errorf("expected 0 entities for nonexistent seed, got %d", len(result.EntityIDs))
		}
	})

	t.Run("empty query entities", func(t *testing.T) {
		result, err := Traverse(ctx, s, []string{}, 1)
		if err != nil {
			t.Fatalf("Traverse: %v", err)
		}
		if len(result.EntityIDs) != 0 {
			t.Errorf("expected 0 entities for empty query, got %d", len(result.EntityIDs))
		}
	})

	t.Run("negative depth", func(t *testing.T) {
		result, err := Traverse(ctx, s, []string{"iso 9001"}, -1)
		if err != nil {
			t.Fatalf("Traverse: %v", err)
		}
		if len(result.EntityIDs) != 0 {
			t.Errorf("expected 0 entities for negative depth, got %d", len(result.EntityIDs))
		}
	})

	t.Run("deep traversal covers full graph", func(t *testing.T) {
		result, err := Traverse(ctx, s, []string{"iso 9001"}, 3)
		if err != nil {
			t.Fatalf("Traverse: %v", err)
		}

		// With depth 3 from iso 9001, all 6 entities should be reachable.
		if len(result.EntityIDs) != len(entityIDs) {
			t.Errorf("entity count at depth 3: got %d, want %d", len(result.EntityIDs), len(entityIDs))
		}

		// All 3 chunks should be referenced.
		if len(result.ChunkIDs) != len(chunkIDs) {
			t.Errorf("chunk count at depth 3: got %d, want %d", len(result.ChunkIDs), len(chunkIDs))
		}
	})
}
