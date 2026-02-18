package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LegalBenchSnippet is a ground-truth snippet from the LegalBench-RAG benchmark.
type LegalBenchSnippet struct {
	FilePath string `json:"file_path"`
	Span     [2]int `json:"span"`   // [start, end] character offsets
	Answer   string `json:"answer"` // pre-extracted snippet text
}

// LegalBenchTest is a single Q&A test case from LegalBench-RAG.
type LegalBenchTest struct {
	Query    string              `json:"query"`
	Snippets []LegalBenchSnippet `json:"snippets"`
	Tags     []string            `json:"tags"`
}

// LegalBenchBenchmark is the top-level benchmark file structure.
type LegalBenchBenchmark struct {
	Tests []LegalBenchTest `json:"tests"`
}

// LegalBenchConfig controls how LegalBench-RAG data is loaded.
type LegalBenchConfig struct {
	// BenchmarkFiles are paths to benchmark JSON files.
	BenchmarkFiles []string
	// CorpusDir is the path to the corpus directory (for reading snippet text).
	CorpusDir string
	// MaxTestsPerBenchmark caps the number of tests loaded per benchmark file.
	// 0 means no limit (load all). 194 matches the LegalBench-RAG-mini subset.
	MaxTestsPerBenchmark int
}

// LoadLegalBenchDatasets loads LegalBench-RAG benchmark JSON files and converts
// them into GoReason Dataset format. Each benchmark file becomes a separate
// dataset (e.g., CUAD, ContractNLI, MAUD, PrivacyQA).
func LoadLegalBenchDatasets(cfg LegalBenchConfig) ([]Dataset, error) {
	var datasets []Dataset

	for _, path := range cfg.BenchmarkFiles {
		ds, err := loadLegalBenchFile(path, cfg.CorpusDir, cfg.MaxTestsPerBenchmark)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", path, err)
		}
		datasets = append(datasets, ds)
	}

	return datasets, nil
}

// UsedCorpusFiles returns the set of corpus file paths referenced by the
// loaded benchmark tests. When MaxTestsPerBenchmark is set, this returns
// only files needed for the subset — useful for skipping ingestion of
// unreferenced documents.
func UsedCorpusFiles(cfg LegalBenchConfig) (map[string]struct{}, error) {
	used := make(map[string]struct{})
	for _, path := range cfg.BenchmarkFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		var benchmark LegalBenchBenchmark
		if err := json.Unmarshal(data, &benchmark); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		tests := benchmark.Tests
		if cfg.MaxTestsPerBenchmark > 0 && len(tests) > cfg.MaxTestsPerBenchmark {
			// Sort by first snippet file path (groups tests by document,
			// matching the paper's SORT_BY_DOCUMENT behavior).
			sort.Slice(tests, func(i, j int) bool {
				fi, fj := "", ""
				if len(tests[i].Snippets) > 0 {
					fi = tests[i].Snippets[0].FilePath
				}
				if len(tests[j].Snippets) > 0 {
					fj = tests[j].Snippets[0].FilePath
				}
				return fi < fj
			})
			tests = tests[:cfg.MaxTestsPerBenchmark]
		}
		for _, t := range tests {
			for _, s := range t.Snippets {
				used[s.FilePath] = struct{}{}
			}
		}
	}
	return used, nil
}

// loadLegalBenchFile loads a single benchmark JSON file and converts it
// into a GoReason Dataset. If maxTests > 0, only the first maxTests entries
// are loaded (sorted by document to minimize corpus, matching the paper).
func loadLegalBenchFile(path string, corpusDir string, maxTests int) (Dataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Dataset{}, fmt.Errorf("reading file: %w", err)
	}

	var benchmark LegalBenchBenchmark
	if err := json.Unmarshal(data, &benchmark); err != nil {
		return Dataset{}, fmt.Errorf("parsing JSON: %w", err)
	}

	// Derive dataset name from filename (e.g., "cuad.json" -> "CUAD")
	baseName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	category := strings.ToUpper(baseName)

	entries := benchmark.Tests
	if maxTests > 0 && len(entries) > maxTests {
		// Sort by first snippet file path (groups by document, matching paper).
		sort.Slice(entries, func(i, j int) bool {
			fi, fj := "", ""
			if len(entries[i].Snippets) > 0 {
				fi = entries[i].Snippets[0].FilePath
			}
			if len(entries[j].Snippets) > 0 {
				fj = entries[j].Snippets[0].FilePath
			}
			return fi < fj
		})
		entries = entries[:maxTests]
	}

	var tests []TestCase
	for _, t := range entries {
		tc, err := convertLegalBenchTest(t, category, corpusDir)
		if err != nil {
			// Skip malformed entries rather than failing the whole file.
			continue
		}
		tests = append(tests, tc)
	}

	return Dataset{
		Name:       fmt.Sprintf("LegalBench-RAG - %s", category),
		Difficulty: "all",
		Tests:      tests,
	}, nil
}

// convertLegalBenchTest converts a LegalBench-RAG test into a GoReason TestCase.
func convertLegalBenchTest(t LegalBenchTest, category string, corpusDir string) (TestCase, error) {
	if len(t.Snippets) == 0 {
		return TestCase{}, fmt.Errorf("no snippets for query: %s", t.Query)
	}

	var expectedFacts []string
	var explanations []string

	for _, snippet := range t.Snippets {
		// Use the pre-extracted answer text from JSON when available;
		// fall back to reading from corpus file via span offsets.
		text := snippet.Answer
		if text == "" {
			var err error
			text, err = readSnippetText(corpusDir, snippet.FilePath, snippet.Span[0], snippet.Span[1])
			if err != nil {
				return TestCase{}, fmt.Errorf("reading snippet: %w", err)
			}
		}

		// Extract key phrases from the snippet text to use as expected facts.
		facts := extractKeyPhrases(text)
		expectedFacts = append(expectedFacts, facts...)

		explanations = append(explanations, fmt.Sprintf("%s [%d:%d]",
			snippet.FilePath, snippet.Span[0], snippet.Span[1]))
	}

	// Deduplicate expected facts
	expectedFacts = dedup(expectedFacts)

	return TestCase{
		Question:      t.Query,
		ExpectedFacts: expectedFacts,
		Category:      category,
		Explanation:   strings.Join(explanations, "; "),
	}, nil
}

// readSnippetText reads a character span from a corpus file.
func readSnippetText(corpusDir, filePath string, start, end int) (string, error) {
	fullPath := filepath.Join(corpusDir, filePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", fullPath, err)
	}

	content := string(data)
	if start < 0 || end > len(content) || start >= end {
		return "", fmt.Errorf("span [%d:%d] out of range for %s (len=%d)",
			start, end, filePath, len(content))
	}

	return content[start:end], nil
}

// extractKeyPhrases extracts meaningful phrases from snippet text for fact matching.
// Uses sentence-level chunks — each sentence becomes an expected fact. Short
// sentences (< 20 chars) are skipped as they're usually noise.
func extractKeyPhrases(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// If the snippet is short enough, use it as a single fact.
	if len(text) <= 200 {
		return []string{text}
	}

	// Split into sentences and use each substantial one as a fact.
	var facts []string
	sentences := splitSentences(text)
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) >= 20 {
			facts = append(facts, s)
		}
	}

	// If sentence splitting produced nothing useful, fall back to the whole text.
	if len(facts) == 0 {
		return []string{text}
	}

	return facts
}

// splitSentences does a basic sentence split on common terminators.
func splitSentences(text string) []string {
	// Split on sentence-ending punctuation followed by space or end-of-string.
	var sentences []string
	current := strings.Builder{}

	for i, r := range text {
		current.WriteRune(r)
		if (r == '.' || r == ';' || r == '!' || r == '?') &&
			(i+1 >= len(text) || text[i+1] == ' ' || text[i+1] == '\n') {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}

	// Remaining text
	if s := strings.TrimSpace(current.String()); s != "" {
		sentences = append(sentences, s)
	}

	return sentences
}

func dedup(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	var result []string
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	return result
}

// GroundTruthSpan records a ground-truth snippet location for retrieval evaluation.
type GroundTruthSpan struct {
	FilePath string
	Start    int
	End      int
	Text     string
}

// LoadLegalBenchGroundTruth loads the raw ground-truth spans from benchmark files.
// This is used for retrieval P@k/R@k computation (matching retrieved chunks
// against exact document spans).
func LoadLegalBenchGroundTruth(cfg LegalBenchConfig) (map[string][]GroundTruthSpan, error) {
	result := make(map[string][]GroundTruthSpan)

	for _, path := range cfg.BenchmarkFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		var benchmark LegalBenchBenchmark
		if err := json.Unmarshal(data, &benchmark); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}

		tests := benchmark.Tests
		if cfg.MaxTestsPerBenchmark > 0 && len(tests) > cfg.MaxTestsPerBenchmark {
			sort.Slice(tests, func(i, j int) bool {
				fi, fj := "", ""
				if len(tests[i].Snippets) > 0 {
					fi = tests[i].Snippets[0].FilePath
				}
				if len(tests[j].Snippets) > 0 {
					fj = tests[j].Snippets[0].FilePath
				}
				return fi < fj
			})
			tests = tests[:cfg.MaxTestsPerBenchmark]
		}

		for _, t := range tests {
			var spans []GroundTruthSpan
			for _, snippet := range t.Snippets {
				text := snippet.Answer
				if text == "" {
					var err error
					text, err = readSnippetText(cfg.CorpusDir, snippet.FilePath, snippet.Span[0], snippet.Span[1])
					if err != nil {
						continue
					}
				}
				spans = append(spans, GroundTruthSpan{
					FilePath: snippet.FilePath,
					Start:    snippet.Span[0],
					End:      snippet.Span[1],
					Text:     text,
				})
			}
			if len(spans) > 0 {
				result[t.Query] = spans
			}
		}
	}

	return result, nil
}
