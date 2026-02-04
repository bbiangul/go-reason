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

// Translator provides cross-language query expansion by detecting the document
// language and translating query terms via an LLM at runtime. Results are
// cached in memory so each unique term is only translated once per engine
// lifetime. This replaces the previous static bilingual dictionary and works
// with any document language.
type Translator struct {
	chatLLM llm.Provider
	store   *store.Store

	mu       sync.RWMutex
	lang     string              // detected document language (e.g. "Spanish")
	langOnce sync.Once           // ensures language detection runs at most once
	cache    map[string][]string // lowercase English term → translated forms
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

// Language returns the detected document language, triggering detection if it
// hasn't happened yet. Returns "" if detection fails or chatLLM is nil.
func (t *Translator) Language() string {
	return t.lang
}

// DetectLanguage samples chunk content and asks the LLM to identify the
// document language. Safe to call multiple times; the LLM call happens only
// once.
func (t *Translator) DetectLanguage(ctx context.Context) string {
	t.langOnce.Do(func() {
		if t.chatLLM == nil || t.store == nil {
			return
		}
		samples, err := t.store.SampleChunks(ctx, 5)
		if err != nil || len(samples) == 0 {
			slog.Warn("translator: cannot sample chunks for language detection", "error", err)
			return
		}

		var buf strings.Builder
		for i, c := range samples {
			if i > 0 {
				buf.WriteString("\n---\n")
			}
			text := c.Content
			if len(text) > 500 {
				text = text[:500]
			}
			buf.WriteString(text)
		}

		resp, err := t.chatLLM.Chat(ctx, llm.ChatRequest{
			Messages: []llm.Message{
				{Role: "system", Content: "You are a language detection assistant. Respond with ONLY the language name in English (e.g. 'Spanish', 'Portuguese', 'French', 'English'). Nothing else."},
				{Role: "user", Content: "What language is this text written in?\n\n" + buf.String()},
			},
			Temperature: 0,
			MaxTokens:   20,
		})
		if err != nil {
			slog.Warn("translator: language detection failed", "error", err)
			return
		}

		raw := resp.Content
		t.lang = stripThinking(strings.TrimSpace(raw))
		// Strip any trailing punctuation the model might add
		t.lang = strings.TrimRight(t.lang, ".")
		// Take only the first line in case the model is verbose
		if idx := strings.IndexAny(t.lang, "\n\r"); idx > 0 {
			t.lang = strings.TrimSpace(t.lang[:idx])
		}

		// Fallback: if LLM returned empty (e.g. thinking model), detect
		// language heuristically from the chunk text.
		if t.lang == "" {
			t.lang = detectLanguageHeuristic(buf.String())
			slog.Info("translator: language detected via heuristic", "language", t.lang)
		} else {
			slog.Info("translator: language detected via LLM", "language", t.lang)
		}
	})
	return t.lang
}

// TranslateTerms translates English query terms to the document language.
// Returns additional terms (translated singular + plural forms) to append to
// search queries. Returns nil if the document is in English, language
// detection failed, or chatLLM is nil.
func (t *Translator) TranslateTerms(ctx context.Context, terms []string) []string {
	if t.chatLLM == nil || len(terms) == 0 {
		return nil
	}

	lang := t.DetectLanguage(ctx)
	if lang == "" || strings.EqualFold(lang, "English") {
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

	// Batch translate via LLM
	translated := t.llmTranslate(ctx, uncached, lang)
	for _, term := range uncached {
		if forms, ok := translated[term]; ok {
			result = append(result, forms...)
		}
	}

	return result
}

// llmTranslate sends a batch of terms to the LLM for translation and caches
// the results. Each term maps to an array of translated forms (singular,
// plural, and any synonyms).
func (t *Translator) llmTranslate(ctx context.Context, terms []string, lang string) map[string][]string {
	prompt := fmt.Sprintf(
		`Translate these English technical terms to %s. For each term provide the singular and plural forms in the target language.

Return ONLY a JSON object where keys are the English terms (lowercase) and values are arrays of all translated forms (singular first, then plural, then any common synonyms).

Example for Spanish:
{"noise": ["ruido", "ruidos"], "valve": ["válvula", "válvulas"]}

If a term is the same in both languages, include it anyway.
If a term has multiple valid translations, include all of them.

Terms: %s`, lang, strings.Join(terms, ", "))

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

	var parsed map[string][]string
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		slog.Warn("translator: failed to parse translation JSON",
			"error", err, "content_len", len(content))
		t.cacheEmpty(terms)
		return nil
	}

	// Cache results
	t.mu.Lock()
	for _, term := range terms {
		if forms, ok := parsed[term]; ok && len(forms) > 0 {
			t.cache[term] = forms
		} else {
			t.cache[term] = nil
		}
	}
	t.mu.Unlock()

	slog.Debug("translator: translated terms",
		"requested", len(terms), "returned", len(parsed), "lang", lang)
	return parsed
}

// cacheEmpty records nil for each term so we don't retry failed translations.
func (t *Translator) cacheEmpty(terms []string) {
	t.mu.Lock()
	for _, term := range terms {
		t.cache[term] = nil
	}
	t.mu.Unlock()
}

// detectLanguageHeuristic detects common languages from text content by
// counting characteristic words. Returns the detected language name or "".
func detectLanguageHeuristic(text string) string {
	lower := strings.ToLower(text)
	words := strings.Fields(lower)
	if len(words) == 0 {
		return ""
	}

	// Count occurrences of language-specific common words
	type langScore struct {
		name  string
		words []string
	}
	langs := []langScore{
		{"Spanish", []string{"de", "en", "la", "el", "del", "los", "las", "para", "por", "con", "que", "una", "como", "está", "más", "también", "según", "puede", "debe", "sobre"}},
		{"Portuguese", []string{"de", "em", "do", "da", "dos", "das", "para", "por", "com", "que", "uma", "como", "está", "mais", "também", "segundo", "pode", "deve", "sobre", "não"}},
		{"French", []string{"de", "le", "la", "les", "des", "du", "en", "pour", "par", "avec", "que", "une", "dans", "est", "plus", "aussi", "selon", "peut", "doit", "sur"}},
		{"German", []string{"der", "die", "das", "den", "dem", "des", "ein", "eine", "und", "ist", "für", "mit", "von", "auf", "nicht", "auch", "nach", "kann", "wird", "über"}},
		{"English", []string{"the", "and", "for", "with", "that", "this", "from", "are", "was", "has", "have", "been", "will", "should", "must", "can", "which", "when", "where", "would"}},
	}

	wordSet := make(map[string]int, len(words))
	for _, w := range words {
		wordSet[w]++
	}

	var bestLang string
	var bestScore float64
	for _, lang := range langs {
		var score float64
		for _, w := range lang.words {
			score += float64(wordSet[w])
		}
		// Normalize by total words to get frequency
		freq := score / float64(len(words))
		if freq > bestScore {
			bestScore = freq
			bestLang = lang.name
		}
	}

	// Disambiguate Spanish vs Portuguese (they share many words)
	// Spanish-specific: "el", "los", "las", "muy", "pero", "como"
	// Portuguese-specific: "não", "muito", "mas", "como", "foi"
	if bestLang == "Portuguese" || bestLang == "Spanish" {
		esOnly := 0
		ptOnly := 0
		for _, w := range []string{"el", "los", "las", "muy", "pero"} {
			esOnly += wordSet[w]
		}
		for _, w := range []string{"não", "muito", "mas", "foi", "são"} {
			ptOnly += wordSet[w]
		}
		if esOnly > ptOnly {
			bestLang = "Spanish"
		} else if ptOnly > esOnly {
			bestLang = "Portuguese"
		}
	}

	if bestScore < 0.01 { // less than 1% hit rate — unreliable
		return ""
	}
	return bestLang
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
