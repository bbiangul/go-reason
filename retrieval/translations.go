package retrieval

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/brunobiangulo/goreason/llm"
	"github.com/brunobiangulo/goreason/store"
)

// Translator provides cross-language query expansion by reading corpus
// languages from the database and translating query terms to all non-English
// document languages via an LLM at runtime. Results are cached in memory so
// each unique term is only translated once per engine lifetime.
type Translator struct {
	chatLLM llm.Provider
	store   *store.Store

	mu    sync.RWMutex
	langs []string            // cached corpus languages
	cache map[string][]string // lowercase English term → translated forms
}

// NewTranslator creates a Translator. If chatLLM is nil translation is a
// no-op (all methods return nil).
func NewTranslator(chatLLM llm.Provider, s *store.Store) *Translator {
	return &Translator{
		chatLLM: chatLLM,
		store:   s,
		cache:   make(map[string][]string),
	}
}

// Languages returns the corpus languages, reading from the DB on first call
// and caching thereafter. Returns nil if no languages are detected.
func (t *Translator) Languages() []string {
	t.mu.RLock()
	if t.langs != nil {
		defer t.mu.RUnlock()
		return t.langs
	}
	t.mu.RUnlock()

	t.mu.Lock()
	defer t.mu.Unlock()

	// Double-check after acquiring write lock.
	if t.langs != nil {
		return t.langs
	}

	if t.store == nil {
		return nil
	}

	langs, err := t.store.GetCorpusLanguages(context.Background())
	if err != nil {
		slog.Warn("translator: failed to get corpus languages", "error", err)
		return nil
	}
	if len(langs) == 0 {
		// Store empty slice so we don't retry.
		t.langs = []string{}
		return nil
	}
	t.langs = langs
	return t.langs
}

// TranslateTerms translates English query terms to all non-English corpus
// languages. Returns additional terms (translated forms) to append to search
// queries. Returns nil if all docs are English, no languages are detected,
// or chatLLM is nil.
func (t *Translator) TranslateTerms(ctx context.Context, terms []string) []string {
	if t.chatLLM == nil || len(terms) == 0 {
		return nil
	}

	langs := t.Languages()

	// Filter out English — query is assumed English
	var targetLangs []string
	for _, l := range langs {
		if !strings.EqualFold(l, "English") {
			targetLangs = append(targetLangs, l)
		}
	}
	if len(targetLangs) == 0 {
		return nil
	}

	// Deduplicate and check cache
	t.mu.RLock()
	var uncached []string
	var result []string
	seen := make(map[string]bool)
	for _, term := range terms {
		lower := strings.ToLower(term)
		if seen[lower] || len(lower) < 2 {
			continue
		}
		seen[lower] = true
		if cached, ok := t.cache[lower]; ok {
			result = append(result, cached...)
		} else {
			uncached = append(uncached, lower)
		}
	}
	t.mu.RUnlock()

	if len(uncached) == 0 {
		return result
	}

	// Batch translate via LLM to all target languages
	translated := t.llmTranslateMulti(ctx, uncached, targetLangs)
	for _, term := range uncached {
		if forms, ok := translated[term]; ok {
			result = append(result, forms...)
		}
	}

	return result
}

// llmTranslateMulti sends a batch of terms to the LLM for translation into
// multiple target languages and caches the results. Each term maps to an
// array of all translated forms across all target languages.
func (t *Translator) llmTranslateMulti(ctx context.Context, terms []string, targetLangs []string) map[string][]string {
	langList := strings.Join(targetLangs, ", ")
	prompt := fmt.Sprintf(
		`Translate these English technical terms to %s. For each term provide the singular and plural forms in each target language.

Return ONLY a JSON object where keys are the English terms (lowercase) and values are objects mapping language names to arrays of translated forms.

Example for [Spanish, Portuguese]:
{"noise": {"Spanish": ["ruido", "ruidos"], "Portuguese": ["ruído", "ruídos"]}, "valve": {"Spanish": ["válvula", "válvulas"], "Portuguese": ["válvula", "válvulas"]}}

If a term is the same in both languages, include it anyway.
If a term has multiple valid translations, include all of them.
For ambiguous terms with distinct meanings in different domains, include the top 2-3 most common translations (e.g., "cap" → both "tapa" and "mayúscula" in Spanish).

Terms: %s`, langList, strings.Join(terms, ", "))

	resp, err := t.chatLLM.Chat(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a technical translator. Return only valid JSON. No markdown fences, no explanation."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0,
		MaxTokens:   2048,
	})
	if err != nil {
		slog.Warn("translator: LLM translation failed", "error", err, "terms", len(terms))
		t.cacheEmpty(terms)
		return nil
	}

	// Parse JSON — strip thinking blocks and markdown fences
	content := stripThinking(strings.TrimSpace(resp.Content))
	if idx := strings.Index(content, "{"); idx >= 0 {
		content = content[idx:]
	}
	if idx := strings.LastIndex(content, "}"); idx >= 0 {
		content = content[:idx+1]
	}

	// Try multi-language format first: {"term": {"Lang": [...]}}
	var multiParsed map[string]map[string][]string
	if err := json.Unmarshal([]byte(content), &multiParsed); err == nil {
		// Flatten to term → all forms
		flat := make(map[string][]string, len(multiParsed))
		for term, langMap := range multiParsed {
			var allForms []string
			for _, forms := range langMap {
				allForms = append(allForms, forms...)
			}
			flat[term] = allForms
		}

		// Cache results
		t.mu.Lock()
		for _, term := range terms {
			if forms, ok := flat[term]; ok && len(forms) > 0 {
				t.cache[term] = forms
			} else {
				t.cache[term] = nil
			}
		}
		t.mu.Unlock()

		slog.Debug("translator: translated terms (multi-lang)",
			"requested", len(terms), "returned", len(flat), "langs", langList)
		return flat
	}

	// Fallback: try simple format {"term": [...]} (single target language)
	var simpleParsed map[string][]string
	if err := json.Unmarshal([]byte(content), &simpleParsed); err != nil {
		slog.Warn("translator: failed to parse translation JSON",
			"error", err, "content_len", len(content))
		t.cacheEmpty(terms)
		return nil
	}

	// Cache results
	t.mu.Lock()
	for _, term := range terms {
		if forms, ok := simpleParsed[term]; ok && len(forms) > 0 {
			t.cache[term] = forms
		} else {
			t.cache[term] = nil
		}
	}
	t.mu.Unlock()

	slog.Debug("translator: translated terms (simple)",
		"requested", len(terms), "returned", len(simpleParsed), "langs", langList)
	return simpleParsed
}

// cacheEmpty records nil for each term so we don't retry failed translations.
func (t *Translator) cacheEmpty(terms []string) {
	t.mu.Lock()
	for _, term := range terms {
		t.cache[term] = nil
	}
	t.mu.Unlock()
}

// stripThinking removes <think>...</think> blocks from LLM output.
// Some models (e.g. Qwen3) wrap reasoning in these tags.
func stripThinking(s string) string {
	for {
		start := strings.Index(s, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(s, "</think>")
		if end == -1 {
			// Unclosed tag — strip from <think> onward
			s = s[:start]
			break
		}
		s = s[:start] + s[end+len("</think>"):]
	}
	return strings.TrimSpace(s)
}
