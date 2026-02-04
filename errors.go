package goreason

import "errors"

var (
	// ErrDocumentNotFound is returned when a document ID does not exist.
	ErrDocumentNotFound = errors.New("goreason: document not found")

	// ErrDocumentExists is returned when trying to ingest a duplicate path.
	ErrDocumentExists = errors.New("goreason: document already exists")

	// ErrUnsupportedFormat is returned for unrecognized file formats.
	ErrUnsupportedFormat = errors.New("goreason: unsupported document format")

	// ErrParsingFailed is returned when document parsing fails.
	ErrParsingFailed = errors.New("goreason: parsing failed")

	// ErrEmbeddingFailed is returned when embedding generation fails.
	ErrEmbeddingFailed = errors.New("goreason: embedding generation failed")

	// ErrLLMUnavailable is returned when the LLM provider is unreachable.
	ErrLLMUnavailable = errors.New("goreason: LLM provider unavailable")

	// ErrLLMRequestFailed is returned when an LLM request fails.
	ErrLLMRequestFailed = errors.New("goreason: LLM request failed")

	// ErrStoreClosed is returned when operating on a closed store.
	ErrStoreClosed = errors.New("goreason: store is closed")

	// ErrNoResults is returned when retrieval yields no matching chunks.
	ErrNoResults = errors.New("goreason: no results found")

	// ErrLowConfidence is returned when the answer confidence is below threshold.
	ErrLowConfidence = errors.New("goreason: answer confidence below threshold")

	// ErrInvalidConfig is returned for invalid configuration values.
	ErrInvalidConfig = errors.New("goreason: invalid configuration")

	// ErrVisionRequired is returned when a document requires vision processing
	// but no vision provider is configured.
	ErrVisionRequired = errors.New("goreason: vision provider required for this document")

	// ErrExternalParserRequired is returned when a legacy format needs an
	// external parsing service that is not configured.
	ErrExternalParserRequired = errors.New("goreason: external parser required for legacy format")
)
