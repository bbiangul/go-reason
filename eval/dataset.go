package eval

// Difficulty levels for evaluation datasets.
const (
	DifficultyEasy      = "easy"
	DifficultyMedium    = "medium"
	DifficultyHard      = "hard"
	DifficultyComplex   = "complex"    // Used by legacy ComplexDataset(); not part of ALTAVision eval.
	DifficultySuperHard = "super-hard"
)

// Dataset is a collection of test cases for evaluation.
type Dataset struct {
	Name       string     `json:"name"`
	Difficulty string     `json:"difficulty"` // easy, medium, hard, complex, super-hard
	Tests      []TestCase `json:"tests"`
}

// TestCase defines a single evaluation question.
type TestCase struct {
	Question      string   `json:"question"`
	ExpectedFacts []string `json:"expected_facts"` // Facts that should appear in the answer
	Category      string   `json:"category"`        // single-fact, multi-hop, cross-document, multi-fact, synthesis
	Explanation   string   `json:"explanation"`      // Ground truth reference with page citations
}

// EasyDataset returns sample easy (single-fact) test cases.
func EasyDataset() Dataset {
	return Dataset{
		Name:       "Easy - Single Fact Lookup",
		Difficulty: "easy",
		Tests: []TestCase{
			{
				Question:      "What is the tensile strength specified in section 3.2?",
				ExpectedFacts: []string{"tensile strength", "section 3.2"},
				Category:      "single-fact",
			},
			{
				Question:      "Who is the responsible party listed in the agreement?",
				ExpectedFacts: []string{"responsible party"},
				Category:      "single-fact",
			},
			{
				Question:      "What is the effective date of the contract?",
				ExpectedFacts: []string{"effective date"},
				Category:      "single-fact",
			},
		},
	}
}

// MediumDataset returns sample medium (multi-hop) test cases.
func MediumDataset() Dataset {
	return Dataset{
		Name:       "Medium - Multi-hop Reasoning",
		Difficulty: "medium",
		Tests: []TestCase{
			{
				Question:      "Which clauses reference the force majeure definition?",
				ExpectedFacts: []string{"force majeure", "clause"},
				Category:      "multi-hop",
			},
			{
				Question:      "What requirements reference ISO 9001?",
				ExpectedFacts: []string{"ISO 9001", "requirement"},
				Category:      "multi-hop",
			},
			{
				Question:      "List all sections that define technical terms.",
				ExpectedFacts: []string{"definition", "technical"},
				Category:      "multi-hop",
			},
		},
	}
}

// ComplexDataset returns sample complex (cross-document) test cases.
func ComplexDataset() Dataset {
	return Dataset{
		Name:       "Complex - Cross-document Synthesis",
		Difficulty: "complex",
		Tests: []TestCase{
			{
				Question:      "Compare the liability provisions across the ingested contracts.",
				ExpectedFacts: []string{"liability", "provision"},
				Category:      "cross-document",
			},
			{
				Question:      "Which standards are referenced by multiple documents?",
				ExpectedFacts: []string{"standard", "reference"},
				Category:      "cross-document",
			},
			{
				Question:      "Summarize all termination clauses and their conditions.",
				ExpectedFacts: []string{"termination", "condition"},
				Category:      "cross-document",
			},
		},
	}
}
