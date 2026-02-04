package reasoning

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bbiangul/go-reason/llm"
	"github.com/bbiangul/go-reason/store"
)

// Config holds reasoning engine configuration.
type Config struct {
	MaxRounds           int
	ConfidenceThreshold float64
}

// Options configures a single reasoning operation.
type Options struct {
	MaxRounds int
}

// Answer is the final output of the reasoning pipeline.
type Answer struct {
	Text             string   `json:"text"`
	Confidence       float64  `json:"confidence"`
	Sources          []Source `json:"sources"`
	Reasoning        []Step   `json:"reasoning"`
	ModelUsed        string   `json:"model_used"`
	Rounds           int      `json:"rounds"`
	PromptTokens     int      `json:"prompt_tokens"`
	CompletionTokens int      `json:"completion_tokens"`
	TotalTokens      int      `json:"total_tokens"`
}

// Source tracks a chunk used in the answer.
type Source struct {
	ChunkID    int64   `json:"chunk_id"`
	DocumentID int64   `json:"document_id"`
	Filename   string  `json:"filename"`
	Content    string  `json:"content"`
	Heading    string  `json:"heading"`
	PageNumber int     `json:"page_number"`
	Score      float64 `json:"score"`
}

// Step records a single round of the reasoning pipeline.
type Step struct {
	Round      int      `json:"round"`
	Action     string   `json:"action"`
	Input      string   `json:"input,omitempty"`
	Output     string   `json:"output,omitempty"`
	Prompt     string   `json:"prompt,omitempty"`     // full prompt sent to LLM (for replay)
	Response   string   `json:"response,omitempty"`   // raw LLM response
	Validation string   `json:"validation,omitempty"`
	ChunksUsed int      `json:"chunks_used,omitempty"`
	Tokens     int      `json:"tokens,omitempty"`
	ElapsedMs  int64    `json:"elapsed_ms,omitempty"`
	Issues     []string `json:"issues,omitempty"` // validation issues found
}

// Engine runs multi-round reasoning with validation between rounds.
type Engine struct {
	chat llm.Provider
	cfg  Config
}

// New creates a new reasoning engine.
func New(chat llm.Provider, cfg Config) *Engine {
	if cfg.MaxRounds == 0 {
		cfg.MaxRounds = 3
	}
	if cfg.ConfidenceThreshold == 0 {
		cfg.ConfidenceThreshold = 0.7
	}
	return &Engine{chat: chat, cfg: cfg}
}

// Reason runs the multi-round reasoning pipeline:
// Round 1: Generate initial answer from retrieved context
// Round 2: Validate citations and check for gaps
// Round 3: If confidence < threshold, refine and re-answer
func (e *Engine) Reason(ctx context.Context, question string, chunks []store.RetrievalResult, opts Options) (*Answer, error) {
	maxRounds := opts.MaxRounds
	if maxRounds == 0 {
		maxRounds = e.cfg.MaxRounds
	}

	sources := make([]Source, len(chunks))
	for i, c := range chunks {
		sources[i] = Source{
			ChunkID:    c.ChunkID,
			DocumentID: c.DocumentID,
			Filename:   c.Filename,
			Content:    c.Content,
			Heading:    c.Heading,
			PageNumber: c.PageNumber,
			Score:      c.Score,
		}
	}

	var steps []Step
	var currentAnswer string
	var confidence float64
	var modelUsed string
	var promptTokens, completionTokens, totalTokens int

	// Round 1: Initial answer generation
	slog.Info("reasoning: round 1 starting", "question_len", len(question), "chunks", len(chunks))
	round1Start := time.Now()
	contextStr := buildContext(chunks)
	initialPrompt := buildAnswerPrompt(question, contextStr)

	resp, err := e.chat.Chat(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: initialPrompt},
		},
		Temperature: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("round 1 generation: %w", err)
	}
	round1Elapsed := time.Since(round1Start)
	slog.Info("reasoning: round 1 complete",
		"tokens", resp.TotalTokens, "elapsed", round1Elapsed.Round(time.Millisecond))

	currentAnswer = resp.Content
	modelUsed = resp.Model
	promptTokens += resp.PromptTokens
	completionTokens += resp.CompletionTokens
	totalTokens += resp.TotalTokens
	steps = append(steps, Step{
		Round:      1,
		Action:     "initial_answer",
		Input:      question,
		Output:     currentAnswer,
		Prompt:     initialPrompt,
		Response:   resp.Content,
		ChunksUsed: len(chunks),
		Tokens:     resp.TotalTokens,
		ElapsedMs:  round1Elapsed.Milliseconds(),
	})

	if maxRounds < 2 {
		confidence = estimateConfidence(currentAnswer, chunks)
		return &Answer{
			Text:             currentAnswer,
			Confidence:       confidence,
			Sources:          sources,
			Reasoning:        steps,
			ModelUsed:        modelUsed,
			Rounds:           1,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		}, nil
	}

	// Round 2: Validation
	validation := validate(currentAnswer, chunks)
	var validationIssues []string
	validationIssues = append(validationIssues, validation.citationIssues...)
	validationIssues = append(validationIssues, validation.consistencyIssues...)
	validationIssues = append(validationIssues, validation.completenessIssues...)
	steps = append(steps, Step{
		Round:      2,
		Action:     "validation",
		Input:      currentAnswer,
		Output:     validation.summary(),
		Validation: validation.summary(),
		Issues:     validationIssues,
	})

	confidence = validation.confidence()

	// Round 3: Refinement if needed
	if maxRounds >= 3 && confidence < e.cfg.ConfidenceThreshold {
		slog.Info("reasoning: round 3 starting (confidence below threshold)",
			"confidence", fmt.Sprintf("%.2f", confidence),
			"threshold", fmt.Sprintf("%.2f", e.cfg.ConfidenceThreshold))
		round3Start := time.Now()
		refinementPrompt := buildRefinementPrompt(question, currentAnswer, contextStr, validation)

		resp, err = e.chat.Chat(ctx, llm.ChatRequest{
			Messages: []llm.Message{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: refinementPrompt},
			},
			Temperature: 0,
		})
		if err != nil {
			// Non-fatal: return the answer from round 1
			return &Answer{
				Text:             currentAnswer,
				Confidence:       confidence,
				Sources:          sources,
				Reasoning:        steps,
				ModelUsed:        modelUsed,
				Rounds:           2,
				PromptTokens:     promptTokens,
				CompletionTokens: completionTokens,
				TotalTokens:      totalTokens,
			}, nil
		}

		round3Elapsed := time.Since(round3Start)
		currentAnswer = resp.Content
		promptTokens += resp.PromptTokens
		completionTokens += resp.CompletionTokens
		totalTokens += resp.TotalTokens
		steps = append(steps, Step{
			Round:      3,
			Action:     "refinement",
			Input:      validation.summary(),
			Output:     currentAnswer,
			Prompt:     refinementPrompt,
			Response:   resp.Content,
			ChunksUsed: len(chunks),
			Tokens:     resp.TotalTokens,
			ElapsedMs:  round3Elapsed.Milliseconds(),
		})

		slog.Info("reasoning: round 3 complete",
			"tokens", resp.TotalTokens, "elapsed", round3Elapsed.Round(time.Millisecond))

		// Re-validate
		validation = validate(currentAnswer, chunks)
		confidence = validation.confidence()
	}

	return &Answer{
		Text:             currentAnswer,
		Confidence:       confidence,
		Sources:          sources,
		Reasoning:        steps,
		ModelUsed:        modelUsed,
		Rounds:           len(steps),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
	}, nil
}

const systemPrompt = `You are a precise document analysis assistant. Answer questions based ONLY on the provided context.
Rules:
1. Only state facts that are directly supported by the provided sources.
2. Cite sources by referencing the document filename and section/page when possible.
3. If the context doesn't contain enough information to answer, say so explicitly.
4. For legal and engineering documents, preserve exact terminology and clause references.
5. Be concise but thorough.`

func buildContext(chunks []store.RetrievalResult) string {
	var b strings.Builder
	for i, c := range chunks {
		fmt.Fprintf(&b, "--- Source %d: %s", i+1, c.Filename)
		if c.Heading != "" {
			fmt.Fprintf(&b, " | %s", c.Heading)
		}
		if c.PageNumber > 0 {
			fmt.Fprintf(&b, " | Page %d", c.PageNumber)
		}
		b.WriteString(" ---\n")
		b.WriteString(c.Content)
		b.WriteString("\n\n")
	}
	return b.String()
}

func buildAnswerPrompt(question, context string) string {
	return fmt.Sprintf(`Context:
%s

Question: %s

Provide a detailed answer based only on the context above. Cite specific sources.`, context, question)
}

func buildRefinementPrompt(question, previousAnswer, context string, v *validationResult) string {
	return fmt.Sprintf(`Context:
%s

Question: %s

Previous answer:
%s

Issues found during validation:
%s

Please provide an improved answer that addresses the validation issues. Ensure all claims are properly cited from the context.`, context, question, previousAnswer, v.summary())
}

func estimateConfidence(answer string, chunks []store.RetrievalResult) float64 {
	if answer == "" || len(chunks) == 0 {
		return 0.0
	}

	score := 0.5 // base score

	// Higher confidence if answer references specific sources
	lowerAnswer := strings.ToLower(answer)
	sourceRefs := 0
	for _, c := range chunks {
		if strings.Contains(lowerAnswer, strings.ToLower(c.Filename)) {
			sourceRefs++
		}
		if c.Heading != "" && strings.Contains(lowerAnswer, strings.ToLower(c.Heading)) {
			sourceRefs++
		}
	}
	if sourceRefs > 0 {
		score += 0.2 * float64(min(sourceRefs, 3)) / 3.0
	}

	// Lower confidence for hedging language
	hedges := []string{"might", "possibly", "unclear", "not enough information", "cannot determine"}
	for _, h := range hedges {
		if strings.Contains(lowerAnswer, h) {
			score -= 0.1
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
