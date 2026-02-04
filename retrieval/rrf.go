package retrieval

import (
	"sort"

	"github.com/brunobiangulo/goreason/store"
)

const rrfK = 60 // RRF constant (standard value from literature)

// FusedResultInfo holds per-result method contribution metadata.
type FusedResultInfo struct {
	Methods   []string `json:"methods"`
	VecRank   int      `json:"vec_rank,omitempty"`   // 1-based, 0 = not present
	FTSRank   int      `json:"fts_rank,omitempty"`   // 1-based, 0 = not present
	GraphRank int      `json:"graph_rank,omitempty"` // 1-based, 0 = not present
}

// fuseRRF implements Reciprocal Rank Fusion to combine results from
// multiple retrieval methods. Each result set is ranked independently,
// then scores are combined using: score = sum(weight_i / (k + rank_i)).
// It also returns per-result method contribution info keyed by ChunkID.
func fuseRRF(
	vecResults, ftsResults, graphResults []store.RetrievalResult,
	weightVec, weightFTS, weightGraph float64,
	maxResults int,
) ([]store.RetrievalResult, map[int64]FusedResultInfo) {
	// Map from chunk_id -> fused score and result data
	type fusedEntry struct {
		result store.RetrievalResult
		score  float64
		info   FusedResultInfo
	}

	fused := make(map[int64]*fusedEntry)

	// Add vector results with their RRF scores
	for rank, r := range vecResults {
		entry, ok := fused[r.ChunkID]
		if !ok {
			entry = &fusedEntry{result: r}
			fused[r.ChunkID] = entry
		}
		entry.score += weightVec / float64(rrfK+rank+1)
		entry.info.Methods = append(entry.info.Methods, "vector")
		entry.info.VecRank = rank + 1
	}

	// Add FTS results
	for rank, r := range ftsResults {
		entry, ok := fused[r.ChunkID]
		if !ok {
			entry = &fusedEntry{result: r}
			fused[r.ChunkID] = entry
		}
		entry.score += weightFTS / float64(rrfK+rank+1)
		entry.info.Methods = append(entry.info.Methods, "fts")
		entry.info.FTSRank = rank + 1
	}

	// Add graph results
	for rank, r := range graphResults {
		entry, ok := fused[r.ChunkID]
		if !ok {
			entry = &fusedEntry{result: r}
			fused[r.ChunkID] = entry
		}
		entry.score += weightGraph / float64(rrfK+rank+1)
		entry.info.Methods = append(entry.info.Methods, "graph")
		entry.info.GraphRank = rank + 1
	}

	// Sort by fused score
	entries := make([]*fusedEntry, 0, len(fused))
	for _, e := range fused {
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})

	// Limit results
	if maxResults > 0 && len(entries) > maxResults {
		entries = entries[:maxResults]
	}

	results := make([]store.RetrievalResult, len(entries))
	infoMap := make(map[int64]FusedResultInfo, len(entries))
	for i, e := range entries {
		results[i] = e.result
		results[i].Score = e.score
		infoMap[e.result.ChunkID] = e.info
	}

	return results, infoMap
}
