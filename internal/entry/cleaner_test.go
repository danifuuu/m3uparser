package entry

import (
	"testing"
)

func TestProcessValueReplace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		replace  map[string]string
		expected string
	}{
		{
			name:     "simple replacement",
			input:    "Hello World",
			replace:  map[string]string{"World": "Go"},
			expected: "Hello Go",
		},
		{
			name:     "multiple replacements",
			input:    "1/2 cup / serving",
			replace:  map[string]string{"1/2": "\u00BD", "/": "-"},
			expected: "\u00BD cup - serving",
		},
		{
			name:     "no match",
			input:    "Hello World",
			replace:  map[string]string{"Foo": "Bar"},
			expected: "Hello World",
		},
		{
			name:     "nil replace",
			input:    "Hello World",
			replace:  nil,
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProcessValue(tt.input, tt.replace, nil)
			if got != tt.expected {
				t.Errorf("ProcessValue(%q, replace) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestProcessValueRemoveHeader(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		removeHeader []string
		expected     string
	}{
		{
			name:         "remove HD prefix",
			input:        "HD : Breaking Bad S01E01",
			removeHeader: []string{"HD :"},
			expected:     "Breaking Bad S01E01",
		},
		{
			name:         "remove SD prefix",
			input:        "SD : The Matrix 1999",
			removeHeader: []string{"SD :"},
			expected:     "The Matrix 1999",
		},
		{
			name:         "no match",
			input:        "Breaking Bad S01E01",
			removeHeader: []string{"HD :"},
			expected:     "Breaking Bad S01E01",
		},
		{
			name:         "nil headers",
			input:        "Hello",
			removeHeader: nil,
			expected:     "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProcessValue(tt.input, nil, tt.removeHeader)
			if got != tt.expected {
				t.Errorf("ProcessValue(%q, header) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRemoveAllTerms(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		terms    []string
		expected string
	}{
		{
			name:     "remove quality terms",
			input:    "The Office 720p HDTV x264",
			terms:    []string{"720p", "HDTV", "x264"},
			expected: "The Office",
		},
		{
			name:     "no matching terms",
			input:    "Hello World",
			terms:    []string{"Foo"},
			expected: "Hello World",
		},
		{
			name:     "empty terms",
			input:    "Hello World",
			terms:    nil,
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveAllTerms(tt.input, tt.terms)
			if got != tt.expected {
				t.Errorf("RemoveAllTerms(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCheckExcluded(t *testing.T) {
	tests := []struct {
		name       string
		groupTitle string
		terms      []string
		expected   bool
	}{
		{
			name:       "matches exclude",
			groupTitle: "Adult Content XXX",
			terms:      []string{"XXX", "NSFW"},
			expected:   true,
		},
		{
			name:       "case insensitive",
			groupTitle: "some xxx channel",
			terms:      []string{"XXX"},
			expected:   true,
		},
		{
			name:       "no match",
			groupTitle: "CNN News",
			terms:      []string{"XXX", "NSFW"},
			expected:   false,
		},
		{
			name:       "empty terms",
			groupTitle: "CNN News",
			terms:      nil,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckExcluded(tt.groupTitle, tt.terms)
			if got != tt.expected {
				t.Errorf("CheckExcluded(%q, %v) = %v, want %v", tt.groupTitle, tt.terms, got, tt.expected)
			}
		})
	}
}
