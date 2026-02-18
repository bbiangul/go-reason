package eval

import "testing"

func TestGDPRDatasetCounts(t *testing.T) {
	tests := []struct {
		name     string
		dataset  Dataset
		wantLen  int
		wantDiff string
	}{
		{"Easy", GDPREasyDataset(), 30, DifficultyEasy},
		{"Medium", GDPRMediumDataset(), 30, DifficultyMedium},
		{"Hard", GDPRHardDataset(), 30, DifficultyHard},
		{"SuperHard", GDPRSuperHardDataset(), 50, DifficultySuperHard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(tt.dataset.Tests); got != tt.wantLen {
				t.Errorf("got %d tests, want %d", got, tt.wantLen)
			}
			if tt.dataset.Difficulty != tt.wantDiff {
				t.Errorf("got difficulty %q, want %q", tt.dataset.Difficulty, tt.wantDiff)
			}
		})
	}
}

func TestGDPRDatasetNonEmpty(t *testing.T) {
	all := GDPRAllDatasets()
	if len(all) != 4 {
		t.Fatalf("got %d difficulty levels, want 4", len(all))
	}

	for diff, ds := range all {
		t.Run(diff, func(t *testing.T) {
			if ds.Name == "" {
				t.Error("dataset Name is empty")
			}
			for i, tc := range ds.Tests {
				if tc.Question == "" {
					t.Errorf("test %d: Question is empty", i)
				}
				if len(tc.ExpectedFacts) == 0 {
					t.Errorf("test %d: ExpectedFacts is empty for %q", i, tc.Question)
				}
				if tc.Category == "" {
					t.Errorf("test %d: Category is empty for %q", i, tc.Question)
				}
				if tc.Explanation == "" {
					t.Errorf("test %d: Explanation is empty for %q", i, tc.Question)
				}
			}
		})
	}
}

func TestGDPRDatasetCategories(t *testing.T) {
	validCategories := map[string]bool{
		"single-fact":  true,
		"multi-hop":    true,
		"synthesis":    true,
		"adversarial":  true,
	}

	all := GDPRAllDatasets()
	for diff, ds := range all {
		for i, tc := range ds.Tests {
			if !validCategories[tc.Category] {
				t.Errorf("[%s] test %d: invalid category %q for %q", diff, i, tc.Category, tc.Question)
			}
		}
	}
}

func TestGDPRTotalCount(t *testing.T) {
	total := 0
	for _, ds := range GDPRAllDatasets() {
		total += len(ds.Tests)
	}
	if total != 140 {
		t.Errorf("total test count = %d, want 140", total)
	}
}
