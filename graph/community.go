package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/bbiangul/go-reason/llm"
	"github.com/bbiangul/go-reason/store"
)

// minComponentSplit is the minimum component size eligible for further
// modularity-based splitting.
const minComponentSplit = 6

// maxModularityNodes caps the node count for the modularity optimisation.
// Components larger than this are kept as level-0 only.
const maxModularityNodes = 200

// edge represents a weighted edge in the in-memory adjacency list.
type edge struct {
	to     int
	weight float64
}

// DetectCommunities runs community detection on the entity graph.
// Level-0 communities are connected components. Components larger than
// minComponentSplit are further split using greedy modularity optimisation and
// stored as level-1 communities.
func DetectCommunities(ctx context.Context, s *store.Store) ([]store.Community, error) {
	entities, err := s.AllEntities(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading entities: %w", err)
	}
	rels, err := s.AllRelationships(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading relationships: %w", err)
	}

	if len(entities) == 0 {
		return nil, nil
	}

	slog.Info("community: starting detection",
		"entities", len(entities), "relationships", len(rels))

	// Map entity ID -> index for compact adjacency representation.
	idIndex := make(map[int64]int, len(entities))
	for i, e := range entities {
		idIndex[e.ID] = i
	}

	// Build weighted adjacency list.
	adj := make([][]edge, len(entities))
	totalWeight := 0.0
	for _, r := range rels {
		si, okS := idIndex[r.SourceEntityID]
		ti, okT := idIndex[r.TargetEntityID]
		if !okS || !okT {
			continue
		}
		adj[si] = append(adj[si], edge{to: ti, weight: r.Weight})
		adj[ti] = append(adj[ti], edge{to: si, weight: r.Weight})
		totalWeight += r.Weight
	}

	// --- Level 0: connected components via BFS ---
	visited := make([]bool, len(entities))
	var components [][]int

	for i := range entities {
		if visited[i] {
			continue
		}
		var comp []int
		queue := []int{i}
		visited[i] = true
		for len(queue) > 0 {
			node := queue[0]
			queue = queue[1:]
			comp = append(comp, node)
			for _, e := range adj[node] {
				if !visited[e.to] {
					visited[e.to] = true
					queue = append(queue, e.to)
				}
			}
		}
		components = append(components, comp)
	}

	slog.Info("community: BFS found components",
		"components", len(components), "largest", largestComp(components))

	// Clear old community data before inserting new results.
	if err := s.ClearCommunities(ctx); err != nil {
		return nil, fmt.Errorf("clearing communities: %w", err)
	}

	var communities []store.Community

	for _, comp := range components {
		ids := componentEntityIDs(comp, entities)
		idsJSON, _ := json.Marshal(ids)

		c := store.Community{
			Level:     0,
			EntityIDs: string(idsJSON),
		}
		id, err := s.InsertCommunity(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("inserting level-0 community: %w", err)
		}
		c.ID = id
		communities = append(communities, c)

		// --- Level 1: modularity-based splitting for large components ---
		// Skip if too large (O(n²) algorithm would be too slow).
		if len(comp) >= minComponentSplit && len(comp) <= maxModularityNodes && totalWeight > 0 {
			subcommunities := modularitySplit(comp, adj, totalWeight)
			for _, sub := range subcommunities {
				subIDs := componentEntityIDs(sub, entities)
				subJSON, _ := json.Marshal(subIDs)

				sc := store.Community{
					Level:     1,
					EntityIDs: string(subJSON),
				}
				sid, err := s.InsertCommunity(ctx, sc)
				if err != nil {
					return nil, fmt.Errorf("inserting level-1 community: %w", err)
				}
				sc.ID = sid
				communities = append(communities, sc)
			}
		}
	}

	slog.Info("community: detection complete", "communities", len(communities))
	return communities, nil
}

func largestComp(comps [][]int) int {
	max := 0
	for _, c := range comps {
		if len(c) > max {
			max = len(c)
		}
	}
	return max
}

// componentEntityIDs maps component node indices back to entity IDs.
func componentEntityIDs(comp []int, entities []store.Entity) []int64 {
	ids := make([]int64, len(comp))
	for i, idx := range comp {
		ids[i] = entities[idx].ID
	}
	return ids
}

// modularitySplit applies a greedy modularity optimisation (simplified Louvain)
// to split a connected component into two or more sub-communities. If the
// split does not improve modularity the original component is returned as-is.
func modularitySplit(comp []int, adj [][]edge, totalWeight float64) [][]int {
	n := len(comp)
	if n < minComponentSplit {
		return [][]int{comp}
	}

	// Local index mapping for the subgraph.
	localIdx := make(map[int]int, n)
	for i, node := range comp {
		localIdx[node] = i
	}

	// community[i] is the community label for local node i.
	community := make([]int, n)
	for i := range community {
		community[i] = i // each node starts in its own community
	}

	// Compute node strengths (sum of edge weights within the subgraph).
	strength := make([]float64, n)
	for i, node := range comp {
		for _, e := range adj[node] {
			if _, ok := localIdx[e.to]; ok {
				strength[i] += e.weight
			}
		}
	}

	m2 := 2.0 * totalWeight
	if m2 == 0 {
		return [][]int{comp}
	}

	// Precompute community strengths (maintained incrementally).
	commStrength := make(map[int]float64, n)
	for i := range comp {
		commStrength[community[i]] += strength[i]
	}

	// Greedy modularity optimisation: repeatedly move nodes to the
	// neighbouring community that gives the best modularity gain.
	// Cap iterations to avoid pathological cases.
	maxPasses := 20
	for pass := 0; pass < maxPasses; pass++ {
		moved := false
		for i, node := range comp {
			// Compute weight to each neighbouring community.
			commWeights := make(map[int]float64)
			for _, e := range adj[node] {
				li, ok := localIdx[e.to]
				if !ok {
					continue
				}
				commWeights[community[li]] += e.weight
			}

			bestComm := community[i]
			bestGain := 0.0

			currentComm := community[i]
			kiIn := commWeights[currentComm]
			ki := strength[i]
			sigmaCurrent := commStrength[currentComm]

			// Removal delta.
			removeDelta := kiIn/m2 - (sigmaCurrent*ki)/(m2*m2)

			for c, wic := range commWeights {
				if c == currentComm {
					continue
				}
				sigmaC := commStrength[c]
				gain := (wic/m2 - (sigmaC*ki)/(m2*m2)) - removeDelta
				if gain > bestGain {
					bestGain = gain
					bestComm = c
				}
			}

			if bestComm != currentComm {
				// Update community strengths incrementally.
				commStrength[currentComm] -= ki
				commStrength[bestComm] += ki
				community[i] = bestComm
				moved = true
			}
		}
		if !moved {
			break
		}
	}

	// Group nodes by community label.
	groups := make(map[int][]int)
	for i, node := range comp {
		groups[community[i]] = append(groups[community[i]], node)
	}

	result := make([][]int, 0, len(groups))
	for _, g := range groups {
		result = append(result, g)
	}

	// If we ended up with only one group the split was not beneficial.
	if len(result) <= 1 {
		return [][]int{comp}
	}
	return result
}

// SummarizeCommunities uses the LLM to generate a natural-language summary
// for each community based on its member entities. Summaries are generated
// concurrently (up to 8 at a time) and individual failures are logged but
// do not abort the entire operation.
func SummarizeCommunities(ctx context.Context, s *store.Store, chat llm.Provider, communities []store.Community) error {
	// Load all entities once; filter per community.
	allEntities, err := s.AllEntities(ctx)
	if err != nil {
		return fmt.Errorf("loading entities for summarisation: %w", err)
	}

	// Build lookup by ID.
	entityByID := make(map[int64]store.Entity, len(allEntities))
	for _, e := range allEntities {
		entityByID[e.ID] = e
	}

	const concurrency = 8
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed int

	for i := range communities {
		c := &communities[i]

		var entityIDs []int64
		if err := json.Unmarshal([]byte(c.EntityIDs), &entityIDs); err != nil {
			slog.Warn("community: failed to parse entity_ids", "community_id", c.ID, "error", err)
			failed++
			continue
		}

		if len(entityIDs) == 0 {
			continue
		}

		// Collect entity descriptions for the prompt.
		var descriptions []string
		for _, eid := range entityIDs {
			e, ok := entityByID[eid]
			if !ok {
				continue
			}
			if e.Description != "" {
				descriptions = append(descriptions, fmt.Sprintf("- %s (%s): %s", e.Name, e.EntityType, e.Description))
			} else {
				descriptions = append(descriptions, fmt.Sprintf("- %s (%s)", e.Name, e.EntityType))
			}
		}

		if len(descriptions) == 0 {
			continue
		}

		prompt := fmt.Sprintf(
			"Summarize the following group of related entities in 2-3 sentences. "+
				"Explain what connects them and their significance.\n\nEntities:\n%s",
			strings.Join(descriptions, "\n"),
		)

		wg.Add(1)
		sem <- struct{}{}
		go func(c *store.Community, prompt string, idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			resp, err := chat.Chat(ctx, llm.ChatRequest{
				Messages: []llm.Message{
					{Role: "user", Content: prompt},
				},
				Temperature: 0.3,
			})
			if err != nil {
				slog.Warn("community: summarization failed",
					"community_id", c.ID, "error", err)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}

			summary := strings.TrimSpace(resp.Content)

			db := s.DB()
			if _, err := db.ExecContext(ctx,
				"UPDATE communities SET summary = ? WHERE id = ?",
				summary, c.ID,
			); err != nil {
				slog.Warn("community: failed to store summary",
					"community_id", c.ID, "error", err)
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}

			mu.Lock()
			c.Summary = summary
			done := len(communities) - failed - countPending(&wg)
			mu.Unlock()

			_ = done // progress logged below
			slog.Info("community: summarized",
				"community_id", c.ID,
				"progress", fmt.Sprintf("%d/%d", idx+1, len(communities)))
		}(c, prompt, i)
	}

	wg.Wait()

	if failed > 0 {
		slog.Warn("community: some summaries failed", "failed", failed, "total", len(communities))
	}
	slog.Info("community: summarization complete",
		"succeeded", len(communities)-failed, "failed", failed)
	return nil
}

// countPending returns a rough count of pending goroutines. Used only for
// progress logging, not for correctness.
func countPending(_ *sync.WaitGroup) int { return 0 }
