package graph

import (
	"context"
	"fmt"

	"github.com/brunobiangulo/goreason/store"
)

// TraversalResult contains entities and chunks found via graph traversal.
type TraversalResult struct {
	EntityIDs []int64
	ChunkIDs  []int64
}

// Traverse finds entities matching query terms and follows relationships
// to discover related chunks. Uses BFS with configurable depth.
//
// queryEntities are entity names (case-insensitive lookup). The traversal
// walks outgoing and incoming relationships up to maxDepth hops, collecting
// all entity IDs and their associated chunk IDs.
func Traverse(ctx context.Context, s *store.Store, queryEntities []string, maxDepth int) (*TraversalResult, error) {
	if len(queryEntities) == 0 || maxDepth < 0 {
		return &TraversalResult{}, nil
	}

	// Seed: look up entities by name.
	seeds, err := s.GetEntitiesByNames(ctx, queryEntities)
	if err != nil {
		return nil, fmt.Errorf("graph.Traverse: looking up seed entities: %w", err)
	}
	if len(seeds) == 0 {
		return &TraversalResult{}, nil
	}

	// Load the full graph into memory for fast traversal.
	allRels, err := s.AllRelationships(ctx)
	if err != nil {
		return nil, fmt.Errorf("graph.Traverse: loading relationships: %w", err)
	}

	// Build adjacency: entity ID -> list of neighbour entity IDs.
	neighbours := make(map[int64][]int64)
	for _, r := range allRels {
		neighbours[r.SourceEntityID] = append(neighbours[r.SourceEntityID], r.TargetEntityID)
		neighbours[r.TargetEntityID] = append(neighbours[r.TargetEntityID], r.SourceEntityID)
	}

	// BFS from seed entities.
	visited := make(map[int64]bool)
	queue := make([]int64, 0, len(seeds))
	for _, e := range seeds {
		if !visited[e.ID] {
			visited[e.ID] = true
			queue = append(queue, e.ID)
		}
	}

	for depth := 0; depth < maxDepth && len(queue) > 0; depth++ {
		var next []int64
		for _, eid := range queue {
			for _, nid := range neighbours[eid] {
				if !visited[nid] {
					visited[nid] = true
					next = append(next, nid)
				}
			}
		}
		queue = next
	}

	// Collect all visited entity IDs.
	entityIDs := make([]int64, 0, len(visited))
	for id := range visited {
		entityIDs = append(entityIDs, id)
	}

	// Resolve chunk IDs linked to the discovered entities via entity_chunks.
	chunkIDs, err := chunkIDsForEntities(ctx, s, entityIDs)
	if err != nil {
		return nil, fmt.Errorf("graph.Traverse: resolving chunks: %w", err)
	}

	return &TraversalResult{
		EntityIDs: entityIDs,
		ChunkIDs:  chunkIDs,
	}, nil
}

// chunkIDsForEntities queries the entity_chunks table to find all chunk IDs
// linked to the given entity IDs. It queries in batches to avoid overly large
// IN clauses.
func chunkIDsForEntities(ctx context.Context, s *store.Store, entityIDs []int64) ([]int64, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}

	db := s.DB()

	const batchSize = 200
	seen := make(map[int64]bool)
	var result []int64

	for start := 0; start < len(entityIDs); start += batchSize {
		end := start + batchSize
		if end > len(entityIDs) {
			end = len(entityIDs)
		}
		batch := entityIDs[start:end]

		placeholders := "?"
		for i := 1; i < len(batch); i++ {
			placeholders += ", ?"
		}

		query := "SELECT DISTINCT chunk_id FROM entity_chunks WHERE entity_id IN (" + placeholders + ")"
		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("querying entity_chunks: %w", err)
		}

		for rows.Next() {
			var cid int64
			if err := rows.Scan(&cid); err != nil {
				rows.Close()
				return nil, err
			}
			if !seen[cid] {
				seen[cid] = true
				result = append(result, cid)
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	return result, nil
}
