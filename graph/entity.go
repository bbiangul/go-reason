package graph

// Entity type constants used during extraction and storage.
const (
	EntityPerson     = "person"
	EntityOrg        = "organization"
	EntityStandard   = "standard"
	EntityClause     = "clause"
	EntityConcept    = "concept"
	EntityTerm       = "term"
	EntityRegulation = "regulation"
)

// Relation type constants used during extraction and storage.
const (
	RelReferences   = "references"
	RelDefines      = "defines"
	RelAmends       = "amends"
	RelRequires     = "requires"
	RelContradicts  = "contradicts"
	RelSupersedes   = "supersedes"
)

// ExtractedEntity is what the LLM returns from entity extraction.
type ExtractedEntity struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ExtractedRelationship is what the LLM returns from relationship extraction.
type ExtractedRelationship struct {
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	RelationType string  `json:"relation_type"`
	Description  string  `json:"description"`
	Weight       float64 `json:"weight"`
}

// ExtractionResult holds the LLM's structured output for a chunk.
type ExtractionResult struct {
	Entities      []ExtractedEntity       `json:"entities"`
	Relationships []ExtractedRelationship `json:"relationships"`
}
