package goreason

import "testing"

func TestKeywordFallback(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFound bool
	}{
		{
			name:      "clear answer",
			input:     "The maximum operating temperature is 85°C as specified in section 4.2.",
			wantFound: true,
		},
		{
			name:      "not found",
			input:     "The information was not found in the provided documents.",
			wantFound: false,
		},
		{
			name:      "not mentioned",
			input:     "This topic is not mentioned anywhere in the source material.",
			wantFound: false,
		},
		{
			name:      "insufficient information",
			input:     "There is insufficient information to answer this question.",
			wantFound: false,
		},
		{
			name:      "cannot determine",
			input:     "Based on the available data, I cannot determine the answer.",
			wantFound: false,
		},
		{
			name:      "no relevant data",
			input:     "There is no relevant information in the document.",
			wantFound: false,
		},
		{
			name:      "does not contain",
			input:     "The document does not contain any reference to this specification.",
			wantFound: false,
		},
		{
			name:      "unable to find",
			input:     "I was unable to find this information in the provided context.",
			wantFound: false,
		},
		{
			name:      "hedging with substance",
			input:     "The voltage rating appears to be 24VDC based on the specifications table.",
			wantFound: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantFound: true,
		},
		{
			name:      "case insensitive not found",
			input:     "NOT FOUND in the document.",
			wantFound: false,
		},
		{
			name:      "does not provide",
			input:     "The document does not provide details on this topic.",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := keywordFallback(tt.input)
			if result.Found != tt.wantFound {
				t.Errorf("keywordFallback(%q): got Found=%v, want %v", tt.input, result.Found, tt.wantFound)
			}
			if result.Response != tt.input {
				t.Errorf("keywordFallback(%q): got Response=%q, want %q", tt.input, result.Response, tt.input)
			}
		})
	}
}
